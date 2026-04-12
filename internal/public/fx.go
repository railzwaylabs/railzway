package public

import (
	appsdomain "github.com/railzwaylabs/railzway/internal/apps/domain"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/config"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	publichandler "github.com/railzwaylabs/railzway/internal/public/transport/http"
	"go.uber.org/fx"
)

// Module provides the invoice-viewer handler (unauthenticated /public/* routes).
var Module = fx.Module("public",
	fx.Provide(
		func(
			cfg *config.Config,
			invoices invoicedomain.Service,
			customers customerdomain.Service,
			orgs organizationdomain.Service,
			apps appsdomain.Service,
			audit *auditlog.Service,
		) *publichandler.Handler {
			return publichandler.NewHandler(cfg, invoices, customers, orgs, apps, audit)
		},
	),
)
