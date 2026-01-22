package customer

import (
	"github.com/smallbiznis/railzway/internal/customer/repository"
	"github.com/smallbiznis/railzway/internal/customer/service"
	"go.uber.org/fx"
)

var Module = fx.Module("customer.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
