package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/entitlement/domain"
	featureDomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	planDomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	planfeaturedomain "github.com/railzwaylabs/railzway/internal/planfeature/domain"
	pfDomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	subDomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"go.uber.org/fx"
)

type service struct {
	subscription subDomain.Service
	plan         planDomain.Service
	feature      featureDomain.Service
	productfeat  pfDomain.Service
	planfeat     planfeaturedomain.Service
}

type Params struct {
	fx.In

	Subscription subDomain.Service
	Plan         planDomain.Service
	Feature      featureDomain.Service
	ProductFeat  pfDomain.Service
	PlanFeat     planfeaturedomain.Service `optional:"true"`
}

func NewService(p Params) domain.Service {
	return &service{
		subscription: p.Subscription,
		plan:         p.Plan,
		feature:      p.Feature,
		productfeat:  p.ProductFeat,
		planfeat:     p.PlanFeat,
	}
}

func (s *service) CheckEntitlement(ctx context.Context, req domain.CheckEntitlementRequest) (domain.EntitlementResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.EntitlementResponse{}, domain.ErrInvalidOrganization
	}

	customerID := strings.TrimSpace(req.CustomerID)
	if customerID == "" {
		return domain.EntitlementResponse{}, domain.ErrInvalidCustomer
	}

	featureCode := strings.TrimSpace(req.FeatureCode)
	if featureCode == "" {
		return domain.EntitlementResponse{}, domain.ErrInvalidFeature
	}

	subResp, err := s.subscription.ListSubscriptions(ctx, subDomain.ListSubscriptionRequest{
		CustomerID: customerID,
		Status:     subDomain.StatusActive,
		PageSize:   10,
	})
	if err != nil {
		return domain.EntitlementResponse{}, err
	}
	if len(subResp.Subscriptions) == 0 {
		return domain.EntitlementResponse{HasAccess: false}, nil
	}

	// For simplicity in MVP, we take the first active subscription.
	sub := subResp.Subscriptions[0]

	planResp, err := s.plan.GetPlanByID(ctx, planDomain.GetPlanRequest{ID: sub.PlanID})
	if err != nil {
		return domain.EntitlementResponse{}, err
	}

	if planResp.ProductID == nil {
		return domain.EntitlementResponse{HasAccess: false}, nil
	}

	productID := *planResp.ProductID

	if s.planfeat != nil {
		planFeatures, err := s.planfeat.ListForPlans(ctx, planfeaturedomain.ListForPlansRequest{
			PlanIDs: []string{sub.PlanID},
		})
		if err != nil {
			return domain.EntitlementResponse{}, err
		}
		for _, f := range planFeatures {
			if f.Code != featureCode || !f.Active {
				continue
			}
			resp := domain.EntitlementResponse{
				HasAccess: f.Enabled,
				IsMetered: f.FeatureType == "metered",
			}
			if f.LimitNumeric != nil {
				resp.Limit = *f.LimitNumeric
			}
			return resp, nil
		}
	}

	pfResp, err := s.productfeat.ListForProducts(ctx, pfDomain.ListForProductsRequest{
		ProductIDs: []string{productID},
	})
	if err != nil {
		return domain.EntitlementResponse{}, err
	}

	var foundFeature *pfDomain.Snapshot
	for i, f := range pfResp {
		if f.Code == featureCode && f.Active {
			foundFeature = &pfResp[i]
			break
		}
	}

	if foundFeature == nil {
		return domain.EntitlementResponse{HasAccess: false}, nil
	}

	return domain.EntitlementResponse{
		HasAccess: true,
		IsMetered: foundFeature.FeatureType == "metered",
	}, nil
}
