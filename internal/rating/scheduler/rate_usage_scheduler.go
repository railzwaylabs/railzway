package scheduler

import (
	"context"
	"hash/crc32"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultRatingInterval = 30 * time.Second
	defaultRatingBatch    = 200
)

type usageRow struct {
	ID    uuid.UUID `gorm:"column:id"`
	OrgID uuid.UUID `gorm:"column:org_id"`
}

func StartRatingScheduler(
	lc fx.Lifecycle,
	cfg *config.Config,
	db *gorm.DB,
	svc ratingdomain.Service,
	logger *zap.Logger,
) {
	interval := defaultRatingInterval
	if cfg != nil && cfg.Billing.RatingJobIntervalSec > 0 {
		interval = time.Duration(cfg.Billing.RatingJobIntervalSec) * time.Second
	}
	batchSize := defaultRatingBatch
	if cfg != nil && cfg.Billing.RatingJobBatchSize > 0 {
		batchSize = cfg.Billing.RatingJobBatchSize
	}
	useAdvisoryLock := cfg != nil && cfg.DB.Type == "postgres"
	lockKey := advisoryLockKey("rating.process_usage")
	var cancel context.CancelFunc

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ticker := time.NewTicker(interval)
			runCtx, stop := context.WithCancel(context.Background())
			cancel = stop
			logger.Info("rating scheduler started", zap.Duration("interval", interval), zap.Int("batch_size", batchSize))

			go func() {
				defer ticker.Stop()
				for {
					locked := true
					if useAdvisoryLock {
						var err error
						locked, err = tryAdvisoryLock(runCtx, db, lockKey)
						if err != nil {
							logger.Warn("rating lock failed", zap.Error(err))
							select {
							case <-ticker.C:
								continue
							case <-runCtx.Done():
								return
							}
						}
						if !locked {
							select {
							case <-ticker.C:
								continue
							case <-runCtx.Done():
								return
							}
						}
					}

					rows, err := listPendingUsage(runCtx, db, batchSize)
					if err != nil {
						logger.Warn("rating fetch failed", zap.Error(err))
					} else {
						rated := 0
						failed := 0
						for _, row := range rows {
							orgCtx := orgcontext.WithOrgID(context.Background(), row.OrgID)
							_, err := svc.RateUsage(orgCtx, ratingdomain.RateUsageRequest{
								UsageEventID: row.ID.String(),
							})
							if err != nil {
								failed++
								logger.Warn("rating failed", zap.Error(err), zap.String("usage_event_id", row.ID.String()))
								continue
							}
							rated++
						}

						if rated > 0 || failed > 0 {
							logger.Info("rating batch completed", zap.Int("rated", rated), zap.Int("failed", failed))
						}
					}

					if locked && useAdvisoryLock {
						if err := releaseAdvisoryLock(context.Background(), db, lockKey); err != nil {
							logger.Warn("rating unlock failed", zap.Error(err))
						}
					}

					select {
					case <-ticker.C:
						continue
					case <-runCtx.Done():
						return
					}
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
			}
			return nil
		},
	})
}

func listPendingUsage(ctx context.Context, db *gorm.DB, limit int) ([]usageRow, error) {
	if db == nil {
		return nil, nil
	}
	var rows []usageRow
	err := db.WithContext(ctx).Raw(
		`SELECT id, org_id
		 FROM usage_events
		 WHERE status IN (?, ?)
		 ORDER BY recorded_at ASC, id ASC
		 LIMIT ?`,
		usagedomain.StatusAccepted, usagedomain.StatusEnriched, limit,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
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
