import { lazy, Suspense, useEffect, useState, type ReactElement, type ReactNode } from "react"
import { Navigate, Outlet, createBrowserRouter, useLocation } from "react-router-dom"

import DashboardLayout from "@/layouts/DashboardLayout"
import OrgResolverPage from "@/pages/orgs"
import ChangePasswordPage from "@/pages/change-password"
import LoginPage from "@/pages/login"
import OnboardingPage from "@/pages/onboarding"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuthStore } from "@/stores/authStore"
import { EnterpriseErrorBoundary } from "@/components/EnterpriseErrorBoundary"

const AcceptInvitePage = lazy(() => import("@/pages/accept-invite"))

const OrgHome = lazy(() => import("@/features/billing/pages/OrgHome"))
const OrgBillingOverviewPage = lazy(
  () => import("@/features/billing/pages/OrgBillingOverviewPage")
)
const OrgBillingOperationsPage = lazy(
  () => import("@/features/billing/pages/OrgBillingOperationsPage")
)
const OrgCustomersPage = lazy(() => import("@/features/billing/pages/OrgCustomersPage"))
const OrgCustomerDetailPage = lazy(
  () => import("@/features/billing/pages/OrgCustomerDetailPage")
)
const OrgSubscriptionsPage = lazy(() => import("@/features/billing/pages/OrgSubscriptionsPage"))
const OrgSubscriptionDetailPage = lazy(
  () => import("@/features/billing/pages/OrgSubscriptionDetailPage")
)
const OrgSubscriptionCreatePage = lazy(
  () => import("@/features/billing/pages/OrgSubscriptionCreatePage")
)
const OrgSettings = lazy(() => import("@/features/billing/pages/OrgSettings"))
const OrgPaymentProvidersPage = lazy(
  () => import("@/features/billing/pages/OrgPaymentProvidersPage")
)
const OrgPaymentMethodConfigsPage = lazy(
  () => import("@/features/payment/pages/OrgPaymentMethodConfigsPage")
)

const OrgApiKeysPage = lazy(() => import("@/features/guard/pages/OrgApiKeysPage"))
const OrgAuditLogsPage = lazy(() => import("@/features/guard/pages/OrgAuditLogsPage"))

const OrgMeterPage = lazy(() => import("@/features/usage/pages/OrgMeterPage"))
const OrgMeterCreatePage = lazy(() => import("@/features/usage/pages/OrgMeterCreatePage"))
const OrgMeterDetailPage = lazy(() => import("@/features/usage/pages/OrgMeterDetailPage"))

const OrgProductsPage = lazy(() => import("@/features/pricing/pages/OrgProductsPage"))
const CreateProduct = lazy(() => import("@/features/pricing/pages/CreateProduct"))
const CreatePrice = lazy(() => import("@/features/pricing/pages/CreatePrice"))
const OrgProductDetailPage = lazy(
  () => import("@/features/pricing/pages/OrgProductDetailPage")
)
const OrgPricesPage = lazy(() => import("@/features/pricing/pages/OrgPricesPage"))
const OrgPricingsPage = lazy(() => import("@/features/pricing/pages/OrgPricingsPage"))
const OrgPricingDetailPage = lazy(
  () => import("@/features/pricing/pages/OrgPricingDetailPage")
)
const OrgPriceAmountsPage = lazy(
  () => import("@/features/pricing/pages/OrgPriceAmountsPage")
)
const OrgPriceAmountDetailPage = lazy(
  () => import("@/features/pricing/pages/OrgPriceAmountDetailPage")
)
const OrgPriceTiersPage = lazy(
  () => import("@/features/pricing/pages/OrgPriceTiersPage")
)
const OrgPriceTierDetailPage = lazy(
  () => import("@/features/pricing/pages/OrgPriceTierDetailPage")
)

const OrgInvoicesPage = lazy(() => import("@/features/invoice/pages/OrgInvoicesPage"))
const OrgInvoiceDetailPage = lazy(
  () => import("@/features/invoice/pages/OrgInvoiceDetailPage")
)
const OrgInvoiceTemplatesPage = lazy(
  () => import("@/features/invoice/pages/OrgInvoiceTemplatesPage")
)
const OrgInvoiceTemplateFormPage = lazy(
  () => import("@/features/invoice/pages/OrgInvoiceTemplateFormPage")
)

const AdminFeaturesPage = lazy(
  () => import("@/features/admin/catalog/pages/AdminFeaturesPage")
)
const AdminTaxDefinitionsPage = lazy(
  () => import("@/features/admin/catalog/pages/AdminTaxDefinitionsPage")
)

const LicensePage = lazy(() => import("@/features/settings/pages/LicensePage"))

// eslint-disable-next-line react-refresh/only-export-components
// eslint-disable-next-line react-refresh/only-export-components
function RequireAuth() {
  const { isAuthenticated, mustChangePassword, refreshSession } = useAuthStore()
  const location = useLocation()

  const [isHydrated, setIsHydrated] = useState(useAuthStore.persist.hasHydrated())
  // Initialize checking state based on authentication status to prevent premature redirects
  const [isChecking, setIsChecking] = useState(!isAuthenticated)

  useEffect(() => {
    const unsubscribe = useAuthStore.persist.onFinishHydration(() => {
      setIsHydrated(true)
    })
    if (useAuthStore.persist.hasHydrated()) {
      setIsHydrated(true)
    }
    return unsubscribe
  }, [])

  useEffect(() => {
    // If not hydrated yet, or if already authenticated, we stop checking/loading
    if (!isHydrated || isAuthenticated) {
      if (isChecking) setIsChecking(false)
      return
    }

    // If not authenticated and hydrated, we must check the session ONCE
    // This handles the "refresh on reload" scenario
    let active = true
    setIsChecking(true)

    refreshSession()
      .catch(() => {
        // Validation failed, user remains unauthenticated
      })
      .finally(() => {
        if (active) setIsChecking(false)
      })

    return () => { active = false }
  }, [isHydrated, isAuthenticated, refreshSession])

  if (!isHydrated || isChecking) {
    return <div className="flex min-h-screen items-center justify-center text-text-muted text-sm">Loading session...</div>
  }

  if (!isAuthenticated) {
    const target = encodeURIComponent(`${location.pathname}${location.search}`)
    return <Navigate to={`/login?redirectTo=${target}`} replace />
  }

  if (mustChangePassword && location.pathname !== "/change-password") {
    return <Navigate to="/change-password" replace />
  }

  return <Outlet />
}

function RouteSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-7 w-48" />
      <Skeleton className="h-4 w-72" />
      <Skeleton className="h-64 w-full" />
    </div>
  )
}

function FeatureBoundary({ children }: { children: ReactNode }) {
  return <Suspense fallback={<RouteSkeleton />}>{children}</Suspense>
}

const withFeatureBoundary = (node: ReactElement) => (
  <FeatureBoundary>{node}</FeatureBoundary>
)

export const router = createBrowserRouter([
  {
    path: "/",
    element: <Navigate to="/orgs" replace />,
  },
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/signup",
    element: <Navigate to="/login" replace />,
  },
  {
    path: "/invite/:inviteId",
    element: (
      <Suspense fallback={<Skeleton className="h-screen w-screen" />}>
        <AcceptInvitePage />
      </Suspense>
    ),
  },
  {
    element: <RequireAuth />,
    children: [
      { path: "/change-password", element: <ChangePasswordPage /> },
      { path: "/orgs", element: <OrgResolverPage /> },
      { path: "/onboarding", element: <OnboardingPage /> },
      {
        path: "/orgs/:orgId",
        element: <DashboardLayout />,
        children: [
          { index: true, element: <Navigate to="home" replace /> },
          { path: "home", element: withFeatureBoundary(<OrgHome />) },
          {
            path: "billing/overview",
            element: withFeatureBoundary(<OrgBillingOverviewPage />),
          },
          {
            path: "billing/operations",
            element: withFeatureBoundary(<OrgBillingOperationsPage />),
            errorElement: <EnterpriseErrorBoundary />,
          },
          {
            path: "products",
            element: withFeatureBoundary(<OrgProductsPage />),
          },
          {
            path: "products/create",
            element: withFeatureBoundary(<CreateProduct />),
          },
          {
            path: "products/:productId",
            element: withFeatureBoundary(<OrgProductDetailPage />),
          },
          {
            path: "products/:productId/prices/create",
            element: withFeatureBoundary(<CreatePrice />),
          },
          {
            path: "products/features",
            element: withFeatureBoundary(<AdminFeaturesPage />),
          },
          {
            path: "products/tax-definitions",
            element: withFeatureBoundary(<AdminTaxDefinitionsPage />),
          },
          {
            path: "prices",
            element: withFeatureBoundary(<OrgPricesPage />),
          },
          {
            path: "pricings",
            element: withFeatureBoundary(<OrgPricingsPage />),
          },
          {
            path: "pricings/:pricingId",
            element: withFeatureBoundary(<OrgPricingDetailPage />),
          },
          {
            path: "price-amounts",
            element: withFeatureBoundary(<OrgPriceAmountsPage />),
          },
          {
            path: "price-amounts/:amountId",
            element: withFeatureBoundary(<OrgPriceAmountDetailPage />),
          },
          {
            path: "price-tiers",
            element: withFeatureBoundary(<OrgPriceTiersPage />),
          },
          {
            path: "price-tiers/:tierId",
            element: withFeatureBoundary(<OrgPriceTierDetailPage />),
          },
          {
            path: "meter",
            element: withFeatureBoundary(<OrgMeterPage />),
          },
          {
            path: "meter/create",
            element: withFeatureBoundary(<OrgMeterCreatePage />),
          },
          {
            path: "meter/:meterId",
            element: withFeatureBoundary(<OrgMeterDetailPage />),
          },
          {
            path: "api-keys",
            element: withFeatureBoundary(<OrgApiKeysPage />),
          },
          {
            path: "audit-logs",
            element: withFeatureBoundary(<OrgAuditLogsPage />),
            errorElement: <EnterpriseErrorBoundary />,
          },
          {
            path: "payment-providers",
            element: withFeatureBoundary(<OrgPaymentProvidersPage />),
          },
          {
            path: "payment-method-configs",
            element: withFeatureBoundary(<OrgPaymentMethodConfigsPage />),
          },
          {
            path: "customers",
            element: withFeatureBoundary(<OrgCustomersPage />),
          },
          {
            path: "customers/:customerId",
            element: withFeatureBoundary(<OrgCustomerDetailPage />),
          },
          {
            path: "subscriptions",
            element: withFeatureBoundary(<OrgSubscriptionsPage />),
          },
          {
            path: "subscriptions/:subscriptionId",
            element: withFeatureBoundary(<OrgSubscriptionDetailPage />),
          },
          {
            path: "subscriptions/create",
            element: withFeatureBoundary(<OrgSubscriptionCreatePage />),
          },
          {
            path: "invoices",
            element: withFeatureBoundary(<OrgInvoicesPage />),
          },
          {
            path: "invoices/:invoiceId",
            element: withFeatureBoundary(<OrgInvoiceDetailPage />),
          },
          {
            path: "invoice-templates",
            element: withFeatureBoundary(<OrgInvoiceTemplatesPage />),
          },
          {
            path: "invoice-templates/create",
            element: withFeatureBoundary(<OrgInvoiceTemplateFormPage />),
          },
          {
            path: "invoice-templates/:templateId",
            element: withFeatureBoundary(<OrgInvoiceTemplateFormPage />),
          },
          { path: "settings", element: withFeatureBoundary(<OrgSettings />), errorElement: <EnterpriseErrorBoundary /> },
          { path: "settings/license", element: withFeatureBoundary(<LicensePage />), errorElement: <EnterpriseErrorBoundary /> },
        ],
      },
    ],
  },
  { path: "*", element: <Navigate to="/orgs" replace /> },
])
