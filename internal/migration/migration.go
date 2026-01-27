package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies all embedded migrations, seeds system-level immutable data,
// and activates the schema bootstrap state. It must be run explicitly by the migrator entrypoint.
func RunMigrations(db *sql.DB) error {
	if db == nil {
		return errors.New("migration database handle is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	unlock, err := acquireAdvisoryLock(ctx, db)
	if err != nil {
		return err
	}
	defer func() {
		_ = unlock(context.Background())
	}()

	latestVersion, err := LatestMigrationVersion()
	if err != nil {
		return err
	}
	expectedVersion := fmt.Sprintf("%d", latestVersion)

	expectedChecksum, err := MigrationsChecksum()
	if err != nil {
		return err
	}

	sub, err := fs.Sub(embeddedMigrations, migrationsDir)
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}

	source, err := iofs.New(sub, ".")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if _, err := ensureNotDirty(migrator); err != nil {
		return err
	}

	upErr := migrator.Up()
	if upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", upErr)
	}

	currentVersion, err := ensureNotDirty(migrator)
	if err != nil {
		return err
	}

	if currentVersion != latestVersion {
		return fmt.Errorf("schema version mismatch after migrate: got %d want %d", currentVersion, latestVersion)
	}

	if err := seedSystemImmutableData(ctx, db); err != nil {
		return err
	}

	if err := activateSystemBootstrapState(ctx, db, expectedVersion, expectedChecksum); err != nil {
		return err
	}

	return nil
}

func ensureNotDirty(migrator *migrate.Migrate) (uint, error) {
	if migrator == nil {
		return 0, errors.New("migrator is required")
	}

	version, dirty, err := migrator.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, nil
		}
		return 0, fmt.Errorf("read migration version: %w", err)
	}
	if dirty {
		return 0, fmt.Errorf("database migrations are dirty at version %d", version)
	}
	return version, nil
}
