package publicinvoice

import (
	"github.com/smallbiznis/railzway/internal/publicinvoice/repository"
	"github.com/smallbiznis/railzway/internal/publicinvoice/service"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"publicinvoice",
	fx.Provide(repository.Provide),
	fx.Provide(repository.ProvideTokenRepository),
	fx.Provide(service.New),
	fx.Provide(service.NewTokenService),
)
