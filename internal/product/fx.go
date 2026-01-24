package product

import (
	"github.com/railzwaylabs/railzway/internal/product/repository"
	"github.com/railzwaylabs/railzway/internal/product/service"
	"go.uber.org/fx"
)

var Module = fx.Module("product.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
