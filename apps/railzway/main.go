package main

import (
	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/migration"
	"github.com/railzwaylabs/railzway/internal/observability"
	"github.com/railzwaylabs/railzway/internal/redis"
	"github.com/railzwaylabs/railzway/internal/scheduler"
	"github.com/railzwaylabs/railzway/internal/server"
	"github.com/railzwaylabs/railzway/pkg/db"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		// Core Infrastructure
		// config.Module,
		observability.Module,
		fx.Provide(RegisterSnowflake),
		db.Module,
		clock.Module,
		redis.Module,
		server.Module,

		// Functional Domains
		scheduler.Module,
		migration.Module,

		// All other domain modules usually imported by specific apps
		// We can mostly rely on server.Module importing them transitively or explicitly here
		// but server.Module already imports MOST of them.

		// server.Module now invokes RegisterRoutes automatically.

		// RunHTTP is invoked by server.Module or explicitly?
		// server.Module has fx.Invoke(RunHTTP).
		// We can leave it or be explicit.
		// To be safe, let's Suppress server.Module's autodrive if needed, or just let it run.
		// But server.Module defines `fx.Invoke(RunHTTP)` at line 125.
		// So it will run automatically.
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
