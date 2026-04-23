package scheduler

import (
	"context"
	"hash/crc32"
	"time"

	"github.com/railzwaylabs/railzway/internal/admin/service"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultRefreshInterval = 60 * time.Second

func StartSummaryRefresher(lc fx.Lifecycle, cfg *config.Config, db *gorm.DB, svc *service.Service, logger *zap.Logger) {
	interval := defaultRefreshInterval
	if cfg != nil && cfg.SummaryRefreshIntervalSec > 0 {
		interval = time.Duration(cfg.SummaryRefreshIntervalSec) * time.Second
	}
	useAdvisoryLock := cfg != nil && cfg.DBType == "postgres"
	lockKey := advisoryLockKey("admin.refresh_summaries")
	var cancel context.CancelFunc

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ticker := time.NewTicker(interval)
			runCtx, stop := context.WithCancel(context.Background())
			cancel = stop
			logger.Info("admin summary refresher started", zap.Duration("interval", interval))

			go func() {
				defer ticker.Stop()
				for {
					locked := true
					if useAdvisoryLock {
						var err error
						locked, err = tryAdvisoryLock(runCtx, db, lockKey)
						if err != nil {
							logger.Warn("admin summary lock failed", zap.Error(err))
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

					if err := svc.RefreshAllSummaries(context.Background()); err != nil {
						logger.Warn("admin summary refresh failed", zap.Error(err))
					}

					if locked && useAdvisoryLock {
						if err := releaseAdvisoryLock(context.Background(), db, lockKey); err != nil {
							logger.Warn("admin summary unlock failed", zap.Error(err))
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
