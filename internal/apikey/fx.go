package apikey

import (
	"github.com/railzwaylabs/railzway/internal/apikey/repository"
	"github.com/railzwaylabs/railzway/internal/apikey/service"
	"go.uber.org/fx"
)

var Module = fx.Module("apikey.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
