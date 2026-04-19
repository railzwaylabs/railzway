package genkit

import "go.uber.org/fx"

// Module provides Genkit runtime initialization.
var Module = fx.Module("genkit",
	fx.Provide(NewGenkit),
)
