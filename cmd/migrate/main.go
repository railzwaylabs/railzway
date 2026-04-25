package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/spf13/cobra"
)

type migrateOptions struct {
	target      string
	path        string
	databaseURL string
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "railzway-migrate: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	opts := &migrateOptions{}

	root := &cobra.Command{
		Use:   "railzway-migrate",
		Short: "Run Railzway database migrations",
	}
	root.PersistentFlags().StringVar(&opts.target, "target", "", "optional config target (for example: admin, scheduler, api)")
	root.PersistentFlags().StringVar(&opts.path, "path", "", "optional path to migration files")
	root.PersistentFlags().StringVar(&opts.databaseURL, "database-url", "", "optional database URL override")

	root.AddCommand(
		newUpCommand(opts),
		newDownCommand(opts),
		newStepsCommand(opts),
		newGotoCommand(opts),
		newForceCommand(opts),
		newVersionCommand(opts),
		newDropCommand(opts),
	)

	return root
}

func newUpCommand(opts *migrateOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "up [steps]",
		Short: "Apply all up migrations or a limited number of steps",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrator(opts, func(m *migrate.Migrate) error {
				if len(args) == 0 {
					if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
						return err
					}
					fmt.Fprintln(cmd.OutOrStdout(), "applied up migrations")
					return nil
				}

				steps, err := parsePositiveInt(args[0], "steps")
				if err != nil {
					return err
				}
				if err := m.Steps(steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "applied %d up step(s)\n", steps)
				return nil
			})
		},
	}
}

func newDownCommand(opts *migrateOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "down [steps]",
		Short: "Rollback all migrations or a limited number of steps",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrator(opts, func(m *migrate.Migrate) error {
				if len(args) == 0 {
					if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
						return err
					}
					fmt.Fprintln(cmd.OutOrStdout(), "rolled back all migrations")
					return nil
				}

				steps, err := parsePositiveInt(args[0], "steps")
				if err != nil {
					return err
				}
				if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "rolled back %d step(s)\n", steps)
				return nil
			})
		},
	}
}

func newStepsCommand(opts *migrateOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "steps <n>",
		Short: "Move migration state by N steps; negative values step down",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrator(opts, func(m *migrate.Migrate) error {
				steps, err := strconv.Atoi(strings.TrimSpace(args[0]))
				if err != nil || steps == 0 {
					return fmt.Errorf("invalid steps value %q", args[0])
				}
				if err := m.Steps(steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "applied %d migration step(s)\n", steps)
				return nil
			})
		},
	}
}

func newGotoCommand(opts *migrateOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "goto <version>",
		Short: "Migrate to an exact version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrator(opts, func(m *migrate.Migrate) error {
				version, err := parseVersion(args[0])
				if err != nil {
					return err
				}
				if err := m.Migrate(version); err != nil && !errors.Is(err, migrate.ErrNoChange) {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "migrated to version %d\n", version)
				return nil
			})
		},
	}
}

func newForceCommand(opts *migrateOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "force <version>",
		Short: "Force-set the migration version without running migrations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrator(opts, func(m *migrate.Migrate) error {
				version, err := parseVersion(args[0])
				if err != nil {
					return err
				}
				if err := m.Force(int(version)); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "forced migration version to %d\n", version)
				return nil
			})
		},
	}
}

func newVersionCommand(opts *migrateOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the current migration version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withMigrator(opts, func(m *migrate.Migrate) error {
				version, dirty, err := m.Version()
				if err != nil {
					if errors.Is(err, migrate.ErrNilVersion) {
						fmt.Fprintln(cmd.OutOrStdout(), "version=0 dirty=false")
						return nil
					}
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "version=%d dirty=%t\n", version, dirty)
				return nil
			})
		},
	}
}

func newDropCommand(opts *migrateOptions) *cobra.Command {
	var confirm bool
	cmd := &cobra.Command{
		Use:   "drop",
		Short: "Drop everything in the configured database",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("drop requires --yes")
			}
			return withMigrator(opts, func(m *migrate.Migrate) error {
				if err := m.Drop(); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "dropped database objects")
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&confirm, "yes", false, "confirm destructive drop operation")
	return cmd
}

func withMigrator(opts *migrateOptions, run func(*migrate.Migrate) error) error {
	cfg, err := loadConfig(opts.target)
	if err != nil {
		return err
	}

	databaseURL, err := resolveDatabaseURL(cfg, opts.databaseURL)
	if err != nil {
		return err
	}

	migrationPath, err := resolveMigrationPath(opts.path)
	if err != nil {
		return err
	}

	sourceURL := migrationSourceURL(migrationPath)
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	return run(m)
}

func loadConfig(target string) (*config.Config, error) {
	if strings.TrimSpace(target) == "" {
		return config.Register()
	}
	return config.RegisterFor(target)()
}

func resolveDatabaseURL(cfg *config.Config, override string) (string, error) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed, nil
	}
	if cfg == nil {
		return "", fmt.Errorf("missing config")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.DB.Type)) {
	case "postgres":
		user := url.UserPassword(cfg.DB.User, cfg.DB.Pass)
		values := url.Values{}
		if strings.TrimSpace(cfg.DB.SSLMode) != "" {
			values.Set("sslmode", cfg.DB.SSLMode)
		}
		values.Set("timezone", "UTC")

		return (&url.URL{
			Scheme:   "postgres",
			User:     user,
			Host:     joinHostPort(cfg.DB.Host, cfg.DB.Port),
			Path:     "/" + strings.TrimPrefix(strings.TrimSpace(cfg.DB.Name), "/"),
			RawQuery: values.Encode(),
		}).String(), nil
	case "mysql":
		return fmt.Sprintf(
			"mysql://%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC&multiStatements=true",
			url.QueryEscape(cfg.DB.User),
			url.QueryEscape(cfg.DB.Pass),
			joinHostPort(cfg.DB.Host, cfg.DB.Port),
			strings.TrimSpace(cfg.DB.Name),
		), nil
	case "sqlite":
		return "", fmt.Errorf("sqlite is not supported by railzway-migrate")
	default:
		return "", fmt.Errorf("unsupported database type %q", cfg.DB.Type)
	}
}

func resolveMigrationPath(override string) (string, error) {
	candidates := []string{}
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		candidates = append(candidates, trimmed)
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "db", "migrations"))
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "db", "migrations"),
			filepath.Join(exeDir, "..", "db", "migrations"),
		)
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			abs, absErr := filepath.Abs(candidate)
			if absErr == nil {
				return abs, nil
			}
			return candidate, nil
		}
	}

	return "", fmt.Errorf("migration path not found; tried %s", strings.Join(uniqueStrings(candidates), ", "))
}

func migrationSourceURL(path string) string {
	slashed := filepath.ToSlash(path)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	return "file://" + slashed
}

func closeMigrator(m *migrate.Migrate) {
	if m == nil {
		return
	}
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		fmt.Fprintf(os.Stderr, "railzway-migrate: source close error: %v\n", srcErr)
	}
	if dbErr != nil {
		fmt.Fprintf(os.Stderr, "railzway-migrate: database close error: %v\n", dbErr)
	}
}

func parsePositiveInt(raw, label string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid %s value %q", label, raw)
	}
	return value, nil
}

func parseVersion(raw string) (uint, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid version value %q", raw)
	}
	return uint(value), nil
}

func joinHostPort(host, port string) string {
	h := strings.TrimSpace(host)
	p := strings.TrimSpace(port)
	if p == "" {
		return h
	}
	return h + ":" + p
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
