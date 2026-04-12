package customerportal

import (
	customerhandler "github.com/railzwaylabs/railzway/internal/customerportal/transport/http"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"go.uber.org/fx"
)

var Module = fx.Module("customerportal",
	fx.Provide(
		func(subscriptions subscriptiondomain.Service, invoices invoicedomain.Service) *customerhandler.Handler {
			return customerhandler.NewHandler(subscriptions, invoices)
		},
	),
)
