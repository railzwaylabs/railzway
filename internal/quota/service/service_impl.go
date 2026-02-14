package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	quotadomain "github.com/railzwaylabs/railzway/internal/quota/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ServiceParam struct {
	fx.In

	DB           *gorm.DB
	Redis        *redis.Client
	Log          *zap.Logger
	Config       *quotadomain.Config
	CustomerRepo customerdomain.Repository
	SubRepo      subscriptiondomain.Repository
}

type service struct {
	db           *gorm.DB
	redis        *redis.Client
	log          *zap.Logger
	cfg          *quotadomain.Config
	customerRepo customerdomain.Repository
	subRepo      subscriptiondomain.Repository
}

func NewService(p ServiceParam) quotadomain.Service {
	return &service{
		db:           p.DB,
		redis:        p.Redis,
		log:          p.Log.Named("quota.service"),
		cfg:          p.Config,
		customerRepo: p.CustomerRepo,
		subRepo:      p.SubRepo,
	}
}

func (s *service) CanIngestUsage(ctx context.Context, orgID snowflake.ID) error {
	if !s.cfg.Enabled {
		return nil
	}

	// Key: quota:usage:{org_id}:{month_year} e.g. quota:usage:123:2023-10
	now := time.Now().UTC()
	key := fmt.Sprintf("quota:usage:%s:%s", orgID.String(), now.Format("2006-01"))

	// Atomic INCR
	val, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		s.log.Error("failed to increment usage quota", zap.Error(err))
		// Fail open to avoid blocking ingestion on redis error
		return nil
	}

	// Set expiration if new key (35 days to be safe)
	if val == 1 {
		s.redis.Expire(ctx, key, 35*24*time.Hour)
	}

	if s.isQuotaExceeded(val, s.cfg.OrgUsageMonthly) {
		return quotadomain.ErrOrgUsageQuotaExceeded
	}

	return nil
}
func (s *service) CanCreateCustomer(ctx context.Context, orgID snowflake.ID) error {
	if !s.cfg.Enabled {
		return nil
	}

	count, err := s.customerRepo.Count(ctx, s.db, orgID)
	if err != nil {
		s.log.Error("failed to count customers", zap.Error(err))
		return err
	}
	if s.isQuotaExceeded(count, s.cfg.OrgCustomer) {
		return quotadomain.ErrOrgCustomerQuotaExceeded
	}

	return nil
}

func (s *service) CanCreateSubscription(ctx context.Context, orgID snowflake.ID) error {
	if !s.cfg.Enabled {
		return nil
	}

	count, err := s.subRepo.Count(ctx, s.db, orgID)
	if err != nil {
		s.log.Error("failed to count subscriptions", zap.Error(err))
		return err
	}

	if s.isQuotaExceeded(count, s.cfg.OrgSubscription) {
		return quotadomain.ErrOrgSubscriptionQuotaExceeded
	}

	return nil
}

func (s *service) GetOrgUsage(ctx context.Context, orgID snowflake.ID) (map[string]int64, error) {
	if !s.cfg.Enabled {
		return nil, quotadomain.ErrQuotaDisabled
	}

	usage := make(map[string]int64)

	custCount, err := s.customerRepo.Count(ctx, s.db, orgID)
	if err == nil {
		usage["customers"] = custCount
	}

	subCount, err := s.subRepo.Count(ctx, s.db, orgID)
	if err == nil {
		usage["subscriptions"] = subCount
	}

	return usage, nil
}

func (s *service) isQuotaExceeded(count int64, limit int) bool {
	return limit != -1 && count >= int64(limit)
}
