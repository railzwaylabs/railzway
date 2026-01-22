package apikey

import (
	"github.com/smallbiznis/railzway/internal/apikey/repository"
	"github.com/smallbiznis/railzway/internal/apikey/service"
	"go.uber.org/fx"
)

var Module = fx.Module("apikey.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
