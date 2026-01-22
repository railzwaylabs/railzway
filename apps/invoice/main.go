package main

import (
	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/config"
	"github.com/smallbiznis/railzway/internal/observability"
	"github.com/smallbiznis/railzway/internal/payment"
	paymentprovider "github.com/smallbiznis/railzway/internal/providers/payment"
	"github.com/smallbiznis/railzway/internal/publicinvoice"
	"github.com/smallbiznis/railzway/internal/ratelimit"
	"github.com/smallbiznis/railzway/internal/server"
	"github.com/smallbiznis/railzway/pkg/db"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		observability.Module,
		fx.Provide(RegisterSnowflake),
		db.Module,

		// Invoice Service dependencies
		publicinvoice.Module,
		payment.Module,         // For payment intent creation
		paymentprovider.Module, // For interfacing with Stripe/etc
		ratelimit.Module,       // Public endpoints are rate limited

		fx.Provide(server.NewEngine),
		fx.Provide(server.NewServer),
		fx.Invoke(func(s *server.Server) {
			s.RegisterPublicRoutes()
			s.RegisterFallback() // If this service also serves a public UI
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
