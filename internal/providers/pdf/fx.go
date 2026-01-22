package pdf

import "go.uber.org/fx"

var Module = fx.Module("providers.pdf",
	fx.Provide(New),
)
