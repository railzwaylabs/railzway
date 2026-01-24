package ledger

import (
	"github.com/railzwaylabs/railzway/internal/ledger/service"
	"go.uber.org/fx"
)

var Module = fx.Module("ledger.service",
	fx.Provide(service.NewService),
)
