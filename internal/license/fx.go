package license

import (
	"go.uber.org/fx"
)

var Module = fx.Module("license",
	fx.Provide(NewService),
)
