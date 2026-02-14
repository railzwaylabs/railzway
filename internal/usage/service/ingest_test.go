package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	"github.com/railzwaylabs/railzway/internal/meter/domain"
	meterdomain "github.com/railzwaylabs/railzway/internal/meter/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// -- Mocks --

type subscriptionMock struct {
	mock.Mock
}

func (m *subscriptionMock) GetActiveByCustomerID(ctx context.Context, req subscriptiondomain.GetActiveByCustomerIDRequest) (subscriptiondomain.Subscription, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return subscriptiondomain.Subscription{}, args.Error(1)
	}
	return args.Get(0).(subscriptiondomain.Subscription), args.Error(1)
}

func (m *subscriptionMock) ValidateUsageEntitlement(ctx context.Context, subID, meterID snowflake.ID, at time.Time) error {
	args := m.Called(ctx, subID, meterID, at)
	return args.Error(0)
}

func (m *subscriptionMock) List(context.Context, subscriptiondomain.ListSubscriptionRequest) (subscriptiondomain.ListSubscriptionResponse, error) {
	return subscriptiondomain.ListSubscriptionResponse{}, nil
}
func (m *subscriptionMock) ListEntitlements(context.Context, subscriptiondomain.ListEntitlementsRequest) (subscriptiondomain.ListEntitlementsResponse, error) {
	return subscriptiondomain.ListEntitlementsResponse{}, nil
}
func (m *subscriptionMock) Create(context.Context, subscriptiondomain.CreateSubscriptionRequest) (subscriptiondomain.CreateSubscriptionResponse, error) {
	return subscriptiondomain.CreateSubscriptionResponse{}, nil
}
func (m *subscriptionMock) ReplaceItems(context.Context, subscriptiondomain.ReplaceSubscriptionItemsRequest) (subscriptiondomain.CreateSubscriptionResponse, error) {
	return subscriptiondomain.CreateSubscriptionResponse{}, nil
}
func (m *subscriptionMock) GetByID(context.Context, string) (subscriptiondomain.Subscription, error) {
	return subscriptiondomain.Subscription{}, nil
}
func (m *subscriptionMock) GetSubscriptionItem(context.Context, subscriptiondomain.GetSubscriptionItemRequest) (subscriptiondomain.SubscriptionItem, error) {
	return subscriptiondomain.SubscriptionItem{}, nil
}
func (m *subscriptionMock) TransitionSubscription(ctx context.Context, id string, status subscriptiondomain.SubscriptionStatus, reason subscriptiondomain.TransitionReason) error {
	return nil
}
func (m *subscriptionMock) ChangePlan(ctx context.Context, req subscriptiondomain.ChangePlanRequest) error {
	return nil
}

type meterMock struct {
	mock.Mock
}

func (m *meterMock) GetByCode(ctx context.Context, code string) (*domain.Response, error) {
	args := m.Called(ctx, code)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*domain.Response), args.Error(1)
}

func (m *meterMock) Create(ctx context.Context, req domain.CreateRequest) (*domain.Response, error) {
	return nil, nil
}
func (m *meterMock) List(ctx context.Context, req domain.ListRequest) ([]domain.Response, error) {
	return nil, nil
}
func (m *meterMock) GetByID(ctx context.Context, id string) (*domain.Response, error) {
	return nil, nil
}
func (m *meterMock) Update(ctx context.Context, req domain.UpdateRequest) (*domain.Response, error) {
	return nil, nil
}
func (m *meterMock) Delete(ctx context.Context, id string) error { return nil }

type quotaMock struct {
	mock.Mock
}

func (m *quotaMock) CanCreateCustomer(ctx context.Context, orgID snowflake.ID) error {
	return m.Called(ctx, orgID).Error(0)
}
func (m *quotaMock) CanCreateSubscription(ctx context.Context, orgID snowflake.ID) error {
	return m.Called(ctx, orgID).Error(0)
}
func (m *quotaMock) CanIngestUsage(ctx context.Context, orgID snowflake.ID) error {
	args := m.Called(ctx, orgID)
	return args.Error(0)
}
func (m *quotaMock) GetOrgUsage(ctx context.Context, orgID snowflake.ID) (map[string]int64, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).(map[string]int64), args.Error(1)
}

// -- Tests --

func TestIngest_EntitlementGating(t *testing.T) {
	// Setup In-Memory DB (needed for NewService and Ingest insert)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Migrate usage_events table
	if err := db.AutoMigrate(&usagedomain.UsageEvent{}); err != nil {
		t.Fatal(err)
	}

	node, _ := snowflake.NewNode(1)
	genID := node
	logger := zap.NewNop()

	orgID := genID.Generate()
	customerID := genID.Generate()
	meterID := genID.Generate()
	subID := genID.Generate()

	tests := []struct {
		name         string
		req          usagedomain.CreateIngestRequest
		setupMocks   func(*subscriptionMock, *meterMock, *quotaMock)
		expectedErr  error
		expectIngest bool // If we expect ingestion to succeed
	}{
		{
			name: "Success: Valid Entitlement",
			req: usagedomain.CreateIngestRequest{
				CustomerID:     customerID.String(),
				MeterCode:      "m1",
				Value:          10,
				RecordedAt:     time.Now(),
				IdempotencyKey: "k1",
			},
			setupMocks: func(s *subscriptionMock, m *meterMock, q *quotaMock) {
				// 0. Check Quota
				q.On("CanIngestUsage", mock.Anything, mock.Anything).Return(nil)
				// 1. Resolve Meter
				m.On("GetByCode", mock.Anything, "m1").Return(&meterdomain.Response{
					ID:   meterID.String(),
					Code: "m1",
				}, nil)
				// 2. Resolve Subscription
				s.On("GetActiveByCustomerID", mock.Anything, mock.MatchedBy(func(req subscriptiondomain.GetActiveByCustomerIDRequest) bool {
					return req.CustomerID == customerID.String()
				})).Return(subscriptiondomain.Subscription{ID: subID}, nil)
				// 3. Validate Entitlement -> Success
				s.On("ValidateUsageEntitlement", mock.Anything, subID, meterID, mock.Anything).Return(nil)
			},
			expectedErr:  nil,
			expectIngest: true,
		},
		{
			name: "Failure: Quota Exceeded",
			req: usagedomain.CreateIngestRequest{
				CustomerID:     customerID.String(),
				MeterCode:      "m1",
				Value:          10,
				RecordedAt:     time.Now(),
				IdempotencyKey: "k_quota_fail",
			},
			setupMocks: func(s *subscriptionMock, m *meterMock, q *quotaMock) {
				// 0. Check Quota matches failure
				q.On("CanIngestUsage", mock.Anything, mock.Anything).Return(errors.New("org_usage_quota_exceeded"))
			},
			expectedErr:  errors.New("org_usage_quota_exceeded"),
			expectIngest: false,
		},
		{
			name: "Failure: Meter Not Found",
			req: usagedomain.CreateIngestRequest{
				CustomerID:     customerID.String(),
				MeterCode:      "invalid_meter",
				Value:          10,
				RecordedAt:     time.Now(),
				IdempotencyKey: "k2",
			},
			setupMocks: func(s *subscriptionMock, m *meterMock, q *quotaMock) {
				q.On("CanIngestUsage", mock.Anything, mock.Anything).Return(nil)
				// Subscription MUST be resolved before Meter
				s.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return(subscriptiondomain.Subscription{ID: subID}, nil)
				m.On("GetByCode", mock.Anything, "invalid_meter").Return(nil, meterdomain.ErrMeterNotFound)
			},
			expectedErr:  usagedomain.ErrInvalidMeter, // Assuming service maps it or returns ErrInvalidMeter if nil
			expectIngest: false,
		},
		{
			name: "Failure: No Active Subscription",
			req: usagedomain.CreateIngestRequest{
				CustomerID:     customerID.String(),
				MeterCode:      "m1",
				Value:          10,
				RecordedAt:     time.Now(),
				IdempotencyKey: "k3",
			},
			setupMocks: func(s *subscriptionMock, m *meterMock, q *quotaMock) {
				q.On("CanIngestUsage", mock.Anything, mock.Anything).Return(nil)
				// 1. Resolve Subscription -> Not Found
				s.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return(subscriptiondomain.Subscription{}, subscriptiondomain.ErrSubscriptionNotFound)
			},
			expectedErr:  usagedomain.ErrInvalidSubscription,
			expectIngest: false,
		},
		{
			name: "Failure: Not Entitled",
			req: usagedomain.CreateIngestRequest{
				CustomerID:     customerID.String(),
				MeterCode:      "m1",
				Value:          10,
				RecordedAt:     time.Now(),
				IdempotencyKey: "k4",
			},
			setupMocks: func(s *subscriptionMock, m *meterMock, q *quotaMock) {
				q.On("CanIngestUsage", mock.Anything, mock.Anything).Return(nil)
				m.On("GetByCode", mock.Anything, "m1").Return(&meterdomain.Response{
					ID:   meterID.String(),
					Code: "m1",
				}, nil)
				s.On("GetActiveByCustomerID", mock.Anything, mock.Anything).Return(subscriptiondomain.Subscription{ID: subID}, nil)
				// 3. Validate Entitlement -> FeatureNotEntitled
				s.On("ValidateUsageEntitlement", mock.Anything, subID, meterID, mock.Anything).Return(subscriptiondomain.ErrFeatureNotEntitled)
			},
			expectedErr:  usagedomain.ErrFeatureNotEntitled,
			expectIngest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSub := new(subscriptionMock)
			mockMeter := new(meterMock)
			mockQuota := new(quotaMock)

			if tt.setupMocks != nil {
				tt.setupMocks(mockSub, mockMeter, mockQuota)
			}

			svc := NewService(ServiceParam{
				DB:       db,
				Log:      logger,
				GenID:    genID,
				MeterSvc: mockMeter,
				SubSvc:   mockSub,
				QuotaSvc: mockQuota,
			})

			ctx := WithTestOrgContext(context.Background(), orgID)
			t.Logf("Test %s: orgID=%s", tt.name, orgID)

			res, err := svc.Ingest(ctx, tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if tt.expectIngest == false {
					assert.Nil(t, res)
				}
				if tt.expectedErr.Error() != "" {
					assert.Contains(t, err.Error(), tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
				if tt.expectIngest {
					assert.NotNil(t, res)
				}
			}
		})
	}
}

func TestIngest_Idempotency_Strict(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&usagedomain.UsageEvent{})
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS ux_usage_events_idempotency ON usage_events(org_id, idempotency_key)")

	node, _ := snowflake.NewNode(1)
	orgID := node.Generate()
	customerID := node.Generate()
	subID := node.Generate()
	meterID := node.Generate()

	mockSub := new(subscriptionMock)
	mockMeter := new(meterMock)
	mockQuota := new(quotaMock)

	// Setup valid responses
	mockQuota.On("CanIngestUsage", mock.Anything, mock.Anything).Return(nil)
	mockMeter.On("GetByCode", mock.Anything, "m1").Return(&meterdomain.Response{ID: meterID.String(), Code: "m1"}, nil)
	mockSub.On("GetActiveByCustomerID", mock.Anything, mock.MatchedBy(func(req subscriptiondomain.GetActiveByCustomerIDRequest) bool {
		return req.CustomerID == customerID.String()
	})).Return(subscriptiondomain.Subscription{ID: subID}, nil)

	// Entitlement passes initially
	mockSub.On("ValidateUsageEntitlement", mock.Anything, subID, meterID, mock.Anything).Return(nil)

	svc := NewService(ServiceParam{
		DB:       db,
		Log:      zap.NewNop(),
		GenID:    node,
		MeterSvc: mockMeter,
		SubSvc:   mockSub,
		QuotaSvc: mockQuota,
	})

	ctx := WithTestOrgContext(context.Background(), orgID)
	key := "strict_idempotency_key_1"
	req := usagedomain.CreateIngestRequest{
		CustomerID:     customerID.String(),
		MeterCode:      "m1",
		Value:          1,
		RecordedAt:     time.Now(),
		IdempotencyKey: key,
	}

	// 1. First Call: Success
	res1, err := svc.Ingest(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, res1)
	assert.Equal(t, key, res1.IdempotencyKey)

	// 2. Second Call: Success (Same Result)
	res2, err := svc.Ingest(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, res1.ID, res2.ID, "Must return same ID")

	// 3. Third Call: With Entitlement Failure (Simulate cancelled sub)
	// We create a NEW service instance or update mocks if possible.
	// Since mocks are referenced, we can try to override or since we used "Return", it might be sticky.
	// Let's rely on the "Fast Path" preventing the mock call entirely.
	// If the code calls ValidateUsageEntitlement, it will return nil (pass) because of the mock above.
	// To verify logic, we should assert that ValidateUsageEntitlement is NOT called again?
	// Or we can construct a test where the fast path is the ONLY way to succeed.
}

func TestIngest_Idempotency_BypassEntitlementFailure(t *testing.T) {
	// dedicated test for the "Entitlement Revoked" Case
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&usagedomain.UsageEvent{})
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS ux_usage_events_idempotency ON usage_events(org_id, idempotency_key)")

	node, _ := snowflake.NewNode(1)
	orgID := node.Generate()
	customerID := node.Generate()
	subID := node.Generate()
	meterID := node.Generate()

	mockSub := new(subscriptionMock)
	mockMeter := new(meterMock)
	mockQuota := new(quotaMock)

	// 1. Setup Valid Mocks for SEEDING
	mockQuota.On("CanIngestUsage", mock.Anything, mock.Anything).Return(nil)
	mockMeter.On("GetByCode", mock.Anything, "m1").Return(&meterdomain.Response{ID: meterID.String(), Code: "m1"}, nil)
	mockSub.On("GetActiveByCustomerID", mock.Anything, mock.MatchedBy(func(req subscriptiondomain.GetActiveByCustomerIDRequest) bool {
		return req.CustomerID == customerID.String()
	})).Return(subscriptiondomain.Subscription{ID: subID}, nil)
	mockSub.On("ValidateUsageEntitlement", mock.Anything, subID, meterID, mock.Anything).Return(nil)

	svc := NewService(ServiceParam{
		DB:       db,
		Log:      zap.NewNop(),
		GenID:    node,
		MeterSvc: mockMeter,
		SubSvc:   mockSub,
		QuotaSvc: mockQuota,
	})

	ctx := WithTestOrgContext(context.Background(), orgID)
	key := "pre_existing_key"
	req := usagedomain.CreateIngestRequest{
		CustomerID:     customerID.String(),
		MeterCode:      "m1",
		Value:          10,
		RecordedAt:     time.Now(),
		IdempotencyKey: key,
	}

	// 2. Seed the Event
	res1, err := svc.Ingest(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, res1)

	// 3. Reset Mocks to FAIL EVERYTHING (Clear expectations)
	mockSub.ExpectedCalls = nil
	mockSub.Calls = nil
	mockMeter.ExpectedCalls = nil
	mockMeter.Calls = nil
	mockQuota.ExpectedCalls = nil
	mockQuota.Calls = nil

	// If any service method is called now, it should panic/fail because there are no expectations.
	// We want to verify that Ingest returns 'res1' purely from DB check without consulting services.

	// 4. Ingest Again (Should be Idempotent)
	res2, err := svc.Ingest(ctx, req)

	// Expect Success (Returning existing) WITHOUT calling mocks
	assert.NoError(t, err)
	assert.Equal(t, res1.ID, res2.ID)
}

// Helper for context
func WithTestOrgContext(ctx context.Context, orgID snowflake.ID) context.Context {
	return orgcontext.WithOrgID(ctx, int64(orgID))
}
