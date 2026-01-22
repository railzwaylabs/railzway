package productfeature

import (
	"github.com/smallbiznis/railzway/internal/productfeature/repository"
	"github.com/smallbiznis/railzway/internal/productfeature/service"
	"go.uber.org/fx"
)

var Module = fx.Module("productfeature.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
