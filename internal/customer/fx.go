package customer

import (
	"github.com/railzwaylabs/railzway/internal/customer/repository"
	"github.com/railzwaylabs/railzway/internal/customer/service"
	"go.uber.org/fx"
)

var Module = fx.Module("customer.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
