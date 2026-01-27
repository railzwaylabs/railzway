package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	bootstrapStatusActive = "active"
)

func activateSystemBootstrapState(ctx context.Context, db *sql.DB, schemaVersion string, checksum string) error {
	if db == nil {
		return errors.New("bootstrap state requires database handle")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	version := strings.TrimSpace(schemaVersion)
	if version == "" {
		return errors.New("schema version is required for bootstrap state activation")
	}

	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
		INSERT INTO system_bootstrap_state (id, status, schema_version, checksum, activated_at, created_at)
		VALUES (TRUE, $1, $2, $3, $4, $4)
		ON CONFLICT (id) DO UPDATE
		SET status = EXCLUDED.status,
		    schema_version = EXCLUDED.schema_version,
		    checksum = EXCLUDED.checksum,
		    activated_at = EXCLUDED.activated_at
	`, bootstrapStatusActive, version, nullIfEmpty(checksum), now)
	if err != nil {
		return fmt.Errorf("activate system bootstrap state: %w", err)
	}

	return nil
}

func nullIfEmpty(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
