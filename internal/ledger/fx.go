package ledger

import (
	"github.com/railzwaylabs/railzway/internal/ledger/repository"
	"github.com/railzwaylabs/railzway/internal/ledger/service"
	"go.uber.org/fx"
)

// Module provides ledger repository and service wiring.
var Module = fx.Module("ledger",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
