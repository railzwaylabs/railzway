// @title           Railzway API
// @version         1.0
// @description     Railzway Billing & Operations API
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@railzway.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api
// @Schemes 	http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

package main

import (
	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/apikey"
	"github.com/railzwaylabs/railzway/internal/auth"
	"github.com/railzwaylabs/railzway/internal/bootstrap"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/meter"
	"github.com/railzwaylabs/railzway/internal/observability"
	"github.com/railzwaylabs/railzway/internal/ratelimit"
	"github.com/railzwaylabs/railzway/internal/server"
	"github.com/railzwaylabs/railzway/internal/usage"
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

		// Core dependencies for API
		auth.Module,   // For API Key validation logic
		apikey.Module, // For API Key domain
		meter.Module,
		usage.Module,
		ratelimit.Module,

		fx.Provide(server.NewEngine),
		fx.Provide(server.NewServer),
		fx.Invoke(func(s *server.Server) {
			s.RegisterAPIRoutes()
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
