package service

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	pricedomain "github.com/railzwaylabs/railzway/internal/price/domain"
	priceamountdomain "github.com/railzwaylabs/railzway/internal/priceamount/domain"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Manual Mocks

type mockPriceService struct {
	prices []pricedomain.Response
}

func (m *mockPriceService) Create(ctx context.Context, req pricedomain.CreateRequest) (*pricedomain.Response, error) {
	return nil, nil
}
func (m *mockPriceService) Get(ctx context.Context, id string) (*pricedomain.Response, error) {
	for _, p := range m.prices {
		if p.ID.String() == id {
			return &p, nil
		}
	}
	return nil, nil
}
func (m *mockPriceService) List(ctx context.Context, opts pricedomain.ListOptions) ([]pricedomain.Response, error) {
	return m.prices, nil
}

type mockProductFeatureRepo struct {
	features []productfeaturedomain.FeatureAssignment
}

func (m *mockProductFeatureRepo) ListByProduct(ctx context.Context, db *gorm.DB, orgID, productID snowflake.ID) ([]productfeaturedomain.FeatureAssignment, error) {
	return nil, nil
}
func (m *mockProductFeatureRepo) ListByProducts(ctx context.Context, db *gorm.DB, orgID snowflake.ID, productIDs []snowflake.ID) ([]productfeaturedomain.FeatureAssignment, error) {
	var result []productfeaturedomain.FeatureAssignment
	for _, f := range m.features {
		for _, pid := range productIDs {
			if f.ProductID == pid {
				result = append(result, f)
			}
		}
	}
	return result, nil
}
func (m *mockProductFeatureRepo) Replace(ctx context.Context, tx *gorm.DB, productID snowflake.ID, featureIDs []snowflake.ID, now time.Time) error {
	return nil
}

// Mock Repository
type mockRepository struct {
	subscriptions map[string]*subscriptiondomain.Subscription
}

func (m *mockRepository) Insert(ctx context.Context, db *gorm.DB, subscription *subscriptiondomain.Subscription) error {
	m.subscriptions[subscription.ID.String()] = subscription
	return db.Create(subscription).Error
}
func (m *mockRepository) InsertItems(ctx context.Context, db *gorm.DB, items []subscriptiondomain.SubscriptionItem) error {
	return db.Create(items).Error
}
func (m *mockRepository) ReplaceItems(ctx context.Context, db *gorm.DB, orgID, subscriptionID snowflake.ID, items []subscriptiondomain.SubscriptionItem) error {
	db.Where("org_id = ? AND subscription_id = ?", orgID, subscriptionID).Delete(&subscriptiondomain.SubscriptionItem{})
	return db.Create(items).Error
}
func (m *mockRepository) InsertEntitlements(ctx context.Context, db *gorm.DB, entitlements []subscriptiondomain.SubscriptionEntitlement) error {
	return db.Create(entitlements).Error
}
func (m *mockRepository) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*subscriptiondomain.Subscription, error) {
	if s, ok := m.subscriptions[id.String()]; ok {
		// Need to return a copy or pointer to updated struct if modified?
		// Actually, if we want to confirm PlanChangedAt, we should read from DB if `ChangePlan` wrote to DB.
		// `ChangePlan` wrote to DB via `tx.Exec`. So memory map is STALE unless we update it.
		// But `ChangePlan` doesn't update repo map (it doesn't know about it).
		// So validation should check DB directly, not use repo.
		return s, nil
	}
	return nil, nil
}
func (m *mockRepository) FindByIDForUpdate(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*subscriptiondomain.Subscription, error) {
	if s, ok := m.subscriptions[id.String()]; ok {
		return s, nil
	}
	return nil, nil
}
func (m *mockRepository) List(ctx context.Context, db *gorm.DB, orgID snowflake.ID) ([]subscriptiondomain.Subscription, error) {
	return nil, nil
}
func (m *mockRepository) FindActiveByCustomerID(ctx context.Context, db *gorm.DB, orgID, customerID snowflake.ID, statuses []subscriptiondomain.SubscriptionStatus) (*subscriptiondomain.Subscription, error) {
	return nil, nil
}
func (m *mockRepository) FindActiveByCustomerIDAt(ctx context.Context, db *gorm.DB, orgID, customerID snowflake.ID, at time.Time) (*subscriptiondomain.Subscription, error) {
	return nil, nil
}
func (m *mockRepository) FindSubscriptionItemByMeterCode(ctx context.Context, db *gorm.DB, orgID, subscriptionID snowflake.ID, meterCode string) (*subscriptiondomain.SubscriptionItem, error) {
	return nil, nil
}
func (m *mockRepository) FindSubscriptionItemByMeterID(ctx context.Context, db *gorm.DB, orgID, subscriptionID, meterID snowflake.ID) (*subscriptiondomain.SubscriptionItem, error) {
	return nil, nil
}
func (m *mockRepository) FindSubscriptionItemByMeterIDAt(ctx context.Context, db *gorm.DB, orgID, subscriptionID, meterID snowflake.ID, at time.Time) (*subscriptiondomain.SubscriptionItem, error) {
	return nil, nil
}
func (m *mockRepository) FindEntitlement(ctx context.Context, db *gorm.DB, subscriptionID snowflake.ID, meterID snowflake.ID, at time.Time) (*subscriptiondomain.SubscriptionEntitlement, error) {
	return nil, nil
}
func (m *mockRepository) Count(ctx context.Context, db *gorm.DB, orgID snowflake.ID) (int64, error) {
	return 0, nil
}

// Helper to init DB
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(
		&subscriptiondomain.Subscription{},
		&subscriptiondomain.SubscriptionItem{},
		&subscriptiondomain.SubscriptionEntitlement{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

func TestChangePlan(t *testing.T) {
	db := setupTestDB(t)
	node, _ := snowflake.NewNode(1)
	repo := &mockRepository{
		subscriptions: make(map[string]*subscriptiondomain.Subscription),
	}
	logger := zap.NewNop()

	// Setup data
	orgID := node.Generate()
	customerID := node.Generate()
	oldProductID := node.Generate()
	newProductID := node.Generate()

	oldPriceID := node.Generate()
	newPriceID := node.Generate()

	// Mocks
	priceSvc := &mockPriceService{
		prices: []pricedomain.Response{
			{
				ID:              oldPriceID,
				OrganizationID:  orgID,
				ProductID:       oldProductID,
				BillingInterval: pricedomain.Month,
				Active:          true,
				PricingModel:    pricedomain.Flat,
				BillingMode:     pricedomain.Licensed,
			},
			{
				ID:              newPriceID,
				OrganizationID:  orgID,
				ProductID:       newProductID,
				BillingInterval: pricedomain.Month, // Same cycle for simplicity
				Active:          true,
				IsDefault:       true,
				PricingModel:    pricedomain.Flat,
				BillingMode:     pricedomain.Licensed,
			},
		},
	}

	pfRepo := &mockProductFeatureRepo{
		features: []productfeaturedomain.FeatureAssignment{
			{
				FeatureID:   node.Generate(),
				ProductID:   newProductID,
				Code:        "new_feature",
				Name:        "New Feature",
				FeatureType: "boolean",
				Active:      true,
			},
		},
	}

	// Svc
	svc := NewService(ServiceParam{
		DB:                 db,
		Log:                logger,
		GenID:              node,
		Clock:              &mockClock{},
		Repo:               repo,
		Pricesvc:           priceSvc,
		ProductFeatureRepo: pfRepo,
		// PriceAmountsvc needed for loadPriceAmount if pricing model not flat
		PriceAmountsvc: &mockPriceAmountService{},
        PaymentMethodSvc: &mockPaymentMethodService{},
	})

	// Create active subscription
	subID := node.Generate()
	now := time.Now().UTC()
	sub := &subscriptiondomain.Subscription{
		ID:               subID,
		OrgID:            orgID,
		CustomerID:       customerID,
		Status:           subscriptiondomain.SubscriptionStatusActive,
		BillingCycleType: "monthly",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	repo.Insert(context.Background(), db, sub)

	// Create old entitlement - WE MUST INSERT into DB manually because repo.InsertEntitlements does (via mock)
	// but direct usage sets effectiveFrom.
	repo.InsertEntitlements(context.Background(), db, []subscriptiondomain.SubscriptionEntitlement{
		{
			ID:             node.Generate(),
			SubscriptionID: subID,
			FeatureCode:    "old_feature",
			EffectiveFrom:  now.Add(-24 * time.Hour),
		},
	})

	// Inject OrgID into context
	ctx := orgcontext.WithOrgID(context.Background(), int64(orgID))

	// Execute ChangePlan
	req := subscriptiondomain.ChangePlanRequest{
		SubscriptionID: subID.String(),
		NewProductID:   newProductID.String(),
	}

	err := svc.ChangePlan(ctx, req)
	if err != nil {
		t.Fatalf("ChangePlan failed: %v", err)
	}

	// Verify Entitlements
	var entitlements []subscriptiondomain.SubscriptionEntitlement
	db.Where("subscription_id = ?", subID).Find(&entitlements)

	if len(entitlements) != 2 {
		t.Errorf("Expected 2 entitlements, got %d", len(entitlements))
	}

	var oldEnt, newEnt subscriptiondomain.SubscriptionEntitlement
	for _, e := range entitlements {
		if e.FeatureCode == "old_feature" {
			oldEnt = e
		} else if e.FeatureCode == "new_feature" {
			newEnt = e
		}
	}

	// Verify old entitlement closed
	if oldEnt.EffectiveTo == nil {
		t.Error("Old entitlement should be closed (EffectiveTo set)")
	}

	// Verify new entitlement open
	if newEnt.EffectiveTo != nil {
		t.Error("New entitlement should be open (EffectiveTo nil)")
	}
	if newEnt.EffectiveFrom.IsZero() {
		t.Error("New entitlement should have EffectiveFrom set")
	}

	// Verify Subscription PlanChangedAt by querying DB directly
	var checkSub subscriptiondomain.Subscription
	if err := db.Where("id = ?", subID).First(&checkSub).Error; err != nil {
		t.Fatalf("Failed to retrieve subscription from DB: %v", err)
	}

	if checkSub.PlanChangedAt == nil {
		t.Error("PlanChangedAt should be set")
	}
}

type mockClock struct{}

func (m *mockClock) Now() time.Time { return time.Now().UTC() }

// Mock PriceAmountService (minimal)
type mockPriceAmountService struct{}

func (m *mockPriceAmountService) Create(ctx context.Context, req priceamountdomain.CreateRequest) (*priceamountdomain.Response, error) {
	return &priceamountdomain.Response{}, nil
}
func (m *mockPriceAmountService) List(ctx context.Context, req priceamountdomain.ListPriceAmountRequest) ([]priceamountdomain.Response, error) {
	return nil, nil
}
func (m *mockPriceAmountService) Get(ctx context.Context, req priceamountdomain.GetPriceAmountByID) (*priceamountdomain.Response, error) {
	return nil, nil
}

type mockPaymentMethodService struct{}

func (m *mockPaymentMethodService) AttachPaymentMethod(ctx context.Context, customerID snowflake.ID, provider, token string) (*paymentdomain.PaymentMethod, error) {
	return nil, nil
}
func (m *mockPaymentMethodService) DetachPaymentMethod(ctx context.Context, customerID, paymentMethodID snowflake.ID) error {
	return nil
}
func (m *mockPaymentMethodService) ListPaymentMethods(ctx context.Context, customerID snowflake.ID) ([]*paymentdomain.PaymentMethod, error) {
	return nil, nil
}
func (m *mockPaymentMethodService) SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID snowflake.ID) error {
	return nil
}
func (m *mockPaymentMethodService) GetDefaultPaymentMethod(ctx context.Context, customerID snowflake.ID) (*paymentdomain.PaymentMethod, error) {
	return &paymentdomain.PaymentMethod{ID: 1}, nil
}
