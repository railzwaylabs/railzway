package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/db"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/tax/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type service struct {
	db   *gorm.DB
	repo domain.Repository
}

type Params struct {
	fx.In

	DB   *gorm.DB
	Repo domain.Repository
}

func NewService(p Params) domain.Service {
	return &service{
		db:   p.DB,
		repo: p.Repo,
	}
}

func (s *service) ListTaxRates(ctx context.Context, req domain.ListTaxRatesRequest) (domain.ListTaxRatesResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListTaxRatesResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.TaxRateListFilter{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		Active:      req.Active,
		CreatedFrom: req.CreatedFrom,
		CreatedTo:   req.CreatedTo,
	}

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListTaxRatesResponse{}, err
	}

	items, err := s.repo.ListTaxRates(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListTaxRatesResponse{}, err
	}

	resp := domain.ListTaxRatesResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.TaxRate) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(items) > pageSize {
			items = items[:pageSize]
		}
	}

	resp.Rates = make([]domain.TaxRate, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Rates = append(resp.Rates, *item)
	}

	return resp, nil
}

func (s *service) CreateTaxRate(ctx context.Context, req domain.CreateTaxRateRequest) (domain.TaxRate, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TaxRate{}, domain.ErrInvalidOrganization
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return domain.TaxRate{}, domain.ErrInvalidCode
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.TaxRate{}, domain.ErrInvalidName
	}
	if req.Percentage < 0 {
		return domain.TaxRate{}, domain.ErrInvalidPercentage
	}

	metadata := req.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}

	now := time.Now().UTC()
	rate := domain.TaxRate{
		ID:         uuid.New(),
		OrgID:      orgID,
		Code:       code,
		Name:       name,
		Percentage: req.Percentage,
		Inclusive:  req.Inclusive,
		Active:     req.Active,
		Metadata:   metadata,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.CreateTaxRate(ctx, &rate); err != nil {
		if db.IsDuplicateKeyErr(err) {
			return domain.TaxRate{}, domain.ErrTaxCodeExists
		}
		return domain.TaxRate{}, err
	}

	return rate, nil
}

func decodeCursor(token string) (*domain.ListCursor, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil
	}
	decoded, err := pagination.DecodeCursor(raw)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}
	if decoded == nil || decoded.ID == "" || decoded.CreatedAt == "" {
		return nil, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}
	parsedID, err := uuid.Parse(decoded.ID)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}
	return &domain.ListCursor{
		ID:        parsedID,
		CreatedAt: parsedTime,
	}, nil
}
