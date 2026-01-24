package payment

import (
	"github.com/railzwaylabs/railzway/internal/providers/payment/repository"
	"github.com/railzwaylabs/railzway/internal/providers/payment/service"
	"go.uber.org/fx"
)

var Module = fx.Module("paymentprovider.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
