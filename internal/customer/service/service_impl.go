package service

import (
	"context"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/customer/domain"
	"github.com/smallbiznis/railzway/internal/orgcontext"
	quotadomain "github.com/smallbiznis/railzway/internal/quota/domain"
	"github.com/smallbiznis/railzway/pkg/db/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Params struct {
	fx.In

	DB       *gorm.DB
	Log      *zap.Logger
	GenID    *snowflake.Node
	Repo     domain.Repository
	QuotaSvc quotadomain.Service
}

type Service struct {
	db       *gorm.DB
	log      *zap.Logger
	genID    *snowflake.Node
	repo     domain.Repository
	quotaSvc quotadomain.Service
}

func New(p Params) domain.Service {
	return &Service{
		db:       p.DB,
		log:      p.Log.Named("customer.service"),
		genID:    p.GenID,
		repo:     p.Repo,
		quotaSvc: p.QuotaSvc,
	}
}

func (s *Service) Create(ctx context.Context, req domain.CreateCustomerRequest) (domain.Customer, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.Customer{}, domain.ErrInvalidOrganization
	}

	// Quota Check
	if err := s.quotaSvc.CanCreateCustomer(ctx, orgID); err != nil {
		return domain.Customer{}, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.Customer{}, domain.ErrInvalidName
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		return domain.Customer{}, domain.ErrInvalidEmail
	}

	now := time.Now().UTC()
	customer := domain.Customer{
		ID:        s.genID.Generate(),
		OrgID:     orgID,
		Name:      name,
		Email:     email,
		Metadata:  datatypes.JSONMap{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Insert(ctx, s.db, &customer); err != nil {
		return domain.Customer{}, err
	}

	return customer, nil
}

func (s *Service) List(ctx context.Context, req domain.ListCustomerRequest) (domain.ListCustomerResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.ListCustomerResponse{}, domain.ErrInvalidOrganization
	}

	filter := domain.ListCustomerFilter{
		Name:        strings.TrimSpace(req.Name),
		Email:       strings.TrimSpace(req.Email),
		Currency:    strings.TrimSpace(req.Currency),
		CreatedFrom: req.CreatedFrom,
		CreatedTo:   req.CreatedTo,
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	items, err := s.repo.List(ctx, s.db, orgID, filter, pagination.Pagination{
		PageToken: req.PageToken,
		PageSize:  int(pageSize),
	})
	if err != nil {
		return domain.ListCustomerResponse{}, err
	}

	pageInfo := pagination.BuildCursorPageInfo(items, pageSize, func(customer *domain.Customer) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        customer.ID.String(),
			CreatedAt: customer.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil && pageInfo.HasMore && len(items) > int(pageSize) {
		items = items[:pageSize]
	}

	customers := make([]domain.Customer, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		customers = append(customers, *item)
	}

	resp := domain.ListCustomerResponse{Customers: customers}
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
	}

	return resp, nil
}

func (s *Service) GetByID(ctx context.Context, req domain.GetCustomerRequest) (domain.Customer, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.Customer{}, domain.ErrInvalidOrganization
	}

	id, err := s.parseID(req.ID)
	if err != nil {
		return domain.Customer{}, err
	}

	item, err := s.repo.FindByID(ctx, s.db, orgID, id)
	if err != nil {
		return domain.Customer{}, err
	}
	if item == nil {
		return domain.Customer{}, domain.ErrNotFound
	}

	return *item, nil
}

func (s *Service) parseID(value string) (snowflake.ID, error) {
	id, err := snowflake.ParseString(strings.TrimSpace(value))
	if err != nil || id == 0 {
		return 0, domain.ErrInvalidID
	}
	return id, nil
}
