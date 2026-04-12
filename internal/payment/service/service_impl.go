package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
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

func (s *service) ListPayments(ctx context.Context, req domain.ListPaymentsRequest) (domain.ListPaymentsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListPaymentsResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.PaymentListFilter{
		Status:   strings.ToLower(strings.TrimSpace(req.Status)),
		Provider: strings.TrimSpace(req.Provider),
	}
	if req.CustomerID != "" {
		id, err := parseUUID(req.CustomerID, domain.ErrInvalidCustomer)
		if err != nil {
			return domain.ListPaymentsResponse{}, err
		}
		filter.CustomerID = id
	}
	if req.InvoiceID != "" {
		id, err := parseUUID(req.InvoiceID, domain.ErrInvalidInvoice)
		if err != nil {
			return domain.ListPaymentsResponse{}, err
		}
		filter.InvoiceID = id
	}
	if filter.Status != "" && !isValidStatus(filter.Status) {
		return domain.ListPaymentsResponse{}, domain.ErrInvalidStatus
	}
	filter.CreatedFrom = req.CreatedFrom
	filter.CreatedTo = req.CreatedTo

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListPaymentsResponse{}, err
	}

	items, err := s.repo.ListPayments(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListPaymentsResponse{}, err
	}

	resp := domain.ListPaymentsResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Payment) string {
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

	resp.Payments = make([]domain.Payment, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Payments = append(resp.Payments, *item)
	}

	return resp, nil
}

func parseUUID(value string, invalidErr error) (uuid.UUID, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return uuid.Nil, invalidErr
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, invalidErr
	}
	return id, nil
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

func isValidStatus(value string) bool {
	switch value {
	case domain.StatusPending, domain.StatusSucceeded, domain.StatusFailed, domain.StatusRefunded:
		return true
	default:
		return false
	}
}
