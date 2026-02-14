package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/audit"
	"github.com/railzwaylabs/railzway/internal/authorization"
	"github.com/railzwaylabs/railzway/internal/billingdashboard/rollup"
	"github.com/railzwaylabs/railzway/internal/billingoperations"
	"github.com/railzwaylabs/railzway/internal/bootstrap"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/feature"
	"github.com/railzwaylabs/railzway/internal/invoice"
	"github.com/railzwaylabs/railzway/internal/invoicetemplate"
	"github.com/railzwaylabs/railzway/internal/ledger"
	"github.com/railzwaylabs/railzway/internal/meter"
	"github.com/railzwaylabs/railzway/internal/migration"
	"github.com/railzwaylabs/railzway/internal/observability"
	"github.com/railzwaylabs/railzway/internal/price"
	"github.com/railzwaylabs/railzway/internal/priceamount"
	"github.com/railzwaylabs/railzway/internal/pricetier"
	"github.com/railzwaylabs/railzway/internal/product"
	"github.com/railzwaylabs/railzway/internal/productfeature"
	"github.com/railzwaylabs/railzway/internal/rating"
	"github.com/railzwaylabs/railzway/internal/redis"
	"github.com/railzwaylabs/railzway/internal/scheduler"
	"github.com/railzwaylabs/railzway/internal/server"
	"github.com/railzwaylabs/railzway/internal/subscription"
	"github.com/railzwaylabs/railzway/pkg/db"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "railzway",
		Short:   "Railzway CLI",
		Version: readVersionFromEnv(),
	}
	root.AddCommand(newMigrateCmd(), newServeCmd(), newSchedulerCmd(), newAllCmd())
	return root
}

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations and activate schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate()
		},
	}
}

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run admin UI + API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			runServe()
			return nil
		},
	}
}

func newSchedulerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scheduler",
		Short: "Run background scheduler workers",
		RunE: func(cmd *cobra.Command, args []string) error {
			runScheduler()
			return nil
		},
	}
}

func newAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Run migrations, then start admin UI + API and scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runMigrate(); err != nil {
				return err
			}
			runMonolith()
			return nil
		},
	}
}

func runMigrate() error {
	app := fx.New(
		config.Module,
		observability.Module,
		fx.Provide(registerSnowflake),
		db.Module,
		migration.Module,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		return fmt.Errorf("migrate failed: %w", err)
	}
	_ = app.Stop(context.Background())
	return nil
}

func runServe() {
	app := fx.New(
		observability.Module,
		fx.Provide(registerSnowflake),
		db.Module,
		clock.Module,
		redis.Module,
		bootstrap.Module,
		fx.Invoke(bootstrap.EnforceSchemaGate),
		fx.Invoke(bootstrap.EnsureDefaultOrgAndUser),
		scheduler.Module,
		server.Module,
		fx.Invoke(startScheduler),
	)
	app.Run()
}

func runScheduler() {
	app := fx.New(
		config.Module,
		observability.Module,
		fx.Provide(registerSnowflake),
		db.Module,
		clock.Module,
		bootstrap.Module,
		fx.Invoke(bootstrap.EnforceSchemaGate),
		scheduler.Module,
		rating.Module,
		invoice.Module,
		ledger.Module,
		subscription.Module,
		audit.Module,
		authorization.Module,
		billingoperations.Module,
		rollup.Module,
		product.Module,
		productfeature.Module,
		feature.Module,
		price.Module,
		priceamount.Module,
		pricetier.Module,
		invoicetemplate.Module,
		meter.Module,
		fx.Invoke(startScheduler),
	)
	app.Run()
}

func runMonolith() {
	app := fx.New(
		observability.Module,
		fx.Provide(registerSnowflake),
		db.Module,
		clock.Module,
		redis.Module,
		bootstrap.Module,
		fx.Invoke(bootstrap.EnforceSchemaGate),
		fx.Invoke(bootstrap.EnsureDefaultOrgAndUser),
		server.Module,
		scheduler.Module,
	)
	app.Run()
}

func registerSnowflake() *snowflake.Node {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	return node
}

func readVersionFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("APP_VERSION")); v != "" {
		return v
	}
	return "dev"
}

func startScheduler(lc fx.Lifecycle, s *scheduler.Scheduler) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go s.RunForever(context.Background())
			return nil
		},
	})
}
