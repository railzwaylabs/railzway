import { useCallback, useEffect, useMemo, useState } from "react"
import { Link, useParams } from "react-router-dom"
import {
  Activity,
  Calendar,
  CheckCircle,
  Clock,
  CreditCard,
  Pause,
  Play,
  User,
  XCircle,
} from "lucide-react"

import { admin } from "@/api/client"
import { ForbiddenState } from "@/components/forbidden-state"
import { Badge } from "@/components/ui/badge"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { useCursorPagination } from "@/hooks/useCursorPagination"
import { getErrorMessage, isForbiddenError } from "@/lib/api-errors"
import { cn } from "@/lib/utils"

type Subscription = {
  id?: string | number
  ID?: string | number
  customer_id?: string | number
  CustomerID?: string | number
  status?: string
  Status?: string
  collection_mode?: string
  CollectionMode?: string
  billing_cycle_type?: string
  BillingCycleType?: string
  start_at?: string
  StartAt?: string
  created_at?: string
  CreatedAt?: string
  updated_at?: string
  UpdatedAt?: string
  activated_at?: string | null
  ActivatedAt?: string | null
  paused_at?: string | null
  PausedAt?: string | null
  resumed_at?: string | null
  ResumedAt?: string | null
  canceled_at?: string | null
  CanceledAt?: string | null
  ended_at?: string | null
  EndedAt?: string | null
}

type Customer = {
  id?: string | number
  ID?: string | number
  name?: string
  Name?: string
}

type Entitlement = {
  id?: string | number
  ID?: string | number
  subscription_id?: string | number
  SubscriptionID?: string | number
  product_id?: string | number
  ProductID?: string | number
  feature_code?: string
  FeatureCode?: string
  feature_name?: string
  FeatureName?: string
  feature_type?: string
  FeatureType?: string
  meter_id?: string | number | null
  MeterID?: string | number | null
  effective_from?: string
  EffectiveFrom?: string
  effective_to?: string | null
  EffectiveTo?: string | null
  created_at?: string
  CreatedAt?: string
}

type Meter = {
  id?: string | number
  ID?: string | number
  code?: string
  Code?: string
  name?: string
  Name?: string
  active?: boolean
  Active?: boolean
}

type UsageEvent = {
  id?: string | number
  ID?: string | number
  customer_id?: string | number
  CustomerID?: string | number
  subscription_id?: string | number
  SubscriptionID?: string | number
  subscription_item_id?: string | number
  SubscriptionItemID?: string | number
  meter_id?: string | number
  MeterID?: string | number
  meter_code?: string
  MeterCode?: string
  value?: number | string
  Value?: number | string
  recorded_at?: string
  RecordedAt?: string
  status?: string
  Status?: string
  error?: string | null
  Error?: string | null
  idempotency_key?: string
  IdempotencyKey?: string
  created_at?: string
  CreatedAt?: string
}

type ActionType = "activate" | "pause" | "resume" | "cancel"

const statusOrder = ["DRAFT", "ACTIVE", "PAUSED", "CANCELED", "ENDED"]
const ENTITLEMENTS_PAGE_SIZE = 25
const USAGE_PAGE_SIZE = 25

const readField = <T,>(
  item: T | null | undefined,
  keys: (keyof T)[],
  fallback = "-"
) => {
  if (!item) return fallback
  for (const key of keys) {
    const value = item[key]
    if (value === undefined || value === null) continue
    if (typeof value === "string") {
      const trimmed = value.trim()
      if (trimmed) return trimmed
      continue
    }
    return String(value)
  }
  return fallback
}

const formatDateTime = (value?: string | null) => {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "2-digit",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date)
}

const formatStatus = (value?: string) => {
  if (!value) return "-"
  switch (value.toUpperCase()) {
    case "ACTIVE":
      return "Active"
    case "PAUSED":
      return "Paused"
    case "CANCELED":
      return "Canceled"
    case "ENDED":
      return "Ended"
    case "DRAFT":
      return "Draft"
    default:
      return value
  }
}

const formatUsageStatus = (value?: string) => {
  if (!value) return "-"
  switch (value.toLowerCase()) {
    case "accepted":
      return "Accepted"
    case "invalid":
      return "Invalid"
    case "enriched":
      return "Enriched"
    case "rated":
      return "Rated"
    case "unmatched_meter":
      return "Unmatched meter"
    case "unmatched_subscription":
      return "Unmatched subscription"
    default:
      return value
  }
}

const usageStatusVariant = (value?: string) => {
  switch (value?.toLowerCase()) {
    case "accepted":
    case "enriched":
    case "rated":
      return "secondary"
    case "invalid":
    case "unmatched_meter":
    case "unmatched_subscription":
      return "destructive"
    default:
      return "outline"
  }
}

const formatUsageValue = (value?: string | number | null) => {
  if (value === null || value === undefined) return "-"
  const numeric = typeof value === "number" ? value : Number(value)
  if (Number.isNaN(numeric)) return String(value)
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 6,
  }).format(numeric)
}

const statusVariant = (value?: string) => {
  switch (value?.toUpperCase()) {
    case "ACTIVE":
      return "secondary" // Green-ish usually in shadcn themes if configured, or secondary default
    case "PAUSED":
      return "outline"
    case "CANCELED":
    case "ENDED":
      return "destructive"
    case "DRAFT":
      return "secondary"
    default:
      return "outline"
  }
}

const statusTone = (value?: string) => {
  switch (value?.toUpperCase()) {
    case "ACTIVE":
      return "bg-status-success/10 text-status-success"
    case "PAUSED":
      return "bg-status-warning/10 text-status-warning"
    case "CANCELED":
    case "ENDED":
      return "bg-status-error/10 text-status-error"
    case "DRAFT":
    default:
      return "bg-bg-subtle text-text-muted"
  }
}

const formatFeatureType = (value?: string) => {
  if (!value) return "-"
  const normalized = value.toLowerCase()
  if (normalized === "metered") return "Usage-based"
  if (normalized === "boolean") return "Boolean"
  return value
}

const parseDate = (value?: string | null) => {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return date
}

// Helper Components
function StatusStepper({ currentStatus }: { currentStatus: string }) {
  const currentUpper = currentStatus.toUpperCase()
  const currentIndex = statusOrder.indexOf(currentUpper)

  return (
    <div className="flex w-full overflow-x-auto pb-2">
      <div className="flex min-w-max items-center">
        {statusOrder.map((status, index) => {
          const isCompleted = index < currentIndex
          const isCurrent = index === currentIndex
          // const isUpcoming = index > currentIndex

          return (
            <div key={status} className="flex items-center">
              <div className="flex flex-col items-center gap-2">
                <div
                  className={cn(
                    "flex h-8 w-8 items-center justify-center rounded-full border-2 text-xs font-bold transition-colors",
                    isCompleted
                      ? "border-primary bg-primary text-primary-foreground"
                      : isCurrent
                        ? "border-primary text-primary"
                        : "border-muted text-muted-foreground"
                  )}
                >
                  {isCompleted ? <CheckCircle className="h-4 w-4" /> : index + 1}
                </div>
                <span
                  className={cn(
                    "text-xs font-medium uppercase",
                    isCurrent ? "text-foreground" : "text-muted-foreground"
                  )}
                >
                  {formatStatus(status)}
                </span>
              </div>
              {index < statusOrder.length - 1 && (
                <div
                  className={cn(
                    "mx-4 h-[2px] w-12 sm:w-20",
                    index < currentIndex ? "bg-primary" : "bg-muted"
                  )}
                />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function DetailItem({
  icon: Icon,
  label,
  value,
  children,
}: {
  icon: React.ElementType
  label: string
  value?: string | React.ReactNode
  children?: React.ReactNode
}) {
  return (
    <div className="flex items-start gap-4 rounded-xl border border-border-subtle bg-bg-surface/50 p-4 shadow-sm transition-all duration-300 hover:border-border-strong hover:shadow-md group">
      <div className="rounded-lg bg-bg-primary border border-border-subtle p-2 group-hover:bg-accent-primary/5 transition-colors">
        <Icon className="h-4 w-4 text-text-muted group-hover:text-accent-primary" />
      </div>
      <div className="flex flex-col gap-1.5 min-w-0">
        <span className="text-[11px] font-semibold text-text-muted uppercase tracking-[0.18em] opacity-70 group-hover:opacity-100 transition-opacity">
          {label}
        </span>
        {children ? (
          <div className="truncate">{children}</div>
        ) : (
          <span className="text-sm font-semibold text-text-primary truncate">
            {value || "-"}
          </span>
        )}
      </div>
    </div>
  )
}

function VerticalTimeline({ items }: { items: { label: string; value: string }[] }) {
  if (items.length === 0) {
    return <div className="text-sm text-text-muted">No lifecycle events yet.</div>
  }

  // Sort items by date descending (newest first)
  // Assuming the `value` is a date string that can be parsed
  const sortedItems = [...items].sort((a, b) => {
    return new Date(b.value).getTime() - new Date(a.value).getTime()
  })

  return (
    <div className="relative space-y-0 pl-4 before:absolute before:left-[5px] before:top-2 before:h-[calc(100%-16px)] before:w-[2px] before:bg-muted">
      {sortedItems.map((item) => (
        <div key={item.label} className="relative pb-6 last:pb-0">
          <div className="absolute left-[-15px] top-1 h-3 w-3 rounded-full border-2 border-background bg-primary ring-2 ring-background" />
          <div className="flex flex-col gap-1 pl-2">
            <span className="text-xs font-medium text-muted-foreground">{item.label}</span>
            <span className="text-sm font-semibold">{formatDateTime(item.value)}</span>
          </div>
        </div>
      ))}
    </div>
  )
}

export default function OrgSubscriptionDetailPage() {
  const { orgId, subscriptionId } = useParams()
  const [subscription, setSubscription] = useState<Subscription | null>(null)
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isForbidden, setIsForbidden] = useState(false)
  const [action, setAction] = useState<ActionType | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [isActing, setIsActing] = useState(false)
  const [entitlementAsOf, setEntitlementAsOf] = useState("")
  const [usageFrom, setUsageFrom] = useState("")
  const [usageTo, setUsageTo] = useState("")
  const [usageStatus, setUsageStatus] = useState("all")
  const [usageMeter, setUsageMeter] = useState("")
  const [meters, setMeters] = useState<Meter[]>([])
  const [metersError, setMetersError] = useState<string | null>(null)
  const [isMetersLoading, setIsMetersLoading] = useState(false)

  const loadSubscription = useCallback(async () => {
    if (!subscriptionId) return
    setIsLoading(true)
    setError(null)
    setIsForbidden(false)
    try {
      const res = await admin.get(`/subscriptions/${subscriptionId}`)
      setSubscription(res.data?.data ?? null)
    } catch (err) {
      if (isForbiddenError(err)) {
        setIsForbidden(true)
      } else {
        setError(getErrorMessage(err, "Unable to load subscription."))
      }
    } finally {
      setIsLoading(false)
    }
  }, [subscriptionId])

  useEffect(() => {
    void loadSubscription()
  }, [loadSubscription])

  useEffect(() => {
    const customerId = readField(subscription, ["customer_id", "CustomerID"], "")
    if (!customerId) return
    let active = true
    admin
      .get(`/customers/${customerId}`)
      .then((response) => {
        if (!active) return
        setCustomer(response.data?.data ?? null)
      })
      .catch(() => {
        if (!active) return
        setCustomer(null)
      })
    return () => {
      active = false
    }
  }, [subscription])

  useEffect(() => {
    let active = true
    setIsMetersLoading(true)
    setMetersError(null)
    admin
      .get("/meters", {
        params: {
          active: true,
          sort_by: "code",
          order_by: "asc",
          page_size: 250,
        },
      })
      .then((response) => {
        if (!active) return
        const payload = response.data?.data
        const items = Array.isArray(payload?.items)
          ? payload.items
          : Array.isArray(payload)
            ? payload
            : Array.isArray(payload?.meters)
              ? payload.meters
              : []
        setMeters(Array.isArray(items) ? items : [])
      })
      .catch((err) => {
        if (!active) return
        setMetersError(getErrorMessage(err, "Unable to load meters."))
        setMeters([])
      })
      .finally(() => {
        if (!active) return
        setIsMetersLoading(false)
      })
    return () => {
      active = false
    }
  }, [orgId])

  const fetchEntitlements = useCallback(
    async (cursor: string | null) => {
      if (!subscriptionId) {
        return { items: [], page_info: null }
      }

      const response = await admin.get(`/subscriptions/${subscriptionId}/entitlements`, {
        params: {
          effective_at: entitlementAsOf || undefined,
          page_token: cursor || undefined,
          page_size: ENTITLEMENTS_PAGE_SIZE,
        },
      })

      const payload = response.data?.data
      const items = Array.isArray(payload?.items)
        ? payload.items
        : Array.isArray(payload)
          ? payload
          : Array.isArray(payload?.entitlements)
            ? payload.entitlements
            : []
      const pageInfo = response.data?.page_info ?? payload?.page_info ?? null

      return { items, page_info: pageInfo }
    },
    [entitlementAsOf, subscriptionId]
  )

  const {
    items: entitlements,
    error: entitlementsError,
    isLoading: isEntitlementsLoading,
    isLoadingMore: isEntitlementsLoadingMore,
    hasNext: entitlementsHasNext,
    loadNext: loadMoreEntitlements,
  } = useCursorPagination<Entitlement>(fetchEntitlements, {
    enabled: Boolean(subscriptionId),
    mode: "append",
    dependencies: [subscriptionId, entitlementAsOf],
  })

  const fetchUsageEvents = useCallback(
    async (cursor: string | null) => {
      if (!subscriptionId) {
        return { items: [], page_info: null }
      }

      const response = await admin.get("/usage", {
        params: {
          subscription_id: subscriptionId,
          meter_code: usageMeter || undefined,
          status: usageStatus === "all" ? undefined : usageStatus,
          recorded_from: usageFrom || undefined,
          recorded_to: usageTo || undefined,
          page_token: cursor || undefined,
          page_size: USAGE_PAGE_SIZE,
        },
      })

      const payload = response.data?.data
      const items = Array.isArray(payload?.items)
        ? payload.items
        : Array.isArray(payload)
          ? payload
          : Array.isArray(payload?.usage_events)
            ? payload.usage_events
            : []
      const pageInfo = response.data?.page_info ?? payload?.page_info ?? null

      return { items, page_info: pageInfo }
    },
    [subscriptionId, usageFrom, usageMeter, usageStatus, usageTo]
  )

  const {
    items: usageEvents,
    error: usageError,
    isLoading: isUsageLoading,
    isLoadingMore: isUsageLoadingMore,
    hasNext: usageHasNext,
    loadNext: loadMoreUsage,
  } = useCursorPagination<UsageEvent>(fetchUsageEvents, {
    enabled: Boolean(subscriptionId),
    mode: "append",
    dependencies: [subscriptionId, usageFrom, usageMeter, usageStatus, usageTo],
  })

  const subscriptionStatus = readField(subscription, ["status", "Status"], "DRAFT")
  const customerId = readField(subscription, ["customer_id", "CustomerID"], "")
  const customerName = readField(customer, ["name", "Name"], customerId || "Customer")
  const collectionMode = readField(subscription, ["collection_mode", "CollectionMode"])
  const billingCycleType = readField(subscription, ["billing_cycle_type", "BillingCycleType"])
  const createdAt = readField(subscription, ["created_at", "CreatedAt"], "")
  const updatedAt = readField(subscription, ["updated_at", "UpdatedAt"], "")
  const startAt = readField(subscription, ["start_at", "StartAt"], "")
  const entitlementAsOfDate = entitlementAsOf ? new Date(entitlementAsOf) : null

  const availableActions = useMemo(() => {
    const normalized = subscriptionStatus.toUpperCase()
    if (normalized === "DRAFT") return ["activate", "cancel"] as ActionType[]
    if (normalized === "ACTIVE") return ["pause", "cancel"] as ActionType[]
    if (normalized === "PAUSED") return ["resume", "cancel"] as ActionType[]
    return [] as ActionType[]
  }, [subscriptionStatus])

  const actionCopy = useMemo(() => {
    if (!action) return null
    switch (action) {
      case "activate":
        return {
          title: "Activate subscription",
          description: "Start billing immediately and open the current cycle.",
        }
      case "pause":
        return {
          title: "Pause subscription",
          description: "Stops billing and usage accrual until you resume.",
        }
      case "resume":
        return {
          title: "Resume subscription",
          description: "Resume billing and continue the current cycle.",
        }
      case "cancel":
        return {
          title: "Cancel subscription",
          description: "Irreversible. Ends future billing for this subscription.",
        }
      default:
        return null
    }
  }, [action])

  const timeline = useMemo(() => {
    if (!subscription) return []
    return [
      { label: "Created", value: readField(subscription, ["created_at", "CreatedAt"], "") },
      { label: "Activated", value: readField(subscription, ["activated_at", "ActivatedAt"], "") },
      { label: "Paused", value: readField(subscription, ["paused_at", "PausedAt"], "") },
      { label: "Resumed", value: readField(subscription, ["resumed_at", "ResumedAt"], "") },
      { label: "Canceled", value: readField(subscription, ["canceled_at", "CanceledAt"], "") },
      { label: "Ended", value: readField(subscription, ["ended_at", "EndedAt"], "") },
    ].filter((item) => item.value && item.value !== "-")
  }, [subscription])

  if (isLoading) {
    return <div className="text-text-muted text-sm">Loading subscription...</div>
  }

  if (error) {
    return <div className="text-status-error text-sm">{error}</div>
  }

  if (isForbidden) {
    return <ForbiddenState description="You do not have access to this subscription." />
  }

  if (!subscription) {
    return <div className="text-text-muted text-sm">Subscription not found.</div>
  }

  return (
    <div className="mx-auto max-w-6xl space-y-8">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to={`/orgs/${orgId}/subscriptions`}>Subscriptions</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage
              className="font-mono text-xs opacity-60"
              title={String(subscriptionId ?? "")}
            >
              {String(subscriptionId).slice(0, 12)}...
            </BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <div className="space-y-6">
        {/* Header Section */}
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
          <div className="space-y-1">
            <h1 className="flex items-center gap-4 text-3xl font-bold tracking-tight text-text-primary">
              Subscription
              <Badge
                variant={statusVariant(subscriptionStatus)}
                className={cn(
                  "text-[10px] font-bold uppercase tracking-widest px-3 py-1 rounded-full border-none",
                  statusTone(subscriptionStatus)
                )}
              >
                {formatStatus(subscriptionStatus)}
              </Badge>
            </h1>
            <p className="text-sm text-text-muted opacity-80 max-w-2xl leading-relaxed">
              Manage lifecycle and billing activity for {customerName}.
            </p>
          </div>
          <div className="flex gap-2">
            {availableActions.includes("activate") && (
              <Button onClick={() => setAction("activate")} className="gap-2">
                <Play className="h-4 w-4" /> Activate
              </Button>
            )}
            {availableActions.includes("pause") && (
              <Button variant="outline" onClick={() => setAction("pause")} className="gap-2">
                <Pause className="h-4 w-4" /> Pause
              </Button>
            )}
            {availableActions.includes("resume") && (
              <Button onClick={() => setAction("resume")} className="gap-2">
                <Play className="h-4 w-4" /> Resume
              </Button>
            )}
            {availableActions.includes("cancel") && (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setAction("cancel")}
                className="gap-2"
              >
                <XCircle className="h-4 w-4" /> Cancel
              </Button>
            )}
          </div>
        </div>

        <Separator />

        {/* Status Stepper */}
        <Card className="bg-muted/30">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base font-medium text-muted-foreground">
              <Activity className="h-4 w-4" /> Lifecycle
            </CardTitle>
          </CardHeader>
          <CardContent>
            <StatusStepper currentStatus={subscriptionStatus} />
          </CardContent>
        </Card>

        <div className="grid gap-6 lg:grid-cols-3">
          {/* Main Details - Spans 2 columns */}
          <div className="space-y-6 lg:col-span-2">
            <Card>
              <CardHeader>
                <CardTitle>Details</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4 sm:grid-cols-2">
                <DetailItem icon={User} label="Customer">
                  {customerId ? (
                    <Link
                      className="text-sm font-bold text-accent-primary hover:underline font-mono"
                      to={`/orgs/${orgId}/customers/${customerId}`}
                    >
                      {customerName}
                    </Link>
                  ) : (
                    <span className="text-sm font-bold text-text-primary">{customerName}</span>
                  )}
                </DetailItem>
                <DetailItem
                  icon={CreditCard}
                  label="Collection Mode"
                  value={collectionMode}
                />
                <DetailItem
                  icon={Clock}
                  label="Billing Cycle"
                  value={billingCycleType}
                />
                <DetailItem
                  icon={Calendar}
                  label="Start Date"
                  value={formatDateTime(startAt)}
                />
                <DetailItem
                  icon={Calendar}
                  label="Created At"
                  value={formatDateTime(createdAt)}
                />
                <DetailItem
                  icon={Calendar}
                  label="Updated"
                  value={formatDateTime(updatedAt)}
                />
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <CardTitle>Entitlements</CardTitle>
                    <p className="text-sm text-text-muted">
                      Active feature access granted by this subscription.
                    </p>
                  </div>
                  <div className="flex flex-wrap items-end gap-2">
                    <div className="space-y-1">
                      <Label htmlFor="entitlements-as-of">Effective at</Label>
                      <Input
                        id="entitlements-as-of"
                        type="date"
                        value={entitlementAsOf}
                        onChange={(event) => setEntitlementAsOf(event.target.value)}
                      />
                    </div>
                    {entitlementAsOf && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEntitlementAsOf("")}
                      >
                        Clear
                      </Button>
                    )}
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {Boolean(entitlementsError) && (
                  <div className="text-status-error text-sm">
                    {getErrorMessage(entitlementsError, "Unable to load entitlements.")}
                  </div>
                )}
                {isEntitlementsLoading && entitlements.length === 0 && (
                  <div className="text-text-muted text-sm">
                    Loading entitlements...
                  </div>
                )}
                {!isEntitlementsLoading && entitlements.length === 0 && !entitlementsError && (
                  <div className="text-text-muted text-sm">
                    No entitlements found for this subscription.
                  </div>
                )}
                {entitlements.length > 0 && (
                  <div className="rounded-lg border">
                    <Table className="min-w-[760px]">
                      <TableHeader className="[&_th]:sticky [&_th]:top-0 [&_th]:z-10 [&_th]:bg-bg-surface">
                        <TableRow>
                          <TableHead>Feature</TableHead>
                          <TableHead>Type</TableHead>
                          <TableHead>Product</TableHead>
                          <TableHead>Meter</TableHead>
                          <TableHead>Effective</TableHead>
                          <TableHead>Status</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {entitlements.map((entitlement, index) => {
                          const rowId = readField(entitlement, ["id", "ID"], "")
                          const rowKey = rowId || `entitlement-${index}`
                          const featureName = readField(entitlement, ["feature_name", "FeatureName"], "")
                          const featureCode = readField(entitlement, ["feature_code", "FeatureCode"], "")
                          const featureType = readField(entitlement, ["feature_type", "FeatureType"], "")
                          const productId = readField(entitlement, ["product_id", "ProductID"], "")
                          const meterId = readField(entitlement, ["meter_id", "MeterID"], "")
                          const effectiveFrom = readField(
                            entitlement,
                            ["effective_from", "EffectiveFrom"],
                            ""
                          )
                          const effectiveTo = readField(
                            entitlement,
                            ["effective_to", "EffectiveTo"],
                            ""
                          )

                          const asOf = entitlementAsOfDate ?? new Date()
                          const startDate = parseDate(effectiveFrom)
                          const endDate = parseDate(effectiveTo)
                          const isActive =
                            (!startDate || startDate <= asOf) &&
                            (!endDate || endDate > asOf)

                          return (
                            <TableRow key={rowKey}>
                              <TableCell>
                                <div className="flex flex-col gap-1">
                                  <span className="font-medium">
                                    {featureName || "Untitled feature"}
                                  </span>
                                  {featureCode && (
                                    <span className="text-xs text-text-muted font-mono">
                                      {featureCode}
                                    </span>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell>
                                <Badge variant="outline">{formatFeatureType(featureType)}</Badge>
                              </TableCell>
                              <TableCell className="font-mono text-xs text-text-muted">
                                {productId ? String(productId).slice(0, 12) : "-"}
                              </TableCell>
                              <TableCell className="font-mono text-xs text-text-muted">
                                {meterId ? String(meterId).slice(0, 12) : "-"}
                              </TableCell>
                              <TableCell>
                                <div className="flex flex-col gap-1 text-xs text-text-muted">
                                  <span>From {formatDateTime(effectiveFrom)}</span>
                                  <span>To {effectiveTo ? formatDateTime(effectiveTo) : "—"}</span>
                                </div>
                              </TableCell>
                              <TableCell>
                                <Badge variant={isActive ? "secondary" : "outline"}>
                                  {isActive ? "Active" : "Inactive"}
                                </Badge>
                              </TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  </div>
                )}
                {entitlementsHasNext && (
                  <div className="flex justify-end">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={loadMoreEntitlements}
                      disabled={isEntitlementsLoadingMore}
                    >
                      {isEntitlementsLoadingMore ? "Loading..." : "Load more"}
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <CardTitle>Usage</CardTitle>
                    <p className="text-sm text-text-muted">
                      Recent usage events recorded for this subscription.
                    </p>
                  </div>
                  <div className="flex flex-wrap items-end gap-2">
                    <div className="space-y-1">
                      <Label htmlFor="usage-status">Status</Label>
                      <Select value={usageStatus} onValueChange={setUsageStatus}>
                        <SelectTrigger id="usage-status" className="w-[160px]">
                          <SelectValue placeholder="All statuses" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All statuses</SelectItem>
                          <SelectItem value="accepted">Accepted</SelectItem>
                          <SelectItem value="enriched">Enriched</SelectItem>
                          <SelectItem value="rated">Rated</SelectItem>
                          <SelectItem value="invalid">Invalid</SelectItem>
                          <SelectItem value="unmatched_meter">Unmatched meter</SelectItem>
                          <SelectItem value="unmatched_subscription">
                            Unmatched subscription
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="usage-meter">Meter</Label>
                      <Select
                        value={usageMeter || "all"}
                        onValueChange={(value) => {
                          setUsageMeter(value === "all" ? "" : value)
                        }}
                        disabled={isMetersLoading}
                      >
                        <SelectTrigger id="usage-meter" className="w-[200px]">
                          <SelectValue
                            placeholder={isMetersLoading ? "Loading meters..." : "All meters"}
                          />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All meters</SelectItem>
                          {meters.map((meter, index) => {
                            const code = readField(meter, ["code", "Code"], "")
                            if (!code) return null
                            const name = readField(meter, ["name", "Name"], "")
                            const label = name && name !== "-"
                              ? `${name} (${code})`
                              : code
                            const key = code || readField(meter, ["id", "ID"], "") || `meter-${index}`
                            return (
                              <SelectItem key={key} value={code}>
                                {label}
                              </SelectItem>
                            )
                          })}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="usage-from">From</Label>
                      <Input
                        id="usage-from"
                        type="date"
                        value={usageFrom}
                        onChange={(event) => setUsageFrom(event.target.value)}
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="usage-to">To</Label>
                      <Input
                        id="usage-to"
                        type="date"
                        value={usageTo}
                        onChange={(event) => setUsageTo(event.target.value)}
                      />
                    </div>
                    {(usageFrom || usageTo) && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setUsageFrom("")
                          setUsageTo("")
                        }}
                      >
                        Clear
                      </Button>
                    )}
                    {(usageStatus !== "all" || usageMeter) && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setUsageStatus("all")
                          setUsageMeter("")
                        }}
                      >
                        Reset filters
                      </Button>
                    )}
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {metersError && (
                  <div className="text-status-error text-sm">{metersError}</div>
                )}
                {Boolean(usageError) && (
                  <div className="text-status-error text-sm">
                    {getErrorMessage(usageError, "Unable to load usage events.")}
                  </div>
                )}
                {isUsageLoading && usageEvents.length === 0 && (
                  <div className="text-text-muted text-sm">Loading usage events...</div>
                )}
                {!isUsageLoading && usageEvents.length === 0 && !usageError && (
                  <div className="text-text-muted text-sm">
                    No usage events recorded yet.
                  </div>
                )}
                {usageEvents.length > 0 && (
                  <div className="rounded-lg border">
                    <Table className="min-w-[760px]">
                      <TableHeader className="[&_th]:sticky [&_th]:top-0 [&_th]:z-10 [&_th]:bg-bg-surface">
                        <TableRow>
                          <TableHead>Meter</TableHead>
                          <TableHead>Value</TableHead>
                          <TableHead>Recorded At</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>Error</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {usageEvents.map((event, index) => {
                          const rowId = readField(event, ["id", "ID"], "")
                          const rowKey = rowId || `usage-${index}`
                          const meterCode = readField(event, ["meter_code", "MeterCode"], "")
                          const meterId = readField(event, ["meter_id", "MeterID"], "")
                          const rawValue =
                            (event as { value?: number | string }).value ??
                            (event as { Value?: number | string }).Value
                          const recordedAt = readField(
                            event,
                            ["recorded_at", "RecordedAt"],
                            ""
                          )
                          const status = readField(event, ["status", "Status"], "")
                          const errorMessage = readField(event, ["error", "Error"], "")

                          return (
                            <TableRow key={rowKey}>
                              <TableCell>
                                <div className="flex flex-col gap-1">
                                  <span className="font-medium">
                                    {meterCode || "Unknown meter"}
                                  </span>
                                  {meterId && meterId !== "-" && (
                                    <span className="text-xs text-text-muted font-mono">
                                      {String(meterId).slice(0, 12)}
                                    </span>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                {formatUsageValue(rawValue)}
                              </TableCell>
                              <TableCell className="text-xs text-text-muted">
                                {recordedAt ? formatDateTime(recordedAt) : "-"}
                              </TableCell>
                              <TableCell>
                                <Badge variant={usageStatusVariant(status)}>
                                  {formatUsageStatus(status)}
                                </Badge>
                              </TableCell>
                              <TableCell className="text-xs text-text-muted">
                                {errorMessage && errorMessage !== "-" ? (
                                  <span title={errorMessage}>{errorMessage}</span>
                                ) : (
                                  "-"
                                )}
                              </TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  </div>
                )}
                {usageHasNext && (
                  <div className="flex justify-end">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={loadMoreUsage}
                      disabled={isUsageLoadingMore}
                    >
                      {isUsageLoadingMore ? "Loading..." : "Load more"}
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Sidebar / Timeline - Spans 1 column */}
          <div className="space-y-6">
            <Card className="h-full">
              <CardHeader>
                <CardTitle>Timeline</CardTitle>
              </CardHeader>
              <CardContent>
                <VerticalTimeline items={timeline} />
              </CardContent>
            </Card>
          </div>
        </div>
      </div>

      <AlertDialog
        open={Boolean(action)}
        onOpenChange={(open) => {
          if (!open) {
            setAction(null)
            setActionError(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{actionCopy?.title ?? "Confirm action"}</AlertDialogTitle>
            <AlertDialogDescription>
              {actionCopy?.description ?? "This action changes subscription state."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {actionError && <div className="text-status-error text-sm">{actionError}</div>}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isActing}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={isActing}
              onClick={async () => {
                if (!action || !subscriptionId) return
                setIsActing(true)
                setActionError(null)
                try {
                  await admin.post(`/subscriptions/${subscriptionId}/${action}`)
                  setAction(null)
                  await loadSubscription()
                } catch (err) {
                  setActionError(getErrorMessage(err, "Unable to update subscription."))
                } finally {
                  setIsActing(false)
                }
              }}
            >
              {isActing ? "Updating..." : "Confirm"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
