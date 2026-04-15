package admin

import (
	"github.com/railzwaylabs/railzway/internal/admin/auth"
	"github.com/railzwaylabs/railzway/internal/admin/scheduler"
	adminservice "github.com/railzwaylabs/railzway/internal/admin/service"
	adminhandler "github.com/railzwaylabs/railzway/internal/admin/transport/http"
	"github.com/railzwaylabs/railzway/internal/aiassistant"
	aiassistantdomain "github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	"github.com/railzwaylabs/railzway/internal/aiworkflow"
	aiworkflowdomain "github.com/railzwaylabs/railzway/internal/aiworkflow/domain"
	apikeyrepo "github.com/railzwaylabs/railzway/internal/apikey/repository"
	apikeyservice "github.com/railzwaylabs/railzway/internal/apikey/service"
	appsdomain "github.com/railzwaylabs/railzway/internal/apps/domain"
	appsrepo "github.com/railzwaylabs/railzway/internal/apps/repository"
	appsservice "github.com/railzwaylabs/railzway/internal/apps/service"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/authz"
	"github.com/railzwaylabs/railzway/internal/config"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	featureflagsvc "github.com/railzwaylabs/railzway/internal/featureflag/service"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	ledgerdomain "github.com/railzwaylabs/railzway/internal/ledger/domain"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	referencedomain "github.com/railzwaylabs/railzway/internal/reference/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	taxdomain "github.com/railzwaylabs/railzway/internal/tax/domain"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
)

// Module provides admin summary services and handlers.
var Module = fx.Module("admin",
	aiassistant.Module,
	aiworkflow.Module,
	fx.Provide(
		auth.NewService,
		auditlog.NewService,
		appsrepo.NewRepository,
		appsservice.NewService,
		apikeyrepo.NewRepository,
		apikeyservice.NewService,
		authz.NewAdminAuthorizer,
		func(s *appsservice.Service) appsdomain.Service { return s },
		adminservice.NewService,
		func(
			summary *adminservice.Service,
			flags *featureflagsvc.Service,
			authSvc *auth.Service,
			adminAuthz *authz.AdminAuthorizer,
			auditSvc *auditlog.Service,
			cfg *config.Config,
			apps appsdomain.Service,
			apiKeys *apikeyservice.Service,
			plans plandomain.Service,
			products productdomain.Service,
			features featuredomain.Service,
			productFeatures productfeaturedomain.Service,
			customers customerdomain.Service,
			organizations organizationdomain.Service,
			subscriptions subscriptiondomain.Service,
			usage usagedomain.Service,
			invoices invoicedomain.Service,
			ledger ledgerdomain.Service,
			rating ratingdomain.Service,
			payments paymentdomain.Service,
			taxes taxdomain.Service,
			testclocks testclockdomain.Service,
			references referencedomain.Repository,
			aiAssistant aiassistantdomain.Service,
			aiWorkflow aiworkflowdomain.Service,
		) *adminhandler.Handler {
			return adminhandler.NewHandler(summary, flags, authSvc, adminAuthz, auditSvc, cfg, apps, apiKeys, plans, products, features, productFeatures, customers, organizations, subscriptions, usage, invoices, ledger, rating, payments, taxes, testclocks, references, aiAssistant, aiWorkflow)
		},
	),
	fx.Invoke(scheduler.StartSummaryRefresher),
)
