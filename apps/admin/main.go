package main

import (
	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/apikey"
	"github.com/railzwaylabs/railzway/internal/audit"
	"github.com/railzwaylabs/railzway/internal/auth"
	authlocal "github.com/railzwaylabs/railzway/internal/auth/local"
	authoauth2provider "github.com/railzwaylabs/railzway/internal/auth/oauth2provider"
	"github.com/railzwaylabs/railzway/internal/auth/session"
	"github.com/railzwaylabs/railzway/internal/authorization"
	"github.com/railzwaylabs/railzway/internal/billingdashboard"
	"github.com/railzwaylabs/railzway/internal/billingoperations"
	"github.com/railzwaylabs/railzway/internal/billingoverview"
	"github.com/railzwaylabs/railzway/internal/bootstrap"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/cloudmetrics"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/customer"
	"github.com/railzwaylabs/railzway/internal/events"
	"github.com/railzwaylabs/railzway/internal/feature"
	"github.com/railzwaylabs/railzway/internal/invoice"
	"github.com/railzwaylabs/railzway/internal/invoicetemplate"
	"github.com/railzwaylabs/railzway/internal/ledger"
	"github.com/railzwaylabs/railzway/internal/meter"
	"github.com/railzwaylabs/railzway/internal/observability"
	"github.com/railzwaylabs/railzway/internal/organization"
	"github.com/railzwaylabs/railzway/internal/payment"
	"github.com/railzwaylabs/railzway/internal/price"
	"github.com/railzwaylabs/railzway/internal/priceamount"
	"github.com/railzwaylabs/railzway/internal/pricetier"
	"github.com/railzwaylabs/railzway/internal/product"
	"github.com/railzwaylabs/railzway/internal/productfeature"
	"github.com/railzwaylabs/railzway/internal/providers/email"
	paymentprovider "github.com/railzwaylabs/railzway/internal/providers/payment"
	"github.com/railzwaylabs/railzway/internal/providers/pdf"
	"github.com/railzwaylabs/railzway/internal/ratelimit"
	"github.com/railzwaylabs/railzway/internal/rating"
	"github.com/railzwaylabs/railzway/internal/reference"
	"github.com/railzwaylabs/railzway/internal/server"
	"github.com/railzwaylabs/railzway/internal/subscription"
	"github.com/railzwaylabs/railzway/internal/usage"
	"github.com/railzwaylabs/railzway/pkg/db"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		cloudmetrics.Module,
		observability.Module,
		fx.Provide(RegisterSnowflake),
		db.Module,
		clock.Module,
		bootstrap.Module,
		fx.Invoke(bootstrap.EnforceSchemaGate),
		fx.Invoke(bootstrap.EnsureDefaultOrgAndUser),

		// Admin needs almost everything
		authorization.Module,
		audit.Module,
		events.Module,
		auth.Module,
		authlocal.Module,
		authoauth2provider.Module,
		session.Module,
		apikey.Module,
		customer.Module,
		billingdashboard.Module,
		billingoperations.Module,
		email.Module,
		pdf.Module,
		billingoverview.Module,
		invoice.Module,
		invoicetemplate.Module,
		ledger.Module,
		meter.Module,
		organization.Module,
		price.Module,
		priceamount.Module,
		pricetier.Module,
		product.Module,
		productfeature.Module,
		feature.Module,
		payment.Module,
		paymentprovider.Module,
		reference.Module,
		rating.Module,
		ratelimit.Module,
		subscription.Module,
		usage.Module, // Needed for dashboard stats, but maybe not ingestion

		fx.Provide(server.NewEngine),
		fx.Provide(server.NewServer),
		fx.Invoke(func(s *server.Server) {
			s.RegisterAuthRoutes()
			s.RegisterAdminRoutes()
			s.RegisterUIRoutes() // Monolith style: serve the react app
			s.RegisterFallback()
		}),
		fx.Invoke(server.RunHTTP),
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
