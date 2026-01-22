package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bwmarrin/snowflake"
	"github.com/redis/go-redis/v9"
	"github.com/smallbiznis/railzway/internal/customer/domain"
	quotadomain "github.com/smallbiznis/railzway/internal/quota/domain"
	"github.com/smallbiznis/railzway/internal/quota/service"
	subdomain "github.com/smallbiznis/railzway/internal/subscription/domain"
	"github.com/smallbiznis/railzway/pkg/db/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// --- Mocks ---

type MockCustomerRepo struct {
	mock.Mock
}

func (m *MockCustomerRepo) Count(ctx context.Context, db *gorm.DB, orgID snowflake.ID) (int64, error) {
	args := m.Called(ctx, db, orgID)
	return args.Get(0).(int64), args.Error(1)
}

// Fixed Signatures
func (m *MockCustomerRepo) Insert(ctx context.Context, db *gorm.DB, c *domain.Customer) error {
	return nil
}
func (m *MockCustomerRepo) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*domain.Customer, error) {
	return nil, nil
}
func (m *MockCustomerRepo) List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, filter domain.ListCustomerFilter, page pagination.Pagination) ([]*domain.Customer, error) {
	return nil, nil
}

type MockSubRepo struct {
	mock.Mock
}

func (m *MockSubRepo) Count(ctx context.Context, db *gorm.DB, orgID snowflake.ID) (int64, error) {
	args := m.Called(ctx, db, orgID)
	return args.Get(0).(int64), args.Error(1)
}

// Fixed Signatures & Missing methods
func (m *MockSubRepo) Insert(ctx context.Context, db *gorm.DB, s *subdomain.Subscription) error { return nil }
func (m *MockSubRepo) InsertItems(ctx context.Context, db *gorm.DB, items []subdomain.SubscriptionItem) error { return nil }
func (m *MockSubRepo) InsertEntitlements(ctx context.Context, db *gorm.DB, entitlements []subdomain.SubscriptionEntitlement) error { return nil }
func (m *MockSubRepo) ReplaceItems(ctx context.Context, db *gorm.DB, orgID, subscriptionID snowflake.ID, items []subdomain.SubscriptionItem) error { return nil }
func (m *MockSubRepo) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*subdomain.Subscription, error) { return nil, nil }
func (m *MockSubRepo) FindByIDForUpdate(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*subdomain.Subscription, error) { return nil, nil }
func (m *MockSubRepo) List(ctx context.Context, db *gorm.DB, orgID snowflake.ID) ([]subdomain.Subscription, error) { return nil, nil }
func (m *MockSubRepo) FindActiveByCustomerID(ctx context.Context, db *gorm.DB, orgID, customerID snowflake.ID, statuses []subdomain.SubscriptionStatus) (*subdomain.Subscription, error) { return nil, nil }
func (m *MockSubRepo) FindActiveByCustomerIDAt(ctx context.Context, db *gorm.DB, orgID, customerID snowflake.ID, at time.Time) (*subdomain.Subscription, error) { return nil, nil }
func (m *MockSubRepo) FindSubscriptionItemByMeterID(ctx context.Context, db *gorm.DB, orgID, subscriptionID, meterID snowflake.ID) (*subdomain.SubscriptionItem, error) { return nil, nil }
func (m *MockSubRepo) FindSubscriptionItemByMeterIDAt(ctx context.Context, db *gorm.DB, orgID, subscriptionID, meterID snowflake.ID, at time.Time) (*subdomain.SubscriptionItem, error) { return nil, nil }
func (m *MockSubRepo) FindSubscriptionItemByMeterCode(ctx context.Context, db *gorm.DB, orgID, subscriptionID snowflake.ID, meterCode string) (*subdomain.SubscriptionItem, error) { return nil, nil }
func (m *MockSubRepo) FindEntitlement(ctx context.Context, db *gorm.DB, subscriptionID snowflake.ID, meterID snowflake.ID, at time.Time) (*subdomain.SubscriptionEntitlement, error) { return nil, nil }


// --- Tests ---

func TestCanIngestUsage(t *testing.T) {
	// Setup Redis
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	cfg := &quotadomain.Config{
		Enabled:         true,
		OrgUsageMonthly: 5, // Small limit for testing
	}

	logger := zap.NewNop()
	svc := service.NewService(service.ServiceParam{
		Redis:  rdb,
		Log:    logger,
		Config: cfg,
	})

	ctx := context.Background()
	orgID := snowflake.ID(123)

	// 1. Ingest 5 events (OK)
	for i := 0; i < 5; i++ {
		err := svc.CanIngestUsage(ctx, orgID)
		assert.NoError(t, err)
	}

	// 2. Ingest 6th event (Exceeded)
	err = svc.CanIngestUsage(ctx, orgID)
	assert.ErrorIs(t, err, quotadomain.ErrOrgUsageQuotaExceeded)

	// 3. Verify Redis Key exists
	// miniredis Keys takes a pattern
	keys := s.Keys() 
	assert.Len(t, keys, 1) // quota:usage:123:YYYY-MM
}

func TestCanCreateCustomer(t *testing.T) {
	cfg := &quotadomain.Config{
		Enabled:     true,
		OrgCustomer: 10,
	}

	mockRepo := new(MockCustomerRepo)
	svc := service.NewService(service.ServiceParam{
		DB:           &gorm.DB{}, // Mock DB struct
		Log:          zap.NewNop(),
		Config:       cfg,
		CustomerRepo: mockRepo,
	})
	
	ctx := context.Background()
	orgID := snowflake.ID(123)

	// 1. Below Limit
	mockRepo.On("Count", ctx, mock.Anything, orgID).Return(int64(5), nil).Once()
	err := svc.CanCreateCustomer(ctx, orgID)
	assert.NoError(t, err)

	// 2. At Limit
	mockRepo.On("Count", ctx, mock.Anything, orgID).Return(int64(10), nil).Once()
	err = svc.CanCreateCustomer(ctx, orgID)
	assert.ErrorIs(t, err, quotadomain.ErrOrgCustomerQuotaExceeded)
}

func TestCanCreateSubscription(t *testing.T) {
	cfg := &quotadomain.Config{
		Enabled:         true,
		OrgSubscription: 2,
	}

	mockRepo := new(MockSubRepo)
	svc := service.NewService(service.ServiceParam{
		DB:      &gorm.DB{},
		Log:     zap.NewNop(),
		Config:  cfg,
		SubRepo: mockRepo,
	})

	ctx := context.Background()
	orgID := snowflake.ID(123)

	// 1. Below Limit
	mockRepo.On("Count", ctx, mock.Anything, orgID).Return(int64(1), nil).Once()
	err := svc.CanCreateSubscription(ctx, orgID)
	assert.NoError(t, err)

	// 2. Over Limit
	mockRepo.On("Count", ctx, mock.Anything, orgID).Return(int64(2), nil).Once()
	err = svc.CanCreateSubscription(ctx, orgID)
	assert.ErrorIs(t, err, quotadomain.ErrOrgSubscriptionQuotaExceeded)
}
