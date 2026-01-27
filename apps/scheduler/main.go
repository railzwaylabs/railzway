package main

import (
	"context"

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
	"github.com/railzwaylabs/railzway/internal/observability"
	"github.com/railzwaylabs/railzway/internal/price"
	"github.com/railzwaylabs/railzway/internal/priceamount"
	"github.com/railzwaylabs/railzway/internal/pricetier"
	"github.com/railzwaylabs/railzway/internal/product"
	"github.com/railzwaylabs/railzway/internal/productfeature"
	"github.com/railzwaylabs/railzway/internal/rating"
	"github.com/railzwaylabs/railzway/internal/scheduler"
	"github.com/railzwaylabs/railzway/internal/subscription"
	"github.com/railzwaylabs/railzway/pkg/db"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		observability.Module,
		fx.Provide(RegisterSnowflake),
		db.Module,
		clock.Module,
		bootstrap.Module,
		fx.Invoke(bootstrap.EnforceSchemaGate),

		// Domain services required by scheduler
		scheduler.Module,
		rating.Module,
		invoice.Module,
		ledger.Module,
		subscription.Module,
		audit.Module,
		authorization.Module,
		billingoperations.Module,
		rollup.Module,

		// Transitive dependencies (invoice needs product/price etc)
		product.Module,
		productfeature.Module,
		feature.Module,
		price.Module,
		priceamount.Module,
		pricetier.Module,
		invoicetemplate.Module,
		meter.Module,

		// No server module!
		fx.Invoke(StartScheduler),
	)
	app.Run()
}

func RegisterSnowflake() *snowflake.Node {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	return node
}

func StartScheduler(lc fx.Lifecycle, s *scheduler.Scheduler) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go s.RunForever(context.Background())
			return nil
		},
	})
}
