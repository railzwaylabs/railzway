package invoicetemplate

import (
	"github.com/smallbiznis/railzway/internal/invoicetemplate/repository"
	"github.com/smallbiznis/railzway/internal/invoicetemplate/service"
	"go.uber.org/fx"
)

var Module = fx.Module("invoicetemplate.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.NewService),
)
