import { BrowserRouter, NavLink, Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom"
import { lazy, Suspense, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import AuthGate from "./components/AuthGate"
import { Input } from "./components/ui/input"
import { ToastContainer, toast } from "./components/Toast"
import ConfigWarningsBanner from "./components/ConfigWarningsBanner"
import { api } from "./lib/api"
import { clearOrgId, getOrgId, isAuthRequired, setOrgId } from "./lib/auth"
import type { OrganizationListItem } from "./lib/types"

const Dashboard = lazy(() => import("./pages/Dashboard"))
const Customers = lazy(() => import("./pages/Customers"))
const CustomersCreate = lazy(() => import("./pages/CustomersCreate"))
const CustomersEdit = lazy(() => import("./pages/CustomersEdit"))
const OrganizationsCreate = lazy(() => import("./pages/OrganizationsCreate"))
const OrganizationsEdit = lazy(() => import("./pages/OrganizationsEdit"))
const Plans = lazy(() => import("./pages/Plans"))
const PlansCreate = lazy(() => import("./pages/PlansCreate"))
const PlansEdit = lazy(() => import("./pages/PlansEdit"))
const Subscriptions = lazy(() => import("./pages/Subscriptions"))
const SubscriptionsCreate = lazy(() => import("./pages/SubscriptionsCreate"))
const SubscriptionsEdit = lazy(() => import("./pages/SubscriptionsEdit"))
const Usage = lazy(() => import("./pages/Usage"))
const UsageCreate = lazy(() => import("./pages/UsageCreate"))
const Meters = lazy(() => import("./pages/Meters"))
const MetersCreate = lazy(() => import("./pages/MetersCreate"))
const MetersEdit = lazy(() => import("./pages/MetersEdit"))
const Rating = lazy(() => import("./pages/Rating"))
const Coupons = lazy(() => import("./pages/Coupons"))
const Ledger = lazy(() => import("./pages/Ledger"))
const LedgerCreate = lazy(() => import("./pages/LedgerCreate"))
const Invoices = lazy(() => import("./pages/Invoices"))
const InvoicesCreate = lazy(() => import("./pages/InvoicesCreate"))
const InvoicesManage = lazy(() => import("./pages/InvoicesManage"))
const Payments = lazy(() => import("./pages/Payments"))
const Taxes = lazy(() => import("./pages/Taxes"))
const TaxesCreate = lazy(() => import("./pages/TaxesCreate"))
const Apps = lazy(() => import("./pages/Apps"))
const TestClock = lazy(() => import("./pages/TestClock"))
const AuditLogs = lazy(() => import("./pages/AuditLogs"))
const Settings = lazy(() => import("./pages/Settings"))
const NotFound = lazy(() => import("./pages/NotFound"))
const Products = lazy(() => import("./pages/Products"))
const ProductsCreate = lazy(() => import("./pages/ProductsCreate"))
const ProductsEdit = lazy(() => import("./pages/ProductsEdit"))
const Features = lazy(() => import("./pages/Features"))
const FeaturesCreate = lazy(() => import("./pages/FeaturesCreate"))
const FeaturesEdit = lazy(() => import("./pages/FeaturesEdit"))
const ApiKeys = lazy(() => import("./pages/ApiKeys"))
const FeatureFlags = lazy(() => import("./pages/FeatureFlags"))
const ProfileSessions = lazy(() => import("./pages/ProfileSessions"))
const OrganizationSessions = lazy(() => import("./pages/OrganizationSessions"))
const AIAssistant = lazy(() => import("./pages/AIAssistant"))
const AIWorkflows = lazy(() => import("./pages/AIWorkflows"))
const AIScheduledJobs = lazy(() => import("./pages/AIScheduledJobs"))

type NavItem = {
  label: string
  path: string
  icon: JSX.Element
  end?: boolean
}

type NavGroup = {
  label: string
  items: NavItem[]
}

// ── Nav Icons ─────────────────────────────────
const icons: Record<string, JSX.Element> = {
  dashboard: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="1" width="6" height="6" rx="1.5"/>
      <rect x="9" y="1" width="6" height="6" rx="1.5"/>
      <rect x="1" y="9" width="6" height="6" rx="1.5"/>
      <rect x="9" y="9" width="6" height="6" rx="1.5"/>
    </svg>
  ),
  organizations: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="5" r="2.5"/>
      <path d="M2 13.5c0-3.314 2.686-5 6-5s6 1.686 6 5" strokeLinecap="round"/>
    </svg>
  ),
  customers: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="6" cy="5" r="2"/>
      <path d="M1 13c0-2.761 2.239-4 5-4s5 1.239 5 4" strokeLinecap="round"/>
      <path d="M11 8c1.5.3 3 1.2 3 3" strokeLinecap="round"/>
      <circle cx="12.5" cy="5" r="1.5"/>
    </svg>
  ),
  plans: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1.5" y="1.5" width="13" height="13" rx="2"/>
      <path d="M5 8h6M5 5h6M5 11h4" strokeLinecap="round"/>
    </svg>
  ),
  subscriptions: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 4h12v9a1 1 0 01-1 1H3a1 1 0 01-1-1V4z"/>
      <path d="M2 4V3a1 1 0 011-1h10a1 1 0 011 1v1" strokeLinecap="round"/>
      <path d="M6 8l1.5 1.5L10 7" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  usage: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 12L5 8l3 2 3-5 3 3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  meters: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="8" r="6"/>
      <path d="M8 8l-3-3" strokeLinecap="round"/>
      <circle cx="8" cy="8" r="1" fill="currentColor" stroke="none"/>
    </svg>
  ),
  rating: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M8 1l1.8 3.6L14 5.3l-3 2.9.7 4.1L8 10.4l-3.7 1.9.7-4.1L2 5.3l4.2-.7z" strokeLinejoin="round"/>
    </svg>
  ),
  coupons: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2.5 6.5l4-4h6.2a.8.8 0 01.8.8v6.2l-4 4a1.2 1.2 0 01-1.7 0L2.5 8.2a1.2 1.2 0 010-1.7z" strokeLinejoin="round"/>
      <circle cx="10.8" cy="5.2" r="1" />
      <path d="M6 10l4-4" strokeLinecap="round"/>
    </svg>
  ),
  ledger: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="1" width="12" height="14" rx="1.5"/>
      <path d="M5 5h6M5 8h6M5 11h3" strokeLinecap="round"/>
    </svg>
  ),
  invoices: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M3 2h10l-1 3H4L3 2z" strokeLinecap="round" strokeLinejoin="round"/>
      <rect x="2" y="5" width="12" height="9" rx="1"/>
      <path d="M5 9h6M5 12h4" strokeLinecap="round"/>
    </svg>
  ),
  payments: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="3" width="14" height="10" rx="1.5"/>
      <path d="M1 7h14" strokeLinecap="round"/>
      <path d="M4 10.5h2" strokeLinecap="round" strokeWidth="2"/>
    </svg>
  ),
  taxes: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M4 12L12 4M5.5 4.5h-2v2M12.5 11.5h-2v2" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  auditlogs: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="8" r="6"/>
      <path d="M8 5v3.5l2 1.5" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  settings: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="8" r="2"/>
      <path d="M8 1v2M8 13v2M1 8h2M13 8h2M2.9 2.9l1.4 1.4M11.7 11.7l1.4 1.4M2.9 13.1l1.4-1.4M11.7 4.3l1.4-1.4" strokeLinecap="round"/>
    </svg>
  ),
  apps: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="2" width="5" height="5" rx="1.2"/>
      <rect x="9" y="2" width="5" height="5" rx="1.2"/>
      <rect x="2" y="9" width="5" height="5" rx="1.2"/>
      <rect x="9" y="9" width="5" height="5" rx="1.2"/>
    </svg>
  ),
  products: (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
      <line x1="8" y1="21" x2="16" y2="21" />
      <line x1="12" y1="17" x2="12" y2="21" />
    </svg>
  ),
  features: (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M12 2L2 7l10 5 10-5-10-5z" />
      <path d="M2 17l10 5 10-5" />
      <path d="M2 12l10 5 10-5" />
    </svg>
  ),
  apikeys: (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
    </svg>
  ),
  feature_flags: (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M4 7h9" strokeLinecap="round" />
      <path d="M4 17h16" strokeLinecap="round" />
      <circle cx="16" cy="7" r="3" />
      <circle cx="9" cy="17" r="3" />
    </svg>
  ),
  sessions: (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="4" y="5" width="16" height="14" rx="2" />
      <path d="M8 9h8M8 13h5" strokeLinecap="round" />
    </svg>
  ),
  test_clock: (
    <svg className="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="12" cy="12" r="8" />
      <path d="M12 8v4l2.5 2.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  ai_assistant: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M8 1.5l1.3 3.2 3.2 1.3-3.2 1.3L8 10.5 6.7 7.3 3.5 6l3.2-1.3L8 1.5z" strokeLinejoin="round"/>
      <path d="M2 11.5l.6 1.4 1.4.6-1.4.6-.6 1.4-.6-1.4-1.4-.6 1.4-.6.6-1.4z" strokeLinejoin="round"/>
      <path d="M12.2 11l.5 1.1 1.1.5-1.1.5-.5 1.1-.5-1.1-1.1-.5 1.1-.5.5-1.1z" strokeLinejoin="round"/>
    </svg>
  ),
  ai_workflows: (
    <svg className="nav-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="2" width="12" height="12" rx="2" />
      <path d="M5 6h6M5 9h6" strokeLinecap="round"/>
      <path d="M6 11.5l1.2-1.2 1.2 1.2" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
}

function usePageTitle() {
  const { t } = useTranslation()
  const location = useLocation()
  const pathname = location.pathname
  const getTitle = (key: string) => ({
    label: t(`routes.${key}.label`),
    desc: t(`routes.${key}.desc`),
  })
  const routeKeyMap: Record<string, string> = {
    "/dashboard": "dashboard",
    "/organizations": "organizations",
    "/organizations/new": "organizations_new",
    "/customers": "customers",
    "/customers/new": "customers_new",
    "/plans": "plans",
    "/plans/new": "plans_new",
    "/subscriptions": "subscriptions",
    "/subscriptions/new": "subscriptions_new",
    "/usage": "usage",
    "/usage/new": "usage_new",
    "/meters": "meters",
    "/meters/new": "meters_new",
    "/rating": "rating",
    "/coupons": "coupons",
    "/ledger": "ledger",
    "/ledger/new": "ledger_new",
    "/invoices": "invoices",
    "/invoices/new": "invoices_new",
    "/payments": "payments",
    "/taxes": "taxes",
    "/taxes/new": "taxes_new",
    "/audit-logs": "audit_logs",
    "/apps": "apps",
    "/test-clock": "test_clock",
    "/settings": "settings",
    "/products": "products",
    "/products/new": "products_new",
    "/features": "features",
    "/features/new": "features_new",
    "/api-keys": "api_keys",
    "/feature-flags": "feature_flags",
    "/profile/sessions": "profile_sessions",
    "/ai-assistant": "ai_assistant",
    "/ai-workflows": "ai_workflows",
    "/ai-scheduled-jobs": "ai_scheduled_jobs",
  }
  if (pathname === "/profile/sessions") {
    return { label: "My Sessions", desc: "Review and revoke your admin sessions" }
  }
  if (pathname === "/organizations" || pathname === "/organizations/new") {
    return getTitle(routeKeyMap[pathname] ?? "organizations")
  }
  if (pathname.match(/^\/organizations\/[^/]+\/edit$/)) {
    return getTitle("organizations_edit")
  }
  const orgMatch = pathname.match(/^\/organizations\/[^/]+(\/.*)?$/)
  const normalizedPath = orgMatch ? orgMatch[1] || "/dashboard" : pathname
  // Handle dynamic routes
  if (normalizedPath.match(/^\/customers\/[^/]+\/edit/)) {
    return getTitle("customers_edit")
  }
  if (normalizedPath.match(/^\/plans\/[^/]+\/edit/)) {
    return getTitle("plans_edit")
  }
  if (normalizedPath.match(/^\/products\/[^/]+\/edit/)) {
    return { label: "Edit Product", desc: "Modify product details" }
  }
  if (normalizedPath.match(/^\/features\/[^/]+\/edit/)) {
    return { label: "Edit Feature", desc: "Modify feature details" }
  }
  if (normalizedPath.match(/^\/meters\/[^/]+\/edit/)) {
    return getTitle("meters_edit")
  }
  if (normalizedPath.match(/^\/invoices\/[^/]+\/manage/)) {
    return getTitle("invoices_manage")
  }
  if (normalizedPath === "/sessions") {
    return { label: "Organization Sessions", desc: "Review and revoke organization admin sessions" }
  }
  if (normalizedPath.match(/^\/subscriptions\/[^/]+\/edit/)) {
    return getTitle("subscriptions_edit")
  }
  const resolved = routeKeyMap[normalizedPath] ?? routeKeyMap[pathname]
  if (!resolved) {
    if (pathname.includes("/products")) return { label: "Products", desc: "Manage catalog products" }
    if (pathname.includes("/features")) return { label: "Features", desc: "Manage feature access" }
    if (pathname.includes("/api-keys")) return { label: "API Keys", desc: "Manage API keys" }
  }
  return resolved ? getTitle(resolved) : getTitle("default")
}

// ── Icons: Topbar ────────────────────────────
function SearchIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="7" cy="7" r="4.5"/>
      <path d="M10.5 10.5L14 14" strokeLinecap="round"/>
    </svg>
  )
}

function LogoutIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M10 3h3a1 1 0 011 1v8a1 1 0 01-1 1h-3" strokeLinecap="round"/>
      <path d="M7 5l-3 3 3 3M4 8h8" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

function ChevronIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M4 6l4 4 4-4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function OrgSwitcher({
  orgs,
  activeOrg,
  loading,
  onSwitch,
  variant,
  avatarText,
  menuPosition = "down",
}: {
  orgs: OrganizationListItem[]
  activeOrg: OrganizationListItem | null
  loading: boolean
  onSwitch: (orgId: string) => void
  variant?: "topbar" | "sidebar"
  avatarText?: string
  menuPosition?: "down" | "up"
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement | null>(null)
  const isSidebar = variant === "sidebar"
  const menuClass = menuPosition === "up" ? " org-switcher--menu-up" : " org-switcher--menu-down"

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      if (!ref.current) return
      if (!ref.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClick)
    return () => document.removeEventListener("mousedown", handleClick)
  }, [])

  return (
    <div className={`org-switcher${isSidebar ? " org-switcher--sidebar" : ""}${isSidebar ? menuClass : ""}`} ref={ref}>
      <button
        type="button"
        className={`org-switcher-trigger${isSidebar ? " org-switcher-trigger--sidebar" : ""}`}
        onClick={() => setOpen((prev) => !prev)}
        disabled={loading}
        data-testid="org-switcher-trigger"
      >
        {isSidebar ? (
          <>
            <div className="org-switcher-current">
              <div className="org-switcher-avatar">{avatarText ?? "ORG"}</div>
              <div className="org-switcher-meta">
                <span className="org-switcher-name">
                  {loading ? t("org_switcher.loading") : activeOrg?.name ?? t("org_switcher.select")}
                </span>
                <span className="org-switcher-role">
                  {loading ? t("org_switcher.fetching_access") : activeOrg?.role ?? t("org_switcher.no_role")}
                </span>
              </div>
            </div>
            <ChevronIcon />
          </>
        ) : (
          <>
            <span className="org-switcher-label">{t("org_switcher.workspace")}</span>
            <span className="org-switcher-name">
              {loading ? t("org_switcher.loading") : activeOrg?.name ?? t("org_switcher.select")}
            </span>
            <ChevronIcon />
          </>
        )}
      </button>
      {open ? (
        <div className="org-switcher-menu" role="menu">
          <div className="org-switcher-header">{t("org_switcher.workspaces")}</div>
          {orgs.length === 0 ? (
            <div className="org-switcher-empty">{t("org_switcher.empty")}</div>
          ) : (
            orgs.map((org) => (
              <button
                key={org.id}
                type="button"
                className={`org-switcher-item${activeOrg?.id === org.id ? " is-active" : ""}`}
                onClick={() => {
                  setOpen(false)
                  onSwitch(org.id)
                }}
              >
                <span className="org-switcher-item-name">{org.name}</span>
                <span className="org-switcher-item-role">{org.role}</span>
              </button>
            ))
          )}
          <div className="org-switcher-divider" />
          <button
            type="button"
            className="org-switcher-item is-action"
            onClick={() => {
              setOpen(false)
              navigate("/organizations/new")
            }}
          >
            {t("org_switcher.new_org")}
          </button>
        </div>
      ) : null}
    </div>
  )
}

function Topbar({ authRequired }: { authRequired: boolean }) {
  const { t } = useTranslation()
  const { label, desc } = usePageTitle()
  const navigate = useNavigate()

  return (
    <header className="topbar">
      <div className="topbar-left">
        <div className="topbar-eyebrow">{t("topbar.eyebrow")}</div>
        <div className="topbar-title-row">
          <h1 className="topbar-title">{label}</h1>
        </div>
      </div>
      <div className="topbar-actions">
        <div className="topbar-search">
          <SearchIcon />
          <Input placeholder={t("topbar.search_placeholder", { label })} style={{ border: "none", boxShadow: "none", padding: 0, height: "auto", background: "transparent" }} />
        </div>
        {authRequired ? (
          <button
            className="btn btn-secondary btn-sm"
            style={{ display: "flex", alignItems: "center", gap: 6 }}
            onClick={() => navigate("/profile/sessions")}
          >
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
              <rect x="2" y="3" width="12" height="8" rx="1.5" />
              <path d="M6 13h4" strokeLinecap="round" />
            </svg>
            My Sessions
          </button>
        ) : null}
        {authRequired ? (
          <button
            className="btn btn-secondary btn-sm"
            style={{ display: "flex", alignItems: "center", gap: 6 }}
            title={desc}
            data-testid="topbar-logout"
            onClick={() => {
              api.auth
                .logout()
                .catch(() => undefined)
                .finally(() => {
                  clearOrgId()
                  window.location.reload()
                })
            }}
          >
            <LogoutIcon />
            {t("topbar.logout")}
          </button>
        ) : null}
        <div className="topbar-avatar" title={t("topbar.admin_user")}>A</div>
      </div>
    </header>
  )
}

function AppShell() {
  return (
    <BrowserRouter>
      <AppLayout />
    </BrowserRouter>
  )
}

function AppLayout() {
  const { t } = useTranslation()
  const authRequired = isAuthRequired()
  const location = useLocation()
  const navigate = useNavigate()
  const [orgs, setOrgs] = useState<OrganizationListItem[]>([])
  const [orgLoading, setOrgLoading] = useState(false)
  const [activeOrgId, setActiveOrgId] = useState(() => getOrgId() ?? "")
  const switchingRef = useRef(false)
  const isOrgCreate = location.pathname === "/organizations/new"

  const orgIdFromPath = useMemo(() => {
    const match = location.pathname.match(/^\/organizations\/([^/]+)(?:\/|$)/)
    const raw = match ? match[1] : ""
    if (!raw || raw === "new") return ""
    return raw
  }, [location.pathname])

  useLayoutEffect(() => {
    if (!orgIdFromPath) return
    const stored = getOrgId()
    if (stored !== orgIdFromPath) {
      setOrgId(orgIdFromPath)
    }
  }, [orgIdFromPath])

  const activeOrg = useMemo(
    () => orgs.find((org) => org.id === activeOrgId) ?? null,
    [orgs, activeOrgId]
  )

  useEffect(() => {
    let mounted = true
    setOrgLoading(true)
    api.organizations
      .list()
      .then((resp) => {
        if (!mounted) return
        setOrgs(resp)
        if (resp.length === 0) {
          return
        }
        const stored = orgIdFromPath || getOrgId() || ""
        if (!stored || !resp.some((org) => org.id === stored)) {
          setActiveOrgId(resp[0].id)
          setOrgId(resp[0].id)
        } else {
          setActiveOrgId(stored)
          setOrgId(stored)
        }
      })
      .catch(() => undefined)
      .finally(() => {
        if (mounted) setOrgLoading(false)
      })
    return () => {
      mounted = false
    }
  }, [orgIdFromPath])

  const syncSessionOrg = async (targetId: string, showToast = false) => {
    if (!targetId) return false
    if (!authRequired) {
      setActiveOrgId(targetId)
      setOrgId(targetId)
      return true
    }
    if (switchingRef.current) return false
    switchingRef.current = true
    try {
      await api.auth.switchOrg(targetId)
      setActiveOrgId(targetId)
      setOrgId(targetId)
      return true
    } catch (err) {
      if (showToast) {
        toast.error(t("toast.org_switch_failed"), err instanceof Error ? err.message : undefined)
      }
      return false
    } finally {
      switchingRef.current = false
    }
  }

  useEffect(() => {
    let cancelled = false
    if (!orgIdFromPath || orgIdFromPath === activeOrgId) return
    void (async () => {
      const ok = await syncSessionOrg(orgIdFromPath)
      if (ok || cancelled) return
      try {
        const me = await api.auth.me()
        if (cancelled) return
        if (me.orgId) {
          setActiveOrgId(me.orgId)
          setOrgId(me.orgId)
          navigate(`/organizations/${me.orgId}/dashboard`, { replace: true })
        }
      } catch {
        // ignore
      }
    })()
    return () => {
      cancelled = true
    }
  }, [activeOrgId, navigate, orgIdFromPath])

  useEffect(() => {
    if (!orgIdFromPath || orgs.length === 0) return
    if (orgs.some((org) => org.id === orgIdFromPath)) return
    navigate("/organizations", { replace: true })
  }, [navigate, orgIdFromPath, orgs])

  const handleOrgSwitch = async (nextId: string) => {
    if (!nextId || nextId === activeOrgId) return
    const ok = await syncSessionOrg(nextId, true)
    if (!ok) return
    const match = location.pathname.match(/^\/organizations\/[^/]+(\/.*)?$/)
    const suffix = match ? match[1] || "/dashboard" : "/dashboard"
    navigate(`/organizations/${nextId}${suffix}`, { replace: true })
  }

  const orgInitials = useMemo(() => {
    if (!activeOrg?.name) return "ORG"
    const parts = activeOrg.name.trim().split(/\s+/)
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
    return `${parts[0][0]}${parts[1][0]}`.toUpperCase()
  }, [activeOrg])

  const orgBase = activeOrgId ? `/organizations/${activeOrgId}` : ""
  const isAssistantImmersive = location.pathname.includes("/ai-assistant")
  const navigation = useMemo(() => {
    const withOrg = (path: string) => (orgBase ? `${orgBase}${path}` : "/organizations")
    const canManageOrgSessions = activeOrg?.role === "OWNER" || activeOrg?.role === "ADMIN"
    const primary: NavGroup[] = [
      {
        label: t("nav.groups.core"),
        items: [
          { label: t("nav.items.dashboard"), path: withOrg("/dashboard"), icon: icons.dashboard, end: true },
          { label: t("nav.items.customers"), path: withOrg("/customers"), icon: icons.customers },
        ],
      },
      {
        label: t("nav.groups.catalog"),
        items: [
          { label: t("nav.items.products"), path: withOrg("/products"), icon: icons.products },
          { label: t("nav.items.features"), path: withOrg("/features"), icon: icons.features },
          { label: t("nav.items.plans"), path: withOrg("/plans"), icon: icons.plans },
          { label: t("nav.items.coupons"), path: withOrg("/coupons"), icon: icons.coupons },
        ],
      },
      {
        label: t("nav.groups.billing"),
        items: [
          { label: t("nav.items.subscriptions"), path: withOrg("/subscriptions"), icon: icons.subscriptions },
          { label: t("nav.items.usage"), path: withOrg("/usage"), icon: icons.usage },
          { label: t("nav.items.meters"), path: withOrg("/meters"), icon: icons.meters },
          { label: t("nav.items.rating"), path: withOrg("/rating"), icon: icons.rating },
        ],
      },
      {
        label: t("nav.groups.finance"),
        items: [
          { label: t("nav.items.ledger"), path: withOrg("/ledger"), icon: icons.ledger },
          { label: t("nav.items.invoices"), path: withOrg("/invoices"), icon: icons.invoices },
          { label: t("nav.items.payments"), path: withOrg("/payments"), icon: icons.payments },
          { label: t("nav.items.taxes"), path: withOrg("/taxes"), icon: icons.taxes },
        ],
      },
      {
        label: t("nav.groups.system"),
        items: [
          { label: t("nav.items.audit_logs"), path: withOrg("/audit-logs"), icon: icons.auditlogs },
          { label: t("nav.items.apps"), path: withOrg("/apps"), icon: icons.apps },
          { label: t("nav.items.test_clock"), path: withOrg("/test-clock"), icon: icons.test_clock },
        ],
      },
      {
        label: t("nav.groups.ai"),
        items: [
          { label: t("nav.items.ai_assistant"), path: withOrg("/ai-assistant"), icon: icons.ai_assistant },
          { label: t("nav.items.ai_workflows"), path: withOrg("/ai-workflows"), icon: icons.ai_workflows },
          { label: t("nav.items.ai_scheduled_jobs"), path: withOrg("/ai-scheduled-jobs"), icon: icons.ai_workflows },
        ],
      },
    ]
    const utilities: NavGroup[] = [
      {
        label: t("nav.groups.developer"),
        items: [
          { label: t("nav.items.api_keys"), path: withOrg("/api-keys"), icon: icons.apikeys },
          { label: t("nav.items.feature_flags"), path: withOrg("/feature-flags"), icon: icons.feature_flags },
          ...(canManageOrgSessions ? [{ label: t("nav.items.sessions"), path: withOrg("/sessions"), icon: icons.sessions }] : []),
          { label: t("nav.items.settings"), path: withOrg("/settings"), icon: icons.settings },
        ],
      },
    ]
    return { primary, utilities }
  }, [activeOrg?.role, orgBase, t])

  const RootRedirect = () => {
    if (activeOrgId) {
      return <Navigate to={`/organizations/${activeOrgId}/dashboard`} replace />
    }
    return <Navigate to="/organizations" replace />
  }

  const OrgRoutes = () => (
    <Routes>
      <Route index element={<Navigate to="dashboard" replace />} />
      <Route path="dashboard" element={<Dashboard />} />
      <Route path="customers" element={<Customers />} />
      <Route path="customers/new" element={<CustomersCreate />} />
      <Route path="customers/:id/edit" element={<CustomersEdit />} />
      <Route path="plans" element={<Plans />} />
      <Route path="plans/new" element={<PlansCreate />} />
      <Route path="plans/:id/edit" element={<PlansEdit />} />
      <Route path="subscriptions" element={<Subscriptions />} />
      <Route path="subscriptions/new" element={<SubscriptionsCreate />} />
      <Route path="subscriptions/:id/edit" element={<SubscriptionsEdit />} />
      <Route path="usage" element={<Usage />} />
      <Route path="usage/new" element={<UsageCreate />} />
      <Route path="meters" element={<Meters />} />
      <Route path="meters/new" element={<MetersCreate />} />
      <Route path="meters/:id/edit" element={<MetersEdit />} />
      <Route path="rating" element={<Rating />} />
      <Route path="coupons" element={<Coupons />} />
      <Route path="ledger" element={<Ledger />} />
      <Route path="ledger/new" element={<LedgerCreate />} />
      <Route path="invoices" element={<Invoices />} />
      <Route path="invoices/new" element={<InvoicesCreate />} />
      <Route path="invoices/:id/manage" element={<InvoicesManage />} />
      <Route path="payments" element={<Payments />} />
      <Route path="taxes" element={<Taxes />} />
      <Route path="taxes/new" element={<TaxesCreate />} />
      <Route path="apps" element={<Apps />} />
      <Route path="audit-logs" element={<AuditLogs />} />
      <Route path="test-clock" element={<TestClock />} />
      <Route path="settings" element={<Settings />} />
      <Route path="sessions" element={<OrganizationSessions />} />
      <Route path="products" element={<Products />} />
      <Route path="products/new" element={<ProductsCreate />} />
      <Route path="products/:id/edit" element={<ProductsEdit />} />
      <Route path="features" element={<Features />} />
      <Route path="features/new" element={<FeaturesCreate />} />
      <Route path="features/:id/edit" element={<FeaturesEdit />} />
      <Route path="api-keys" element={<ApiKeys />} />
      <Route path="feature-flags" element={<FeatureFlags />} />
      <Route path="ai-assistant" element={<AIAssistant />} />
      <Route path="ai-workflows" element={<AIWorkflows />} />
      <Route path="ai-scheduled-jobs" element={<AIScheduledJobs />} />
      <Route path="*" element={<NotFound />} />
    </Routes>
  )

  const content = (
    <Suspense
      fallback={
        <div className="route-loading">
          <div className="loader" />
          {t("app.loading_module")}
        </div>
      }
    >
      <Routes>
        <Route path="/" element={<RootRedirect />} />
        <Route path="/organizations" element={<RootRedirect />} />
        <Route path="/organizations/new" element={<OrganizationsCreate />} />
        <Route path="/organizations/:id/edit" element={<OrganizationsEdit />} />
        <Route path="/profile/sessions" element={<ProfileSessions />} />
        <Route path="/organizations/:orgId/*" element={<OrgRoutes />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </Suspense>
  )

  if (isOrgCreate) {
    return (
      <div className="org-create-layout">
        {content}
        <ToastContainer />
      </div>
    )
  }

  const showAssistantLauncher = Boolean(activeOrgId) && !location.pathname.includes("/ai-assistant")

  if (isAssistantImmersive) {
    return (
      <div className="min-h-screen bg-[hsl(var(--bg-surface))]">
        {content}
        <ToastContainer />
      </div>
    )
  }

  return (
    <div className="app-shell">
      {/* ── Sidebar ── */}
      <aside className="sidebar">
        <div className="sidebar-org">
          <OrgSwitcher
            orgs={orgs}
            activeOrg={activeOrg}
            loading={orgLoading}
            onSwitch={handleOrgSwitch}
            variant="sidebar"
            avatarText={orgInitials}
            menuPosition="down"
          />
        </div>

        <nav className="nav-groups">
          <div className="nav-section">
            {navigation.primary.map((group) => (
              <div key={group.label} className="nav-group">
                <div className="nav-group-label">{group.label}</div>
                {group.items.map((item) => (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    end={item.end}
                    className={({ isActive }) => (isActive ? "nav-link active" : "nav-link")}
                  >
                    {item.icon}
                    {item.label}
                  </NavLink>
                ))}
              </div>
            ))}
          </div>

          <div className="nav-section nav-section-secondary">
            {navigation.utilities.map((group) => (
              <div key={group.label} className="nav-group nav-group-secondary">
                <div className="nav-group-label">{group.label}</div>
                {group.items.map((item) => (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    end={item.end}
                    className={({ isActive }) => (isActive ? "nav-link active" : "nav-link")}
                  >
                    {item.icon}
                    {item.label}
                  </NavLink>
                ))}
              </div>
            ))}
          </div>
        </nav>

      </aside>

      {/* ── Main ── */}
      <main className="main">
        <ConfigWarningsBanner />
        <Topbar authRequired={authRequired} />
        {content}
      </main>
      {showAssistantLauncher ? (
        <button
          type="button"
          onClick={() => navigate(`${orgBase}/ai-assistant`)}
          className="fixed bottom-6 right-6 z-40 inline-flex h-12 w-12 items-center justify-center rounded-full border border-slate-200 bg-slate-950 text-white shadow-[0_18px_40px_rgba(15,23,42,0.28)] transition hover:-translate-y-0.5 hover:bg-slate-900"
          aria-label="Open AI Assistant"
        >
          {icons.ai_assistant}
        </button>
      ) : null}
      <ToastContainer />
    </div>
  )
}

export default function App() {
  return (
    <AuthGate>
      <AppShell />
    </AuthGate>
  )
}
