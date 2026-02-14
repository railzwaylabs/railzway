import { useState } from "react"
import { Link, useParams } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Circle,
  ArrowRight,
  Sparkles,
  ShieldCheck,
  Users,
  Webhook,
  Key
} from "lucide-react"

import { cn } from "@/lib/utils"
import { getSystemReadiness } from "../api/dashboard"

type ReadinessStatus = "blocked" | "completed" | "optional" | "ready"

interface ReadinessItemConfig {
  title: string
  description: string
  href: string
  actionLabel: string
  icon?: any
}

const READINESS_CONFIG: Record<string, ReadinessItemConfig> = {
  // --- REQUIRED ---
  product_exists: {
    title: "Products",
    description: "At least one active product is required.",
    href: "/products",
    actionLabel: "Create Product",
  },
  price_exists_for_product: {
    title: "Pricing Rules",
    description: "Billing cannot start without active pricing rules.",
    href: "/prices",
    actionLabel: "Define Prices",
  },
  meter_exists_if_usage_price: {
    title: "Usage Meters",
    description: "Usage events are ignored until a meter is defined.",
    href: "/meter",
    actionLabel: "View Meters",
  },
  payment_provider_connected: {
    title: "Payment Provider",
    description: "Connect a payment provider to process payments.",
    href: "/payment-providers",
    actionLabel: "Connect Provider",
  },
  payment_configuration_complete: {
    title: "Checkout Options",
    description: "Enable payment methods for customer checkout.",
    href: "/checkout-options",
    actionLabel: "Manage Options",
  },

  // --- RECOMMENDED ---
  invoice_template_customized: {
    title: "Invoice Template",
    description: "Customize your invoice branding and footer.",
    href: "/invoice-templates",
    actionLabel: "Customize",
    icon: Sparkles,
  },
  tax_configuration_explicit: {
    title: "Tax Configuration",
    description: "Ensure tax rates are defined or providers connected.",
    href: "/products/tax-definitions",
    actionLabel: "Configure Tax",
    icon: ShieldCheck,
  },
  secondary_admin_present: {
    title: "Team Access",
    description: "Invite a secondary admin for account recovery.",
    href: "/settings",
    actionLabel: "Invite Team",
    icon: Users,
  },
  api_key_created: {
    title: "API Keys",
    description: "Create an API key to integrate your application.",
    href: "/api-keys",
    actionLabel: "Create Key",
    icon: Key,
  },
  webhooks_configured: {
    title: "Webhooks",
    description: "Listen for billing events in your application.",
    href: "/payment-providers",
    actionLabel: "Setup Webhooks",
    icon: Webhook,
  },
}

export function SystemReadiness() {
  const { orgId } = useParams()
  const [isOpen, setIsOpen] = useState(true)

  const { data, isLoading } = useQuery({
    queryKey: ["readiness", orgId],
    queryFn: () => getSystemReadiness(orgId!),
    enabled: !!orgId,
  })

  // Map backend issues to frontend config
  const issues = data?.issues || []
  const mappedItems = issues.map((issue) => {
    const config = READINESS_CONFIG[issue.id]
    if (!config) return null

    let status: ReadinessStatus = "blocked"
    if (issue.status === "ready") status = "completed"
    if (issue.status === "optional") status = "optional"

    return {
      ...config,
      id: issue.id,
      status: status,
      href: config.href.startsWith("/") ? `/orgs/${orgId}${config.href}` : config.href
    }
  }).filter(Boolean) as (ReadinessItemConfig & { id: string, status: ReadinessStatus })[]

  const blockedItems = mappedItems.filter((i) => i.status === "blocked")
  const recommendedItems = mappedItems.filter((i) => i.status === "optional")

  const isSystemReady = data?.system_state === "ready"
  const blockedCount = blockedItems.length

  if (isLoading) {
    return (
      <div className="rounded-xl border border-border-subtle bg-bg-surface overflow-hidden shadow-sm p-6">
        <div className="flex items-center gap-3 animate-pulse">
          <div className="w-8 h-8 rounded-full bg-slate-200" />
          <div className="space-y-2">
            <div className="h-4 w-32 bg-slate-200 rounded" />
            <div className="h-3 w-48 bg-slate-100 rounded" />
          </div>
        </div>
      </div>
    )
  }

  const hasRecommendations = recommendedItems.length > 0

  return (
    <div className="rounded-xl border border-border-subtle bg-bg-surface overflow-hidden shadow-sm">
      {/* Header Section */}
      <div
        className="flex items-center justify-between px-6 py-4 bg-white border-b border-border-subtle cursor-pointer transition-colors hover:bg-slate-50"
        onClick={() => setIsOpen(!isOpen)}
      >
        <div className="flex items-center gap-3">
          <div className={cn(
            "flex items-center justify-center w-8 h-8 rounded-full transition-colors",
            isSystemReady ? "bg-emerald-100 text-emerald-600" : "bg-amber-100 text-amber-600"
          )}>
            {isSystemReady ? <CheckCircle2 className="w-5 h-5" /> : <AlertTriangle className="w-5 h-5" />}
          </div>
          <div>
            <h3 className="text-base font-semibold text-text-primary">
              {isSystemReady ? "System Ready" : "Configuration Status"}
            </h3>
            <p className="text-sm text-text-secondary">
              {isSystemReady
                ? (hasRecommendations ? "System is active. Review recommendations below." : "All systems operational.")
                : `${blockedCount} required configuration items remaining.`}
            </p>
          </div>
        </div>
        <button className="text-text-muted hover:text-text-primary transition-colors">
          {isOpen ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
        </button>
      </div>

      <div
        className={cn(
          "transition-all duration-300 ease-in-out overflow-hidden grid",
          isOpen ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"
        )}
      >
        <div className="min-h-0">
          {/* Required Section (Only if Blocked) */}
          {!isSystemReady && (
            <div className="px-6 py-4">
              <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wider mb-4 pl-2">Required Configuration</h4>
              <div className="grid gap-3">
                {blockedItems.map((item) => (
                  <ReadinessRow key={item.id} item={item} />
                ))}
              </div>
            </div>
          )}

          {/* Recommended Section (Only if Ready) */}
          {isSystemReady && hasRecommendations && (
            <div className="px-6 py-4 bg-slate-50/50">
              <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wider mb-4 pl-2">Recommended Actions</h4>
              <div className="grid gap-3">
                {recommendedItems.map((item) => (
                  <ReadinessRow key={item.id} item={item} />
                ))}
              </div>
            </div>
          )}
          {isSystemReady && !hasRecommendations && (
            <div className="px-6 py-8 text-center text-text-muted text-sm">
              <ShieldCheck className="w-8 h-8 mx-auto mb-2 opacity-20" />
              <p>Your billing engine is fully configured.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function ReadinessRow({ item }: { item: ReadinessItemConfig & { status: ReadinessStatus } }) {
  const isCompleted = item.status === "completed"
  const isBlocked = item.status === "blocked"
  const isOptional = item.status === "optional"

  const Icon = item.icon || Circle

  return (
    <Link
      to={item.href}
      className={cn(
        "group flex items-center justify-between p-3 rounded-lg border transition-all duration-200",
        isCompleted
          ? "bg-white border-transparent opacity-60 hover:opacity-100 hover:border-border-subtle"
          : "bg-white border-border-subtle hover:border-accent-primary/30 hover:shadow-sm"
      )}
    >
      <div className="flex items-center gap-4">
        <div className={cn(
          "mt-0.5",
          isCompleted ? "text-emerald-500" :
            isBlocked ? "text-amber-500" :
              isOptional ? "text-blue-500" : "text-slate-300"
        )}>
          {isCompleted ? (
            <CheckCircle2 className="w-5 h-5" />
          ) : isBlocked ? (
            <Circle className="w-5 h-5 fill-amber-500/10 stroke-[2.5]" />
          ) : (
            <Icon className="w-5 h-5" />
          )}
        </div>
        <div>
          <h5 className={cn(
            "text-sm font-medium",
            isCompleted ? "text-text-secondary line-through decoration-slate-300" : "text-text-primary"
          )}>
            {item.title}
          </h5>
          <p className="text-sm text-text-muted">
            {item.description}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2 pr-2 opacity-0 group-hover:opacity-100 transition-opacity -translate-x-2 group-hover:translate-x-0 duration-200">
        <span className="text-xs font-medium text-accent-primary">
          {isCompleted ? "Edit" : item.actionLabel}
        </span>
        <ArrowRight className="w-3.5 h-3.5 text-accent-primary" />
      </div>
    </Link>
  )
}
