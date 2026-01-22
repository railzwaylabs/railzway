package snapshot

import (
	"context"

	"go.uber.org/fx"
)

var Module = fx.Module("usage.snapshot",
	fx.Provide(DefaultConfig),
	fx.Provide(NewWorker),
	fx.Invoke(runWorker),
)

func runWorker(lc fx.Lifecycle, worker *Worker) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithCancel(context.Background())

			go worker.RunForever(ctx)

			lc.Append(fx.Hook{
				OnStop: func(context.Context) error {
					cancel()
					return nil
				},
			})

			return nil
		},
	})
}
