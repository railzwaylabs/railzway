package billingprovisioning

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

const pollInterval = 5 * time.Second

var Module = fx.Module("billing.provisioning",
	fx.Provide(NewConsumer),
	fx.Invoke(runConsumer),
)

func runConsumer(lc fx.Lifecycle, consumer *Consumer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				ticker := time.NewTicker(pollInterval)
				defer ticker.Stop()

				for {
					if err := consumer.ProcessPending(context.Background()); err != nil {
						consumer.log.Error("provisioning poll failed", zap.Error(err))
					}
					<-ticker.C
				}
			}()
			return nil
		},
	})
}
