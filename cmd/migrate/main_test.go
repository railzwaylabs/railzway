package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/railzwaylabs/railzway/internal/config"
)

func TestResolveDatabaseURLPostgres(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.Type = "postgres"
	cfg.DB.Host = "db.internal"
	cfg.DB.Port = "5432"
	cfg.DB.User = "postgres"
	cfg.DB.Pass = "secret"
	cfg.DB.Name = "railzway"
	cfg.DB.SSLMode = "disable"

	got, err := resolveDatabaseURL(cfg, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := "postgres://postgres:secret@db.internal:5432/railzway?sslmode=disable&timezone=UTC"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveMigrationPathUsesOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := resolveMigrationPath(filepath.Join(dir, "nested"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want, _ := filepath.Abs(filepath.Join(dir, "nested"))
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestMigrationSourceURL(t *testing.T) {
	got := migrationSourceURL("/app/db/migrations")
	if got != "file:///app/db/migrations" {
		t.Fatalf("unexpected source url %q", got)
	}
}
