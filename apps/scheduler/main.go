package main

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/audit"
	"github.com/smallbiznis/railzway/internal/authorization"
	"github.com/smallbiznis/railzway/internal/billingdashboard/rollup"
	"github.com/smallbiznis/railzway/internal/billingoperations"
	"github.com/smallbiznis/railzway/internal/clock"
	"github.com/smallbiznis/railzway/internal/config"
	"github.com/smallbiznis/railzway/internal/feature"
	"github.com/smallbiznis/railzway/internal/invoice"
	"github.com/smallbiznis/railzway/internal/invoicetemplate"
	"github.com/smallbiznis/railzway/internal/ledger"
	"github.com/smallbiznis/railzway/internal/meter"
	"github.com/smallbiznis/railzway/internal/observability"
	"github.com/smallbiznis/railzway/internal/price"
	"github.com/smallbiznis/railzway/internal/priceamount"
	"github.com/smallbiznis/railzway/internal/pricetier"
	"github.com/smallbiznis/railzway/internal/product"
	"github.com/smallbiznis/railzway/internal/productfeature"
	"github.com/smallbiznis/railzway/internal/rating"
	"github.com/smallbiznis/railzway/internal/scheduler"
	"github.com/smallbiznis/railzway/internal/subscription"
	"github.com/smallbiznis/railzway/pkg/db"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		observability.Module,
		fx.Provide(RegisterSnowflake),
		db.Module,
		clock.Module,

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
