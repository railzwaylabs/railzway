package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const advisoryLockKey int64 = 8_423_771_091

type unlockFunc func(ctx context.Context) error

func acquireAdvisoryLock(ctx context.Context, db *sql.DB) (unlockFunc, error) {
	if db == nil {
		return nil, errors.New("advisory lock requires database handle")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var locked bool
	err := db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockKey).Scan(&locked)
	if err != nil {
		return nil, fmt.Errorf("acquire advisory lock: %w", err)
	}
	if !locked {
		return nil, errors.New("another migration process holds the advisory lock")
	}

	return func(unlockCtx context.Context) error {
		if unlockCtx == nil {
			unlockCtx = context.Background()
		}
		var released bool
		if err := db.QueryRowContext(unlockCtx, "SELECT pg_advisory_unlock($1)", advisoryLockKey).Scan(&released); err != nil {
			return fmt.Errorf("release advisory lock: %w", err)
		}
		if !released {
			return errors.New("advisory lock was not held by this session")
		}
		return nil
	}, nil
}
