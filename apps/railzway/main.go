// @title           Railzway API
// @version         1.0
// @description     Railzway Billing & Operations API
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@railzway.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      railzway.com/docs
// @BasePath  /api
// @Schemes 	http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

package main

import (
	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/bootstrap"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/observability"
	"github.com/railzwaylabs/railzway/internal/redis"
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
		bootstrap.Module,
		fx.Invoke(bootstrap.EnforceSchemaGate),
		fx.Invoke(bootstrap.EnsureDefaultOrgAndUser),
		server.Module,

		// Functional Domains
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
