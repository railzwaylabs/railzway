package auth

import (
	authconfig "github.com/railzwaylabs/railzway/internal/auth/config"
	"github.com/railzwaylabs/railzway/internal/auth/oauth"
	"github.com/railzwaylabs/railzway/internal/auth/repository"
	"github.com/railzwaylabs/railzway/internal/auth/service"
	"go.uber.org/fx"
)

var Module = fx.Module("auth.service",
	fx.Provide(repository.New),
	fx.Provide(service.New),
	fx.Provide(oauth.NewService),
	fx.Provide(authconfig.ParseAuthProvidersFromEnv),
	fx.Provide(authconfig.BuildAuthProviderRegistry),
	fx.Invoke(ensureAuthProviderRegistry),
)

func ensureAuthProviderRegistry(_ authconfig.AuthProviderRegistry) {}
