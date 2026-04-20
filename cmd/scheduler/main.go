package main

import (
	"context"
	"os"

	aimodule "github.com/railzwaylabs/railzway/internal/ai"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/coupon"
	"github.com/railzwaylabs/railzway/internal/db"
	"github.com/railzwaylabs/railzway/internal/invoice"
	"github.com/railzwaylabs/railzway/internal/ledger"
	"github.com/railzwaylabs/railzway/internal/plan"
	planrepo "github.com/railzwaylabs/railzway/internal/plan/repository"
	"github.com/railzwaylabs/railzway/internal/rating"
	"github.com/railzwaylabs/railzway/internal/reconciliation"
	"github.com/railzwaylabs/railzway/internal/subscription"
	subscriptionrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	"github.com/railzwaylabs/railzway/internal/testclock"
	"github.com/railzwaylabs/railzway/internal/usage"
	usagerepo "github.com/railzwaylabs/railzway/internal/usage/repository"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Railzway scheduler (runs all jobs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			buildAllApp().Run()
			return nil
		},
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "rating",
		Short: "Run only rating scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			buildRatingApp().Run()
			return nil
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "close-period",
		Short: "Run only subscription close period scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			buildClosePeriodApp().Run()
			return nil
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "ai-worker",
		Short: "Run only AI Assistant job worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			buildAIWorkerApp().Run()
			return nil
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg.AppEnv.IsProduction() {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func registerLoggerLifecycle(lc fx.Lifecycle, logger *zap.Logger) {
	zap.ReplaceGlobals(logger)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return logger.Sync()
		},
	})
}

func buildAllApp() *fx.App {
	return fx.New(
		fx.Provide(
			config.Register,
			newLogger,
		),
		db.Module,
		clock.Module,
		testclock.Module,
		plan.Module,
		subscription.Module,
		usage.Module,
		rating.Module,
		coupon.Module,
		invoice.Module,
		ledger.Module,
		reconciliation.Module,
		fx.Invoke(registerLoggerLifecycle),
		fx.Invoke(telemetry.StartProfiler(6060)),
	)
}

func buildRatingApp() *fx.App {
	return fx.New(
		fx.Provide(
			config.Register,
			newLogger,
			planrepo.NewRepository,
			subscriptionrepo.NewRepository,
			usagerepo.NewRepository,
		),
		db.Module,
		coupon.Module,
		invoice.Module,
		ledger.Module,
		rating.Module,
		fx.Invoke(registerLoggerLifecycle),
		fx.Invoke(telemetry.StartProfiler(6060)),
	)
}

func buildClosePeriodApp() *fx.App {
	return fx.New(
		fx.Provide(
			config.Register,
			newLogger,
		),
		db.Module,
		clock.Module,
		testclock.Module,
		plan.Module,
		subscription.Module,
		coupon.Module,
		invoice.Module,
		ledger.Module,
		fx.Invoke(registerLoggerLifecycle),
		fx.Invoke(telemetry.StartProfiler(6060)),
	)
}

func buildAIWorkerApp() *fx.App {
	return fx.New(
		fx.Provide(
			config.Register,
			newLogger,
		),
		db.Module,
		clock.Module,
		aimodule.Module,
		fx.Invoke(registerLoggerLifecycle),
		fx.Invoke(telemetry.StartProfiler(6060)),
	)
}
