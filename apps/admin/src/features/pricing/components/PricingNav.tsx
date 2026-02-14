import { Link, useLocation, useParams } from "react-router-dom"

import { cn } from "@/lib/utils"

const navItems = [
  { label: "Prices", path: "prices" },
  { label: "Pricing models", path: "pricings" },
  { label: "Price amounts", path: "price-amounts" },
  { label: "Price tiers", path: "price-tiers" },
]

export function PricingNav() {
  const { orgId } = useParams()
  const base = orgId ? `/orgs/${orgId}` : "/orgs"
  const { pathname } = useLocation()

  const normalizePath = (value: string) => {
    const trimmed = value.replace(/\/+$/, "")
    return trimmed.length ? trimmed : "/"
  }
  const currentPath = normalizePath(pathname)

  return (
    <div className="flex flex-wrap items-center gap-2">
      {navItems.map((item) => {
        const to = `${base}/${item.path}`
        const target = normalizePath(to)
        const isActive =
          currentPath === target || currentPath.startsWith(`${target}/`)
        return (
          <Link
            key={item.path}
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
