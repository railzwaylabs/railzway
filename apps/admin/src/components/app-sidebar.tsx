import * as React from "react"
import { Link, NavLink, useLocation, useParams } from "react-router-dom"

import { NavMain } from "@/components/nav-main"
import { NavSecondary } from "@/components/nav-secondary"
import { canManageBilling } from "@/lib/roles"
import { useOrgStore } from "@/stores/orgStore"
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { BarChart3, Copy, Gauge, History, Home, Key, LayoutGrid, Package, Receipt, RefreshCcw, Settings, Tag, Users, Wallet, Zap } from "lucide-react"
import { cn } from "@/lib/utils"

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { orgId } = useParams()
  const { pathname } = useLocation()
  const role = useOrgStore((state) => state.currentOrg?.role)
  const canAccessAdmin = canManageBilling(role)
  const orgBasePath = orgId ? `/orgs/${orgId}` : "/orgs"
  const matchPrefix = (base: string) => (pathname: string) =>
    pathname === base || pathname.startsWith(`${base}/`)
  const normalizePath = (value: string) => {
    const trimmed = value.replace(/\/+$/, "")
    return trimmed.length ? trimmed : "/"
  }
  const currentPath = normalizePath(pathname)
  const isPathActive = (target: string) => {
    const normalized = normalizePath(target)
    return currentPath === normalized || currentPath.startsWith(`${normalized}/`)
  }

  const navMain = [
    {
      title: "Home",
      url: `${orgBasePath}/home`,
      icon: Home,
    },
    // Pricing stays within each product so navigation reflects user intent, not backend tables.
    {
      title: "Products",
      url: `${orgBasePath}/products`,
      icon: Package,
    },
    {
      title: "Pricing",
      url: `${orgBasePath}/prices`,
      icon: Tag,
      isActive: (pathname: string) =>
        pathname === `${orgBasePath}/prices` ||
        pathname.startsWith(`${orgBasePath}/prices/`) ||
        pathname.startsWith(`${orgBasePath}/pricings`) ||
        pathname.startsWith(`${orgBasePath}/price-amounts`) ||
        pathname.startsWith(`${orgBasePath}/price-tiers`),
    },
    {
      title: "Meters",
      url: `${orgBasePath}/meter`,
      icon: Gauge,
    },
    {
      title: "Marketplace",
      url: `${orgBasePath}/integrations`,
      icon: LayoutGrid,
      isActive: (pathname: string) =>
        matchPrefix(`${orgBasePath}/integrations`)(pathname) &&
        !matchPrefix(`${orgBasePath}/integrations/connections`)(pathname),
    },
  ].filter(() => canAccessAdmin)

  const billingNav = [
    {
      title: "Overview",
      url: `${orgBasePath}/billing/overview`,
      icon: BarChart3,
    },
    {
      title: "Operations",
      url: `${orgBasePath}/billing/operations`,
      icon: Zap,
    },
    {
      title: "Invoices",
      url: `${orgBasePath}/invoices`,
      icon: Receipt,
    },
    {
      title: "Customers",
      url: `${orgBasePath}/customers`,
      icon: Users,
    },
    {
      title: "Subscriptions",
      url: `${orgBasePath}/subscriptions`,
      icon: RefreshCcw,
      isActive: (path: string) =>
        path === normalizePath(`${orgBasePath}/subscriptions`) ||
        path.startsWith(`${normalizePath(`${orgBasePath}/subscriptions`)}/`) ||
        path === normalizePath(`${orgBasePath}/test-clocks`) ||
        path.startsWith(`${normalizePath(`${orgBasePath}/test-clocks`)}/`),
    },
    {
      title: "Invoice Templates",
      url: `${orgBasePath}/invoice-templates`,
      icon: Copy,
    },
  ].filter(() => canAccessAdmin)

  const navSecondary = [
    {
      title: "API Keys",
      url: `${orgBasePath}/api-keys`,
      icon: Key,
    },
    {
      title: "Integrations",
      url: `${orgBasePath}/integrations/connections`,
      icon: RefreshCcw,
    },
    {
      title: "Checkout Options",
      url: `${orgBasePath}/checkout-options`,
      icon: Wallet,
    },
    {
      title: "Audit Logs",
      url: `${orgBasePath}/audit-logs`,
      icon: History,
    },
    {
      title: "Settings",
      url: `${orgBasePath}/settings`,
      icon: Settings,
    },
  ].filter(() => canAccessAdmin)


  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:!p-1.5 transition-all duration-300 hover:bg-transparent"
            >
              <NavLink to={`${orgBasePath}/home`}>
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 text-white text-sm font-black shadow-[0_0_15px_rgba(79,70,229,0.3)]">
                  R
                </span>
                <span className="text-base font-bold tracking-tight text-text-primary">Railzway</span>
              </NavLink>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={navMain} />
        {billingNav.length > 0 && (
          <SidebarGroup>
            <SidebarGroupLabel className="px-3">Billing</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {billingNav.map((item) => {
                  const isActive = item.isActive
                    ? item.isActive(currentPath)
                    : isPathActive(item.url)
                  return (
                    <SidebarMenuItem key={item.title}>
                      <SidebarMenuButton asChild tooltip={item.title} isActive={isActive}>
                        <Link to={item.url} aria-current={isActive ? "page" : undefined}>
                          <item.icon className={cn("transition-colors", isActive ? "text-accent-primary" : "text-text-muted")} />
                          <span className={cn("transition-colors", isActive ? "text-accent-primary font-bold" : "text-text-primary")}>
                            {item.title}
                          </span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
        <NavSecondary items={navSecondary} className="mt-auto" />
      </SidebarContent>
    </Sidebar>
  )
}
