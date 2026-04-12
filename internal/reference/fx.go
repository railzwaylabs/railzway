package reference

import (
	"go.uber.org/fx"
)

// Module provides reference lookup repository (countries/timezones).
var Module = fx.Module("reference",
	fx.Provide(
		NewRepository,
	),
)
