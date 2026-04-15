package api

import (
	apikeyservice "github.com/railzwaylabs/railzway/internal/apikey/service"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	apihandler "github.com/railzwaylabs/railzway/internal/api/transport/http"
	entitlementdomain "github.com/railzwaylabs/railzway/internal/entitlement/domain"
	"github.com/railzwaylabs/railzway/internal/ratelimit"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
)

// Module wires the public API handler (API key-authenticated endpoints).
var Module = fx.Module("api",
	fx.Provide(
		func(
			apiKeys *apikeyservice.Service,
			invoices invoicedomain.Service,
			customers customerdomain.Service,
			subscriptions subscriptiondomain.Service,
			usage usagedomain.Service,
			entitlement entitlementdomain.Service,
			rateLimiter *ratelimit.Limiter,
		) *apihandler.Handler {
			return apihandler.NewHandler(apiKeys, invoices, customers, subscriptions, usage, entitlement, rateLimiter)
		},
	),
)
