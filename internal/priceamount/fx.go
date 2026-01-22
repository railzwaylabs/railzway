package priceamount

import (
	"github.com/smallbiznis/railzway/internal/priceamount/repository"
	"github.com/smallbiznis/railzway/internal/priceamount/service"
	"go.uber.org/fx"
)

var Module = fx.Module("priceamount.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
