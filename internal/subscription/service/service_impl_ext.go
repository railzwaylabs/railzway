package service

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	pricedomain "github.com/railzwaylabs/railzway/internal/price/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"gorm.io/gorm"
)

func (s *Service) ChangePlan(ctx context.Context, req subscriptiondomain.ChangePlanRequest) error {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		return subscriptiondomain.ErrInvalidOrganization
	}
	subscriptionID, err := snowflake.ParseString(req.SubscriptionID)
	if err != nil {
		return subscriptiondomain.ErrInvalidSubscription
	}
	newProductID, err := snowflake.ParseString(req.NewProductID)
	if err != nil {
		return subscriptiondomain.ErrInvalidProduct
	}

	if !isValidStatus(subscriptiondomain.SubscriptionStatusActive) {
		return subscriptiondomain.ErrInvalidSubscriptionStatus
	}

	now := s.clock.Now().UTC()

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Fetch Subscription
		sub, err := s.repo.FindByIDForUpdate(ctx, tx, orgID, subscriptionID)
		if err != nil {
			return err
		}
		if sub == nil {
			return subscriptiondomain.ErrSubscriptionNotFound
		}
		if sub.Status != subscriptiondomain.SubscriptionStatusActive {
			return subscriptiondomain.ErrInvalidSubscriptionStatus
		}

		// 2. Resolve Price for New Product
		// We need to find the default active price for the product.
		newPrice, err := s.resolveProductPrice(ctx, orgID, newProductID)
		if err != nil {
			return err
		}

		// 3. Build New Items and Entitlements
		// Re-calculate based on new product features and price
		cycleType, err := billingCycleTypeForInterval(newPrice.BillingInterval)
		if err != nil {
			return err
		}

		// Construct request item (assuming quantity 1 for plan change for offered product)
		itemReqs := []subscriptiondomain.CreateSubscriptionItemRequest{
			{
				PriceID:  newPrice.ID.String(),
				Quantity: 1,
			},
		}

		// buildSubscriptionItems does not take tx, uses service db/cache, which is safe for read-only static data (prices/meters)
		subscriptionItems, _, err := s.buildSubscriptionItems(ctx, orgID, subscriptionID, itemReqs, cycleType, now)
		if err != nil {
			return err
		}

		entitlements, err := s.buildSubscriptionEntitlements(ctx, tx, orgID, subscriptionID, []snowflake.ID{newProductID}, now)
		if err != nil {
			return err
		}

		// 4. Update Database

		// Close old entitlements
		if err := s.closeActiveEntitlements(ctx, tx, subscriptionID, now); err != nil {
			return err
		}

		// Replace items
		if err := s.repo.ReplaceItems(ctx, tx, orgID, subscriptionID, subscriptionItems); err != nil {
			return err
		}

		// Insert new entitlements
		if err := s.repo.InsertEntitlements(ctx, tx, entitlements); err != nil {
			return err
		}

		// Update subscription
		newCycleType, err := billingCycleTypeForInterval(newPrice.BillingInterval)
		if err != nil {
			return err
		}

		// Optimization: Only update if changed, but PlanChangedAt must strictly update
		if err := tx.Exec(
			`UPDATE subscriptions
             SET plan_changed_at = ?, billing_cycle_type = ?, updated_at = ?
             WHERE org_id = ? AND id = ?`,
			now,
			newCycleType,
			now,
			orgID,
			subscriptionID,
		).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) resolveProductPrice(ctx context.Context, orgID, productID snowflake.ID) (*pricedomain.Response, error) {
	allPrices, err := s.pricesvc.List(ctx, pricedomain.ListOptions{
		ProductID: productID.String(),
	})
	if err != nil {
		return nil, err
	}

	var candidates []pricedomain.Response
	for _, p := range allPrices {
		if p.ProductID == productID && p.Active && p.RetiredAt == nil {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return nil, subscriptiondomain.ErrMissingPricing
	}

	if len(candidates) == 1 {
		return &candidates[0], nil
	}

	// Filter for default
	var defaults []pricedomain.Response
	for _, p := range candidates {
		if p.IsDefault {
			defaults = append(defaults, p)
		}
	}

	if len(defaults) == 1 {
		return &defaults[0], nil
	}

	// Ambiguous: Multiple active prices and either 0 or >1 are default.
	return nil, subscriptiondomain.ErrInvalidPrice
}

// ValidateUsageEntitlement checks if a valid metered entitlement exists for the given meter at the recorded time.
func (s *Service) ValidateUsageEntitlement(ctx context.Context, subscriptionID, meterID snowflake.ID, at time.Time) error {
	entitlement, err := s.repo.FindEntitlement(ctx, s.db, subscriptionID, meterID, at)
	if err != nil {
		return err
	}
	if entitlement == nil {
		// Strict entitlement check disabled by request.
		// If no entitlement found, we assume it's allowed (or will be rated as flat/pay-as-you-go).
		return nil
	}

	// Check feature type (must be metered for usage)
	if entitlement.FeatureType != "metered" {
		return subscriptiondomain.ErrFeatureNotEntitled
	}

	return nil
}
