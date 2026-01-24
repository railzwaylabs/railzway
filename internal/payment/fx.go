package payment

import (
	"github.com/railzwaylabs/railzway/internal/payment/adapters"
	"github.com/railzwaylabs/railzway/internal/payment/adapters/adyen"
	"github.com/railzwaylabs/railzway/internal/payment/adapters/braintree"
	"github.com/railzwaylabs/railzway/internal/payment/adapters/stripe"
	"github.com/railzwaylabs/railzway/internal/payment/adapters/xendit"
	disputerepo "github.com/railzwaylabs/railzway/internal/payment/dispute/repository"
	disputeservice "github.com/railzwaylabs/railzway/internal/payment/dispute/service"
	"github.com/railzwaylabs/railzway/internal/payment/repository"
	paymentservice "github.com/railzwaylabs/railzway/internal/payment/service"
	"github.com/railzwaylabs/railzway/internal/payment/webhook"
	"go.uber.org/fx"
)

var Module = fx.Module("payment.service",
	fx.Provide(repository.Provide),
	fx.Provide(disputerepo.Provide),
	fx.Provide(func() *adapters.Registry {
		return adapters.NewRegistry(
			stripe.NewFactory(),
			adyen.NewFactory(),
			braintree.NewFactory(),
			xendit.NewFactory(),
		)
	}),
	fx.Provide(paymentservice.NewService),
	fx.Provide(disputeservice.NewService),
	fx.Provide(webhook.NewService),
	fx.Provide(paymentservice.NewPaymentMethodService),
	fx.Provide(paymentservice.NewPaymentMethodConfigService),
)
