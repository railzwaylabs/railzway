import { Link, useLocation } from "react-router-dom"

import { cn } from "@/lib/utils"
import { useOrgStore } from "@/stores/orgStore"

const navItems = [
  { label: "All products", path: "products" },
  { label: "Features", path: "features" },
  { label: "Tax definitions", path: "tax-definitions" },
]

export function AdminCatalogTabs() {
  const orgId = useOrgStore((state) => state.currentOrg?.id)
  const basePath = orgId ? `/orgs/${orgId}/products` : ""
  const { pathname } = useLocation()

  const normalizePath = (value: string) => {
    const trimmed = value.replace(/\/+$/, "")
    return trimmed.length ? trimmed : "/"
  }
  const currentPath = normalizePath(pathname)
  const baseNormalized = basePath ? normalizePath(basePath) : ""

  const isProductsActive = () => {
    if (!baseNormalized) return false
    if (currentPath === baseNormalized) return true
    if (!currentPath.startsWith(`${baseNormalized}/`)) return false
    if (currentPath.startsWith(`${baseNormalized}/features`)) return false
    if (currentPath.startsWith(`${baseNormalized}/tax-definitions`)) return false
    return true
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      {navItems.map((item) => {
        const to = basePath
          ? item.path === "products"
            ? basePath
            : `${basePath}/${item.path}`
          : ""
        const target = normalizePath(to)
        const isActive =
          item.path === "products"
            ? isProductsActive()
            : currentPath === target || currentPath.startsWith(`${target}/`)
        if (!to) {
          return (
            <span
              key={item.label}
              className="rounded-full border border-border-subtle px-3 py-1 text-xs font-medium text-text-muted"
            >
              {item.label}
            </span>
          )
        }
        return (
          <Link
            key={item.label}
            to={to}
            aria-current={isActive ? "page" : undefined}
            className={cn(
              "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
              isActive
                ? "border-accent-primary text-accent-primary"
                : "border-border-subtle text-text-muted hover:text-text-primary"
            )}
          >
            {item.label}
          </Link>
        )
      })}
    </div>
  )
}
