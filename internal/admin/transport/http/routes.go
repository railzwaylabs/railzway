package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, h *Handler) {
	registerAdminRoutes(r, h, "/admin/v1")
	// Backward-compat until all clients migrate to /admin/v1.
	registerAdminRoutes(r, h, "/admin")
}

func registerAdminRoutes(r *gin.Engine, h *Handler, basePath string) {
	admin := r.Group(basePath)
	admin.POST("/auth/login", h.Login)
	admin.POST("/auth/logout", h.Logout)
	admin.GET("/apps/oauth/stripe/callback", h.StripeOAuthCallback)

	protected := admin.Group("/")
	protected.Use(h.AuthRequired())
	protected.Use(h.RequireCSRF())
	protected.Use(h.InjectTestClock())
	protected.GET("/auth/me", h.Me)
	protected.POST("/auth/using/:org_id", h.SwitchOrganization)
	protected.PUT("/auth/change-password", h.ChangePassword)
	protected.POST("/auth/skip-password-change", h.SkipPasswordChange)
	protected.GET("/reference/countries", h.ListCountries)
	protected.GET("/reference/timezones", h.ListTimezones)
	protected.GET("/reference/currencies", h.ListCurrencies)

	// Organizations
	protected.POST("/organizations", h.CreateOrganization)
	protected.GET("/organizations", h.ListOrganizations)
	protected.POST("/organizations/invites/:invite_id/accept", h.AcceptOrganizationInvite)

	orgRequired := admin.Group("/")
	orgRequired.Use(h.AuthRequired())
	orgRequired.Use(h.RequireCSRF())
	orgRequired.Use(h.InjectTestClock())
	orgRequired.Use(h.RequireOrgHeader())
	orgRequired.Use(h.AuthorizeAdmin())

	orgRequired.GET("/organizations/:org_id", h.GetOrganization)
	orgRequired.PATCH("/organizations/:org_id", h.UpdateOrganization)
	orgRequired.GET("/organizations/:org_id/members", h.ListOrganizationMembers)
	orgRequired.POST("/organizations/:org_id/invites", h.InviteOrganizationMembers)
	orgRequired.PUT("/organizations/:org_id/billing-preferences", h.UpsertOrganizationBillingPreferences)
	orgRequired.POST("/organizations/:org_id/invoice-formats", h.CreateOrganizationInvoiceFormat)
	orgRequired.GET("/organizations/:org_id/invoice-formats", h.ListOrganizationInvoiceFormats)
	orgRequired.POST("/organizations/:org_id/invoice-formats/:format_id/close", h.CloseOrganizationInvoiceFormat)
	orgRequired.POST("/organizations/:org_id/links", h.LinkChildOrganization)

	// Plans
	orgRequired.POST("/plans", h.CreatePlan)
	orgRequired.GET("/plans", h.ListPlans)
	orgRequired.GET("/plans/:plan_id", h.GetPlan)
	orgRequired.PATCH("/plans/:plan_id", h.UpdatePlan)
	orgRequired.POST("/plans/:plan_id/prices", h.CreatePlanPrice)
	orgRequired.GET("/plans/:plan_id/prices", h.ListPlanPrices)

	// Prices
	orgRequired.GET("/prices/:price_id", h.GetPlanPrice)
	orgRequired.POST("/prices/:price_id/amounts", h.CreatePlanAmount)
	orgRequired.GET("/prices/:price_id/amounts", h.ListPlanAmounts)
	orgRequired.GET("/prices/amounts/:amount_id", h.GetPlanAmount)
	orgRequired.POST("/prices/:price_id/tiers", h.CreatePlanTier)
	orgRequired.GET("/prices/:price_id/tiers", h.ListPlanTiers)
	orgRequired.GET("/prices/tiers/:tier_id", h.GetPlanTier)

	// Products
	orgRequired.POST("/products", h.CreateProduct)
	orgRequired.GET("/products", h.ListProducts)
	orgRequired.GET("/products/:product_id", h.GetProduct)
	orgRequired.PATCH("/products/:product_id", h.UpdateProduct)

	// Features
	orgRequired.POST("/features", h.CreateFeature)
	orgRequired.GET("/features", h.ListFeatures)
	orgRequired.GET("/features/:feature_id", h.GetFeature)
	orgRequired.PATCH("/features/:feature_id", h.UpdateFeature)

	// Product Features
	orgRequired.GET("/products/:product_id/features", h.ListProductFeatures)
	orgRequired.PUT("/products/:product_id/features", h.ReplaceProductFeatures)

	// Customers
	orgRequired.POST("/customers", h.CreateCustomer)
	orgRequired.GET("/customers", h.ListCustomers)
	orgRequired.GET("/customers/:customer_id", h.GetCustomer)
	orgRequired.PATCH("/customers/:customer_id", h.UpdateCustomer)

	// Subscriptions
	orgRequired.POST("/subscriptions", h.CreateSubscription)
	orgRequired.GET("/subscriptions", h.ListSubscriptions)
	orgRequired.GET("/subscriptions/:subscription_id", h.GetSubscription)
	orgRequired.PATCH("/subscriptions/:subscription_id", h.UpdateSubscription)
	orgRequired.POST("/subscriptions/:subscription_id/items", h.CreateSubscriptionItem)
	orgRequired.GET("/subscriptions/:subscription_id/items", h.ListSubscriptionItems)
	orgRequired.GET("/subscription-items/:item_id", h.GetSubscriptionItem)

	// Meters
	orgRequired.POST("/meters", h.CreateMeter)
	orgRequired.GET("/meters", h.ListMeters)
	orgRequired.GET("/meters/:meter_id", h.GetMeter)
	orgRequired.PATCH("/meters/:meter_id", h.UpdateMeter)

	// Invoices
	orgRequired.GET("/invoices", h.ListInvoices)
	orgRequired.GET("/invoices/:invoice_id", h.GetInvoice)
	orgRequired.GET("/invoices/:invoice_id/public-link", h.GetInvoicePublicLink)
	orgRequired.POST("/invoices/generate", h.GenerateInvoice)
	orgRequired.POST("/invoices/:invoice_id/resend", h.ResendInvoice)
	orgRequired.POST("/invoices/:invoice_id/mark-paid", h.MarkInvoicePaid)
	orgRequired.POST("/invoices/:invoice_id/open", h.OpenInvoice)
	orgRequired.POST("/invoices/:invoice_id/pay", h.PayInvoice)
	orgRequired.POST("/invoices/:invoice_id/void", h.VoidInvoice)

	// Ledger
	orgRequired.POST("/ledger/accounts", h.CreateLedgerAccount)
	orgRequired.GET("/ledger/accounts", h.ListLedgerAccounts)
	orgRequired.GET("/ledger/transactions", h.ListLedgerTransactions)
	orgRequired.POST("/ledger/transactions", h.CreateLedgerTransaction)
	orgRequired.GET("/ledger/transactions/:transaction_id", h.GetLedgerTransaction)

	// Summary
	orgRequired.GET("/dashboard", h.DashboardSummary)
	orgRequired.GET("/customers/summary", h.CustomersSummary)
	orgRequired.GET("/plans/summary", h.PlansSummary)
	orgRequired.GET("/subscriptions/summary", h.SubscriptionsSummary)
	orgRequired.GET("/usage/summary", h.UsageSummary)
	orgRequired.GET("/usage/events", h.ListUsageEvents)
	orgRequired.POST("/usage/events", h.IngestUsage)
	orgRequired.GET("/rating/summary", h.RatingSummary)
	orgRequired.GET("/rating/results", h.ListRatingResults)
	orgRequired.GET("/rating/aggregates", h.ListUsageAggregates)
	orgRequired.POST("/rating/usage/:usage_event_id", h.RateUsageEvent)
	orgRequired.GET("/invoices/summary", h.InvoicesSummary)
	orgRequired.GET("/payments/summary", h.PaymentsSummary)
	orgRequired.GET("/payments", h.ListPayments)
	orgRequired.GET("/taxes/summary", h.TaxesSummary)
	orgRequired.POST("/taxes", h.CreateTaxRate)
	orgRequired.GET("/taxes", h.ListTaxes)
	orgRequired.GET("/audit-logs/summary", h.AuditLogsSummary)
	orgRequired.GET("/audit-logs", h.ListAuditLogs)
	orgRequired.GET("/settings/summary", h.SettingsSummary)
	orgRequired.GET("/api-keys", h.ListAPIKeys)
	orgRequired.POST("/api-keys", h.CreateAPIKey)
	orgRequired.POST("/api-keys/:id/revoke", h.RevokeAPIKey)

	// Test Clock
	orgRequired.GET("/test-clock", h.GetTestClock)
	orgRequired.POST("/test-clock", h.UpsertTestClock)
	orgRequired.POST("/test-clock/advance", h.AdvanceTestClock)
	orgRequired.POST("/test-clock/pause", h.PauseTestClock)
	orgRequired.POST("/test-clock/resume", h.ResumeTestClock)

	// Feature Flags
	orgRequired.GET("/feature-flags", h.ListFeatureFlags)
	orgRequired.POST("/feature-flags", h.UpsertFeatureFlag)

	// Apps
	orgRequired.GET("/apps/catalog", h.ListAppsCatalog)
	orgRequired.GET("/apps/installations", h.ListAppInstallations)
	orgRequired.POST("/apps/installations", h.InstallApp)
	orgRequired.PATCH("/apps/installations/:installation_id", h.UpdateAppInstallation)
	orgRequired.GET("/apps/oauth/stripe/start", h.StartStripeOAuth)

	// Misc
	orgRequired.GET("/warnings", h.ConfigWarnings)
	orgRequired.GET("/reconciliation/summary", h.ReconciliationSummary)
	orgRequired.GET("/authz/policies", h.ListAuthzPolicies)
	orgRequired.POST("/authz/policies", h.AddAuthzPolicy)
	orgRequired.DELETE("/authz/policies", h.RemoveAuthzPolicy)
}
