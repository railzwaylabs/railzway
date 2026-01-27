package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// LatestMigrationVersion returns the highest embedded migration version.
func LatestMigrationVersion() (uint, error) {
	entries, err := fs.ReadDir(embeddedMigrations, migrationsDir)
	if err != nil {
		return 0, fmt.Errorf("list migrations: %w", err)
	}

	var maxVersion uint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		version, ok := parseMigrationVersion(name)
		if !ok {
			return 0, fmt.Errorf("invalid migration filename: %s", name)
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	if maxVersion == 0 {
		return 0, errors.New("no embedded migrations found")
	}
	return maxVersion, nil
}

// MigrationsChecksum computes a deterministic checksum of embedded migrations.
func MigrationsChecksum() (string, error) {
	entries, err := fs.ReadDir(embeddedMigrations, migrationsDir)
	if err != nil {
		return "", fmt.Errorf("list migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if strings.HasSuffix(name, ".up.sql") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	hasher := sha256.New()
	for _, name := range names {
		path := migrationsDir + "/" + name
		content, err := embeddedMigrations.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read migration %s: %w", name, err)
		}
		_, _ = hasher.Write([]byte(name))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write(content)
		_, _ = hasher.Write([]byte{0})
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func parseMigrationVersion(name string) (uint, bool) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0, false
	}
	value := strings.TrimSpace(parts[0])
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(parsed), true
}
