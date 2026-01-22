import { NavLink } from "react-router-dom"

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

  return (
    <div className="flex flex-wrap items-center gap-2">
      {navItems.map((item) => {
        const to = basePath
          ? item.path === "products"
            ? basePath
            : `${basePath}/${item.path}`
          : ""
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
          <NavLink
            key={item.label}
            to={to}
            end={item.path === "products"}
            className={({ isActive }) =>
              cn(
                "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
                isActive
                  ? "border-accent-primary text-accent-primary"
                  : "border-border-subtle text-text-muted hover:text-text-primary"
              )
            }
          >
            {item.label}
          </NavLink>
        )
      })}
    </div>
  )
}
