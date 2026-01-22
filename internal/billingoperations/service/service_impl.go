package service

import (
	"crypto/sha256"
	"strings"

	"github.com/bwmarrin/snowflake"
	auditdomain "github.com/smallbiznis/railzway/internal/audit/domain"
	"github.com/smallbiznis/railzway/internal/billingoperations/domain"
	"github.com/smallbiznis/railzway/internal/billingoperations/repository"
	"github.com/smallbiznis/railzway/internal/clock"
	"github.com/smallbiznis/railzway/internal/config"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Params struct {
	fx.In

	DB       *gorm.DB
	Log      *zap.Logger
	Clock    clock.Clock
	GenID    *snowflake.Node
	AuditSvc auditdomain.Service `optional:"true"`
	Cfg      config.Config

	BillingConfig *config.BillingConfigHolder
}

type Service struct {
	repo     domain.Repository
	db       *gorm.DB
	log      *zap.Logger
	clock    clock.Clock
	genID    *snowflake.Node
	auditSvc auditdomain.Service
	encKey   []byte

	billingCfg *config.BillingConfigHolder
}

func NewService(p Params) domain.Service {
	repo := repository.NewRepository(p.DB)

	secret := strings.TrimSpace(p.Cfg.PaymentProviderConfigSecret)
	var key []byte
	if secret != "" {
		sum := sha256.Sum256([]byte(secret))
		key = sum[:]
	}

	return &Service{
		repo:       repo,
		db:         p.DB,
		log:        p.Log.Named("billingoperations.service"),
		clock:      p.Clock,
		genID:      p.GenID,
		auditSvc:   p.AuditSvc,
		encKey:     key,
		billingCfg: p.BillingConfig,
	}
}
