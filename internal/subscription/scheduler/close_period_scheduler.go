package scheduler

import (
	"context"
	"hash/crc32"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/config"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultClosePeriodInterval = 60 * time.Second
	defaultClosePeriodBatch    = 200
)

func StartClosePeriodScheduler(
	lc fx.Lifecycle,
	cfg *config.Config,
	db *gorm.DB,
	repo subscriptiondomain.Repository,
	invoiceSvc invoicedomain.Service,
	testclocks testclockdomain.Service,
	clk clock.Clock,
	logger *zap.Logger,
) {
	if clk == nil {
		clk = clock.SystemClock{}
	}

	interval := defaultClosePeriodInterval
	if cfg != nil && cfg.SubscriptionClosePeriodIntervalSec > 0 {
		interval = time.Duration(cfg.SubscriptionClosePeriodIntervalSec) * time.Second
	}
	batchSize := defaultClosePeriodBatch
	if cfg != nil && cfg.SubscriptionClosePeriodBatchSize > 0 {
		batchSize = cfg.SubscriptionClosePeriodBatchSize
	}

	gracePeriod := time.Duration(0)
	if cfg != nil && cfg.LateUsageGraceHours > 0 {
		gracePeriod = time.Duration(cfg.LateUsageGraceHours) * time.Hour
	}

	job := NewClosePeriodJob(db, repo, invoiceSvc, gracePeriod)
	useAdvisoryLock := cfg != nil && cfg.DBType == "postgres"
	lockKey := advisoryLockKey("subscription.close_period")

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ticker := time.NewTicker(interval)
			logger.Info("subscription close period scheduler started", zap.Duration("interval", interval), zap.Int("batch_size", batchSize))

			go func() {
				defer ticker.Stop()
				for {
					locked := true
					if useAdvisoryLock {
						var err error
						locked, err = tryAdvisoryLock(ctx, db, lockKey)
						if err != nil {
							logger.Warn("subscription close period lock failed", zap.Error(err))
							select {
							case <-ticker.C:
								continue
							case <-ctx.Done():
								return
							}
						}
						if !locked {
							select {
							case <-ticker.C:
								continue
							case <-ctx.Done():
								return
							}
						}
					}
					baseNow := clk.Now(context.Background())
					clockMap := map[uuid.UUID]time.Time{}
					maxNow := baseNow
					if testclocks != nil {
						if active, err := testclocks.ListActive(context.Background()); err == nil {
							for _, tc := range active {
								clockMap[tc.OrgID] = tc.CurrentTime
								if tc.CurrentTime.After(maxNow) {
									maxNow = tc.CurrentTime
								}
							}
						} else {
							logger.Warn("test clock list failed", zap.Error(err))
						}
					}
					periods, err := repo.ListOpenSubscriptionPeriods(context.Background(), maxNow, batchSize)
					if err != nil {
						logger.Warn("subscription close period failed", zap.Error(err))
					} else {
						result := ClosePeriodResult{}
						for _, period := range periods {
							asOf := baseNow
							if t, ok := clockMap[period.OrgID]; ok {
								asOf = t
							}
							if period.PeriodEnd.After(asOf) {
								continue
							}
							generated, err := job.closeOne(context.Background(), period, asOf)
							if err != nil {
								logger.Warn("subscription close period failed", zap.Error(err))
								break
							}
							result.ClosedPeriods++
							if generated {
								result.GeneratedInvoices++
							}
						}
						if result.ClosedPeriods > 0 {
							logger.Info("subscription close period completed",
								zap.Int("closed_periods", result.ClosedPeriods),
								zap.Int("generated_invoices", result.GeneratedInvoices),
							)
						}
					}

					if locked && useAdvisoryLock {
						if err := releaseAdvisoryLock(context.Background(), db, lockKey); err != nil {
							logger.Warn("subscription close period unlock failed", zap.Error(err))
						}
					}

					if gracePeriod > 0 {
						opened, err := openDueDraftInvoices(context.Background(), db, invoiceSvc, baseNow, clockMap, maxNow, gracePeriod, batchSize)
						if err != nil {
							logger.Warn("open draft invoices failed", zap.Error(err))
						} else if opened > 0 {
							logger.Info("opened draft invoices", zap.Int("opened", opened))
						}
					}

					select {
					case <-ticker.C:
						continue
					case <-ctx.Done():
						return
					}
				}
			}()

			return nil
		},
	})
}

func advisoryLockKey(name string) int64 {
	return int64(crc32.ChecksumIEEE([]byte(name)))
}

func tryAdvisoryLock(ctx context.Context, db *gorm.DB, key int64) (bool, error) {
	if db == nil {
		return false, nil
	}
	var locked bool
	if err := db.WithContext(ctx).Raw("SELECT pg_try_advisory_lock(?)", key).Scan(&locked).Error; err != nil {
		return false, err
	}
	return locked, nil
}

func releaseAdvisoryLock(ctx context.Context, db *gorm.DB, key int64) error {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx).Exec("SELECT pg_advisory_unlock(?)", key).Error
}

type draftInvoiceRow struct {
	ID        uuid.UUID `gorm:"column:id"`
	OrgID     uuid.UUID `gorm:"column:org_id"`
	PeriodEnd time.Time `gorm:"column:period_end"`
}

func openDueDraftInvoices(ctx context.Context, db *gorm.DB, invoiceSvc invoicedomain.Service, baseNow time.Time, clockMap map[uuid.UUID]time.Time, maxNow time.Time, grace time.Duration, limit int) (int, error) {
	if db == nil || invoiceSvc == nil {
		return 0, nil
	}
	cutoff := maxNow.UTC().Add(-grace)
	if limit <= 0 {
		limit = defaultClosePeriodBatch
	}

	var rows []draftInvoiceRow
	if err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, period_end
		 FROM invoices
		 WHERE status = ? AND period_end <= ?
		 ORDER BY period_end ASC, id ASC
		 LIMIT ?`,
		invoicedomain.StatusDraft, cutoff, limit,
	).Scan(&rows).Error; err != nil {
		return 0, err
	}

	opened := 0
	for _, row := range rows {
		asOf := baseNow
		if t, ok := clockMap[row.OrgID]; ok {
			asOf = t
		}
		if asOf.UTC().Before(row.PeriodEnd.Add(grace)) {
			continue
		}
		orgCtx := orgcontext.WithOrgID(context.Background(), row.OrgID)
		_, err := invoiceSvc.OpenInvoice(orgCtx, invoicedomain.OpenInvoiceRequest{ID: row.ID.String()})
		if err != nil {
			if err == invoicedomain.ErrInvalidStatus || err == invoicedomain.ErrNotFound {
				continue
			}
			return opened, err
		}
		opened++
	}

	return opened, nil
}
