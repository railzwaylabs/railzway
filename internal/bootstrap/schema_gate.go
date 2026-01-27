package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/railzwaylabs/railzway/internal/migration"
	"gorm.io/gorm"
)

var (
	ErrBootstrapStateInactive = errors.New("system bootstrap state is not active")
	ErrSchemaVersionMismatch  = errors.New("schema version mismatch")
	ErrSchemaChecksumMismatch = errors.New("schema checksum mismatch")
)

type SchemaGate interface {
	MustBeActive(ctx context.Context) error
}

type schemaGate struct {
	db               *gorm.DB
	expectedVersion  string
	expectedChecksum string
}

func NewSchemaGate(db *gorm.DB) (SchemaGate, error) {
	if db == nil {
		return nil, errors.New("schema gate requires database handle")
	}

	latestVersion, err := migration.LatestMigrationVersion()
	if err != nil {
		return nil, err
	}
	expectedVersion := fmt.Sprintf("%d", latestVersion)

	expectedChecksum, err := migration.MigrationsChecksum()
	if err != nil {
		return nil, err
	}

	return &schemaGate{
		db:               db,
		expectedVersion:  expectedVersion,
		expectedChecksum: expectedChecksum,
	}, nil
}

func (g *schemaGate) MustBeActive(ctx context.Context) error {
	state, err := loadSystemBootstrapState(ctx, g.db)
	if err != nil {
		return err
	}

	if state.Status != StatusActive {
		return fmt.Errorf("%w: status=%s", ErrBootstrapStateInactive, state.Status)
	}

	if state.SchemaVersion != g.expectedVersion {
		return fmt.Errorf("%w: state=%s expected=%s", ErrSchemaVersionMismatch, state.SchemaVersion, g.expectedVersion)
	}

	if state.Checksum != nil && strings.TrimSpace(*state.Checksum) != "" {
		if g.expectedChecksum == "" || *state.Checksum != g.expectedChecksum {
			return fmt.Errorf("%w: state=%s expected=%s", ErrSchemaChecksumMismatch, *state.Checksum, g.expectedChecksum)
		}
	}

	return nil
}
