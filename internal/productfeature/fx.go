package productfeature

import (
	"github.com/railzwaylabs/railzway/internal/productfeature/repository"
	"github.com/railzwaylabs/railzway/internal/productfeature/service"
	"go.uber.org/fx"
)

var Module = fx.Module("productfeature",
	fx.Provide(
		repository.NewRepository,
		service.New,
	),
)
