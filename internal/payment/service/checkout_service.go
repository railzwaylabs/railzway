package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/payment/adapters"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
	priceamountdomain "github.com/railzwaylabs/railzway/internal/priceamount/domain"
	providerservice "github.com/railzwaylabs/railzway/internal/providers/payment/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CheckoutServiceParams struct {
	fx.In

	Registry             *adapters.Registry
	ProviderService      providerservice.Service
	PaymentMethodService domain.PaymentMethodService
	SubscriptionService  subscriptiondomain.Service
	PriceAmountService   priceamountdomain.Service
	Repo                 domain.CheckoutSessionRepository
	GenID                *snowflake.Node
	Logger               *zap.Logger
	DB                   *gorm.DB
}

type CheckoutServiceImpl struct {
	registry             *adapters.Registry
	providerService      providerservice.Service
	paymentMethodService domain.PaymentMethodService
	subscriptionService  subscriptiondomain.Service
	priceAmountService   priceamountdomain.Service
	repo                 domain.CheckoutSessionRepository
	genID                *snowflake.Node
	logger               *zap.Logger
	db                   *gorm.DB
}

func NewCheckoutService(p CheckoutServiceParams) domain.CheckoutService {
	return &CheckoutServiceImpl{
		registry:             p.Registry,
		providerService:      p.ProviderService,
		paymentMethodService: p.PaymentMethodService,
		subscriptionService:  p.SubscriptionService,
		priceAmountService:   p.PriceAmountService,
		repo:                 p.Repo,
		genID:                p.GenID,
		logger:               p.Logger,
		db:                   p.DB,
	}
}

// calculateLineItemsTotal fetches price amounts for line items and calculates total
// Filters price amounts by the requested currency
// Returns: totalAmount, lineItemsJSON, error
func (s *CheckoutServiceImpl) calculateLineItemsTotal(ctx context.Context, lineItems []domain.LineItemInput, currency string) (int64, []byte, error) {
	if len(lineItems) == 0 {
		return 0, nil, fmt.Errorf("line_items cannot be empty")
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return 0, nil, fmt.Errorf("currency is required")
	}

	var totalAmount int64
	type lineItemData struct {
		PriceID  string `json:"price_id"`
		Quantity int    `json:"quantity"`
	}
	lineItemsData := make([]lineItemData, 0, len(lineItems))

	for _, item := range lineItems {
		// Parse price ID
		priceID, err := snowflake.ParseString(item.PriceID)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid price_id: %s", item.PriceID)
		}

		// Fetch price amounts for this price
		listResp, err := s.priceAmountService.List(ctx, priceamountdomain.ListPriceAmountRequest{
			PriceID:  priceID.String(),
			PageSize: -1,
		})
		if err != nil {
			return 0, nil, fmt.Errorf("failed to fetch price amount for price %s: %w", item.PriceID, err)
		}

		priceAmounts := listResp.Amounts
		if len(priceAmounts) == 0 {
			return 0, nil, fmt.Errorf("no price amount found for price %s", item.PriceID)
		}

		// Filter by currency
		var priceAmount *priceamountdomain.Response
		for i := range priceAmounts {
			if strings.ToUpper(priceAmounts[i].Currency) == currency {
				priceAmount = &priceAmounts[i]
				break
			}
		}

		if priceAmount == nil {
			return 0, nil, fmt.Errorf("no price amount found for price %s with currency %s", item.PriceID, currency)
		}

		// Calculate line item total
		lineTotal := priceAmount.UnitAmountCents * int64(item.Quantity)
		totalAmount += lineTotal

		// Store line item data
		lineItemsData = append(lineItemsData, lineItemData{
			PriceID:  item.PriceID,
			Quantity: item.Quantity,
		})
	}

	// Marshal line items to JSON
	lineItemsJSON, err := json.Marshal(lineItemsData)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to marshal line items: %w", err)
	}

	return totalAmount, lineItemsJSON, nil
}

func (s *CheckoutServiceImpl) CreateSession(ctx context.Context, input domain.CheckoutSessionInput) (*domain.CheckoutSession, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	// 1. Calculate total amount from line items (filter by currency)
	totalAmount, lineItemsJSON, err := s.calculateLineItemsTotal(ctx, input.LineItems, input.Currency)
	if err != nil {
		s.logger.Error("failed to calculate line items total", zap.Error(err))
		return nil, fmt.Errorf("invalid line_items: %w", err)
	}

	// 2. Get provider config
	providerCfg, err := s.providerService.GetActiveProviderConfig(ctx, orgID, input.Provider)
	if err != nil {
		return nil, err
	}

	var configMap map[string]any
	if err := json.Unmarshal(providerCfg.Config, &configMap); err != nil {
		return nil, domain.ErrInvalidConfig
	}

	// 3. Create adapter
	adapter, err := s.registry.NewAdapter(input.Provider, domain.AdapterConfig{
		OrgID:    orgID,
		Provider: input.Provider,
		Config:   configMap,
	})
	if err != nil {
		return nil, err
	}

	// 4. Create checkout session via provider
	// Set calculated amount in input for adapter to use
	input.Amount = totalAmount
	providerSession, err := adapter.CreateCheckoutSession(ctx, input)
	if err != nil {
		return nil, err
	}

	// 5. Save to database
	now := time.Now().UTC()

	// Convert metadata to JSONMap
	metadata := make(map[string]any)
	for k, v := range input.Metadata {
		metadata[k] = v
	}

	session := &domain.CheckoutSession{
		ID:                s.genID.Generate(),
		OrgID:             orgID,
		CustomerID:        &input.CustomerID,
		Provider:          input.Provider,
		Status:            providerSession.Status,
		PaymentStatus:     domain.PaymentStatusUnpaid,
		AmountTotal:       totalAmount,
		Currency:          strings.ToUpper(input.Currency),
		LineItems:         lineItemsJSON,
		SuccessURL:        input.SuccessURL,
		CancelURL:         input.CancelURL,
		ClientReferenceID: input.ClientReferenceID,
		PaymentIntentID:   providerSession.PaymentIntentID,
		ProviderSessionID: providerSession.ID,
		Metadata:          metadata,
		ExpiresAt:         &providerSession.ExpiresAt,
		CreatedAt:         now,
		UpdatedAt:         now,
		URL:               providerSession.URL,
	}

	if err := s.repo.Insert(ctx, s.db, session); err != nil {
		s.logger.Error("failed to save checkout session", zap.Error(err))
		return nil, err
	}

	if err := s.repo.Update(ctx, nil, session); err != nil {
		s.logger.Error("failed to update checkout session", zap.Error(err))
		// Return success anyway
		// But our DB is out of sync. Better to return error or try to fix.
		// For consistency, returning session with logged error is acceptable for now.
		return nil, err
	}

	s.logger.Info("checkout session created",
		zap.String("session_id", session.ID.String()),
		zap.String("provider_session_id", providerSession.ID))

	return session, nil
}

// GetSession retrieves a checkout session by ID
func (s *CheckoutServiceImpl) GetSession(ctx context.Context, id snowflake.ID) (*domain.CheckoutSession, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	session, err := s.repo.FindByID(ctx, s.db, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCheckoutSessionNotFound
		}
		return nil, err
	}

	// Verify org ownership
	if session.OrgID != orgID {
		return nil, domain.ErrCheckoutSessionNotFound
	}

	return session, nil
}

// GetLineItems retrieves line items for a checkout session
func (s *CheckoutServiceImpl) GetLineItems(ctx context.Context, sessionID snowflake.ID) ([]domain.LineItem, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	// Get session
	session, err := s.repo.FindByID(ctx, s.db, sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCheckoutSessionNotFound
		}
		return nil, err
	}

	// Verify org ownership
	if session.OrgID != orgID {
		return nil, domain.ErrCheckoutSessionNotFound
	}

	// Parse line items from JSON
	type storedLineItem struct {
		PriceID  string `json:"price_id"`
		Quantity int    `json:"quantity"`
	}
	var storedItems []storedLineItem
	if err := json.Unmarshal(session.LineItems, &storedItems); err != nil {
		return nil, fmt.Errorf("failed to parse line items: %w", err)
	}

	// Fetch price amounts for each line item
	result := make([]domain.LineItem, 0, len(storedItems))
	for _, item := range storedItems {
		priceID, err := snowflake.ParseString(item.PriceID)
		if err != nil {
			continue // Skip invalid price IDs
		}

		// Fetch price amount (filter by session currency)
		listResp, err := s.priceAmountService.List(ctx, priceamountdomain.ListPriceAmountRequest{
			PriceID:  priceID.String(),
			PageSize: -1,
		})
		if err != nil || len(listResp.Amounts) == 0 {
			continue // Skip if price not found
		}
		priceAmounts := listResp.Amounts

		// Find matching currency
		var unitAmount int64
		for _, pa := range priceAmounts {
			if strings.ToUpper(pa.Currency) == session.Currency {
				unitAmount = pa.UnitAmountCents
				break
			}
		}

		result = append(result, domain.LineItem{
			PriceID:     item.PriceID,
			Quantity:    item.Quantity,
			UnitAmount:  unitAmount,
			AmountTotal: unitAmount * int64(item.Quantity),
		})
	}

	return result, nil
}

// ExpireSession marks a checkout session as expired
func (s *CheckoutServiceImpl) ExpireSession(ctx context.Context, id snowflake.ID) (*domain.CheckoutSession, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	session, err := s.repo.FindByID(ctx, s.db, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrCheckoutSessionNotFound
		}
		return nil, err
	}

	// Verify org ownership
	if session.OrgID != orgID {
		return nil, domain.ErrCheckoutSessionNotFound
	}

	// Check if already expired or completed
	if session.Status == domain.CheckoutSessionStatusExpired {
		return session, nil // Already expired
	}

	if session.Status == domain.CheckoutSessionStatusComplete {
		return nil, fmt.Errorf("cannot expire completed session")
	}

	// Update status
	now := time.Now().UTC()
	session.Status = domain.CheckoutSessionStatusExpired
	session.ExpiredAt = &now
	session.UpdatedAt = now

	if err := s.repo.Update(ctx, s.db, session); err != nil {
		s.logger.Error("failed to expire checkout session", zap.Error(err))
		return nil, err
	}

	s.logger.Info("checkout session expired",
		zap.String("session_id", session.ID.String()))

	return session, nil
}

// CompleteSession marks a checkout session as complete and creates subscription
func (s *CheckoutServiceImpl) CompleteSession(ctx context.Context, provider, providerSessionID string) (*domain.CheckoutSession, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	// 1. Find Session
	session, err := s.repo.FindByProviderSessionID(ctx, nil, provider, providerSessionID)
	if err != nil {
		return nil, err
	}

	// 2. Idempotency Check - if already completed, return existing
	if session.Status == domain.CheckoutSessionStatusComplete {
		s.logger.Info("session already completed, returning existing",
			zap.String("session_id", session.ID.String()),
			zap.String("provider_session_id", providerSessionID))
		return session, nil
	}

	// 3. Complete Session Logic
	// a. Retrieve Provider Session details
	providerCfg, err := s.providerService.GetActiveProviderConfig(ctx, session.OrgID, provider)
	if err != nil {
		return nil, err
	}
	var configMap map[string]any
	if err := json.Unmarshal(providerCfg.Config, &configMap); err != nil {
		return nil, domain.ErrInvalidConfig
	}
	adapter, err := s.registry.NewAdapter(provider, domain.AdapterConfig{
		OrgID:    session.OrgID,
		Provider: provider,
		Config:   configMap,
	})
	if err != nil {
		return nil, err
	}

	providerSession, err := adapter.RetrieveCheckoutSession(ctx, providerSessionID)
	if err != nil {
		s.logger.Error("failed to retrieve provider session", zap.Error(err))
		// Continue anyway - we can still mark as complete based on webhook verification
	} else if providerSession != nil {
		session.PaymentIntentID = providerSession.PaymentMethodID
		// Note: Payment method is automatically attached by Stripe Checkout
		// No need to manually attach payment method here
	}

	// c. Update Status
	session.Status = domain.CheckoutSessionStatusComplete
	session.PaymentStatus = domain.PaymentStatusPaid
	now := time.Now().UTC()
	session.CompletedAt = &now

	if err := s.repo.Update(ctx, nil, session); err != nil {
		return nil, err
	}

	// d. Create Subscription via Service
	if session.CustomerID != nil {
		customerID := session.CustomerID.String()

		// Parse Line Items or Metadata
		priceID := ""
		if val, ok := session.Metadata["price_id"]; ok {
			if priceIDStr, ok := val.(string); ok {
				priceID = priceIDStr
			} else {
				s.logger.Warn("price_id in metadata is not a string",
					zap.String("session_id", session.ID.String()),
					zap.Any("price_id_value", val))
			}
		}

		if priceID != "" {
			s.logger.Info("creating subscription from checkout session",
				zap.String("customer_id", customerID),
				zap.String("price_id", priceID),
				zap.String("session_id", session.ID.String()))

			req := subscriptiondomain.CreateSubscriptionRequest{
				CustomerID:       customerID,
				BillingCycleType: "monthly", // Default to monthly for checkout sessions
				CollectionMode:   subscriptiondomain.SubscriptionCollectionModeChargeAutomatically,
				Items: []subscriptiondomain.CreateSubscriptionItemRequest{
					{PriceID: priceID, Quantity: 1},
				},
			}

			subResp, err := s.subscriptionService.Create(ctx, req)
			if err != nil {
				s.logger.Error("failed to create subscription",
					zap.Error(err),
					zap.String("customer_id", customerID),
					zap.String("price_id", priceID),
					zap.String("session_id", session.ID.String()))
				// Don't fail the entire checkout completion, but log prominently
				// The session is still marked as complete, subscription can be retried
			} else {
				s.logger.Info("subscription created successfully",
					zap.String("subscription_id", subResp.ID),
					zap.String("customer_id", customerID))

				// e. Transition to Active
				err = s.subscriptionService.TransitionSubscription(ctx, subResp.ID, subscriptiondomain.SubscriptionStatusActive, "checkout_session")
				if err != nil {
					s.logger.Error("failed to transition subscription to active",
						zap.Error(err),
						zap.String("subscription_id", subResp.ID))
				} else {
					s.logger.Info("subscription transitioned to active",
						zap.String("subscription_id", subResp.ID))
				}

				// Save subscription ID to metadata and DTO
				if session.Metadata == nil {
					session.Metadata = make(map[string]any)
				}
				session.Metadata["subscription_id"] = subResp.ID
				session.SubscriptionID = subResp.ID

				// Update session with new metadata
				if err := s.repo.Update(ctx, nil, session); err != nil {
					s.logger.Error("failed to update session metadata with subscription_id", zap.Error(err))
				}
			}
		} else {
			s.logger.Warn("no price_id in session metadata, skipping subscription creation",
				zap.String("session_id", session.ID.String()))
		}
	}

	return session, nil
}

// VerifyAndComplete verifies a checkout session and completes it if payment succeeded
// This is called synchronously from the frontend after payment redirect
func (s *CheckoutServiceImpl) VerifyAndComplete(ctx context.Context, sessionID string) (*domain.CheckoutSession, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	var session *domain.CheckoutSession

	// 1. Parse session ID
	id, err := snowflake.ParseString(sessionID)
	if err != nil {
		// Not a valid Snowflake ID, try looking up by Provider Session ID
		// This happens when verifying via Stripe ID directly (cs_test_...)
		s.logger.Info("verify: session_id is not a snowflake id, trying provider lookup",
			zap.String("session_id", sessionID))

		session, err = s.repo.FindByAnyProviderSessionID(ctx, nil, sessionID)
		if err != nil {
			return nil, err
		}
	} else {
		// 2. Find session in DB by Snowflake ID
		session, err = s.repo.FindByID(ctx, nil, id)
		if err != nil {
			return nil, err
		}
		if session == nil {
			// Try provider lookup even if it looked like a snowflake ID but wasn't found
			session, err = s.repo.FindByAnyProviderSessionID(ctx, nil, sessionID)
			if err != nil {
				return nil, err
			}
		}
	}

	if session == nil {
		return nil, domain.ErrCheckoutSessionNotFound
	}

	// 3. Verify org ownership
	if session.OrgID != orgID {
		return nil, domain.ErrInvalidOrganization
	}

	// 4. If already completed, return existing
	if session.Status == domain.CheckoutSessionStatusComplete {
		s.logger.Info("session already completed in verify",
			zap.String("session_id", sessionID))

		// Populate DTO fields from metadata
		if val, ok := session.Metadata["subscription_id"]; ok {
			if subID, ok := val.(string); ok {
				session.SubscriptionID = subID
			}
		}
		return session, nil
	}

	// 5. Retrieve from provider to verify payment status
	providerCfg, err := s.providerService.GetActiveProviderConfig(ctx, session.OrgID, session.Provider)
	if err != nil {
		return nil, err
	}

	var configMap map[string]any
	if err := json.Unmarshal(providerCfg.Config, &configMap); err != nil {
		return nil, domain.ErrInvalidConfig
	}

	adapter, err := s.registry.NewAdapter(session.Provider, domain.AdapterConfig{
		OrgID:    session.OrgID,
		Provider: session.Provider,
		Config:   configMap,
	})
	if err != nil {
		return nil, err
	}

	providerSession, err := adapter.RetrieveCheckoutSession(ctx, session.ProviderSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provider session: %w", err)
	}

	// 6. Check if payment completed
	if providerSession.Status != domain.CheckoutSessionStatusComplete {
		s.logger.Info("session not yet completed",
			zap.String("session_id", sessionID),
			zap.String("status", string(providerSession.Status)))
		return session, nil // Return current status without error
	}

	// 7. Complete the session (this will create subscription)
	s.logger.Info("payment verified, completing session",
		zap.String("session_id", sessionID))

	return s.CompleteSession(ctx, session.Provider, session.ProviderSessionID)
}
