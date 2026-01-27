import { useCallback, useEffect, useState } from "react"
import { Link, useParams } from "react-router-dom"
import {
  IconExternalLink,
  IconSearch,
  IconDownload,
  IconClock,
  IconUser,
  IconRobot,
  IconKey,
} from "@tabler/icons-react"

import { admin } from "@/api/client"
import { ForbiddenState } from "@/components/forbidden-state"
import { RestrictedFeature } from "@/components/RestrictedFeature"
import { AuditLogDetailSheet } from "../components/AuditLogDetailSheet"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { getErrorMessage, isForbiddenError } from "@/lib/api-errors"

type AuditLog = Record<string, unknown>

// Enterprise-grade timestamp formatting (ISO-ish but readable)
const formatTimestamp = (value?: string | null) => {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZoneName: "short",
  }).format(date)
}

const formatDateInput = (value: string) => {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ""
  return date.toISOString()
}

const readField = <T extends Record<string, unknown>>(
  item: T | undefined,
  fields: Array<keyof T | string>
) => {
  if (!item) return undefined
  for (const field of fields) {
    if (field in item) {
      return item[field as keyof T]
    }
  }
  return undefined
}

const getMetadata = (log?: AuditLog) => {
  const raw = readField(log, ["metadata", "Metadata"])
  if (raw && typeof raw === "object" && !Array.isArray(raw)) {
    return raw as Record<string, unknown>
  }
  return {}
}

const readMetadataValue = (
  metadata: Record<string, unknown>,
  fields: string[]
) => {
  for (const field of fields) {
    if (field in metadata) {
      return metadata[field]
    }
  }
  return undefined
}

const humanizeAction = (action: string) => {
  if (!action) return "-"
  // Split by dot, capitalize first letter of each word
  // e.g. "invoice.finalized" -> "Invoice Finalized"
  return action
    .split(".")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ")
}

// Actor Representation Component
const ActorCell = ({ log }: { log: AuditLog }) => {
  const actorType = String(readField(log, ["actor_type", "ActorType"]) ?? "").toLowerCase()
  const actorID = String(readField(log, ["actor_id", "ActorID"]) ?? "")

  if (!actorType) return <span className="text-text-muted">-</span>

  let Icon = IconUser
  let label = "USER"

  if (actorType === "system") {
    Icon = IconRobot
    label = "SYSTEM"
  } else if (actorType === "api_key") {
    Icon = IconKey
    label = "API_KEY"
  }

  return (
    <div className="flex flex-col text-xs">
      <div className="flex items-center gap-1.5 font-medium mb-0.5">
        <Icon className="h-3 w-3 opacity-70" />
        <span className="uppercase tracking-wider text-[10px] text-text-muted font-bold">{label}</span>
      </div>
      {(actorType === "user" || actorType === "api_key") && actorID && (
        <span className="font-mono text-[10px] text-text-primary ml-4 truncate max-w-[150px]" title={actorID}>
          {actorID}
        </span>
      )}
    </div>
  )
}

const buildSummary = (log: AuditLog) => {
  const metadata = getMetadata(log)
  const summaryParts = []

  const fromStatus = readMetadataValue(metadata, ["from_status", "fromStatus", "previous_status", "prev_status"])
  const toStatus = readMetadataValue(metadata, ["to_status", "toStatus", "next_status"])
  const status = readMetadataValue(metadata, ["status", "current_status"])

  if (typeof fromStatus === "string" && typeof toStatus === "string" && fromStatus && toStatus) {
    summaryParts.push(`Status: ${fromStatus} → ${toStatus}`)
  } else if (typeof status === "string" && status.trim()) {
    summaryParts.push(`Status: ${status}`)
  }

  if (metadata.error) summaryParts.push(`Error: ${metadata.error}`)
  if (metadata.reason) summaryParts.push(`Reason: ${metadata.reason}`)

  // Specific financial context
  if (metadata.amount) summaryParts.push(`Amount: ${metadata.amount}`)
  if (metadata.currency) summaryParts.push(`${metadata.currency}`)

  if (summaryParts.length > 0) return summaryParts.join(" · ")
  return "—"
}

const buildTargetLink = (orgId: string | undefined, log: AuditLog) => {
  if (!orgId) return null
  const targetType = String(readField(log, ["target_type", "TargetType"]) ?? "")
  const targetID = String(readField(log, ["target_id", "TargetID"]) ?? "")
  if (!targetType || !targetID) return null
  switch (targetType) {
    case "invoice":
      return `/orgs/${orgId}/invoices/${targetID}`
    case "subscription":
      return `/orgs/${orgId}/subscriptions/${targetID}`
    case "customer":
      return `/orgs/${orgId}/customers/${targetID}`
    case "api_key":
      return `/orgs/${orgId}/api-keys`
    default:
      return null
  }
}

const actorTypeOptions = [
  { value: "user", label: "User" },
  { value: "system", label: "System" },
  { value: "api_key", label: "API Key" },
]

const ALL_FILTER_VALUE = "__all__"

const resourceTypeOptions = [
  { value: "product", label: "Product" },
  { value: "price", label: "Price" },
  { value: "customer", label: "Customer" },
  { value: "subscription", label: "Subscription" },
  { value: "invoice", label: "Invoice" },
  { value: "payment_method", label: "Payment Method" },
  { value: "api_key", label: "API Key" },
  { value: "user", label: "User" },
]

export default function OrgAuditLogsPage() {
  const { orgId } = useParams()
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isForbidden, setIsForbidden] = useState(false)
  const [pageToken, setPageToken] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(false)

  const [filters, setFilters] = useState({
    action: "",
    resourceType: "",
    resourceID: "",
    actorType: "",
    startAt: "",
    endAt: "",
  })
  const [appliedFilters, setAppliedFilters] = useState(filters)

  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null)

  const loadLogs = useCallback(
    async (nextToken?: string | null, append = false) => {
      if (!orgId) {
        setIsLoading(false)
        return
      }

      setIsLoading(true)
      setError(null)
      setIsForbidden(false)

      const params: Record<string, string | number> = {
        page_size: 50,
      }
      if (appliedFilters.action.trim()) {
        params.action = appliedFilters.action.trim()
      }
      if (appliedFilters.resourceType) {
        params.resource_type = appliedFilters.resourceType
      }
      if (appliedFilters.resourceID.trim()) {
        params.resource_id = appliedFilters.resourceID.trim()
      }
      if (appliedFilters.actorType) {
        params.actor_type = appliedFilters.actorType
      }
      if (appliedFilters.startAt) {
        const startAt = formatDateInput(appliedFilters.startAt)
        if (startAt) {
          params.from = startAt
        }
      }
      if (appliedFilters.endAt) {
        const endAt = formatDateInput(appliedFilters.endAt)
        if (endAt) {
          params.to = endAt
        }
      }
      if (nextToken) {
        params.page_token = nextToken
      }

      try {
        const res = await admin.get("/audit-logs", { params })
        const payload = res.data ?? {}
        const data = Array.isArray(payload.data) ? payload.data : []
        const info = payload.page_info ?? {}
        const next = readField(info, [
          "next_page_token",
          "NextPageToken",
          "nextPageToken",
        ])
        const more = readField(info, ["has_more", "HasMore", "hasMore"])
        setPageToken(typeof next === "string" ? next : null)
        setHasMore(Boolean(more))
        setLogs((prev) => (append ? [...prev, ...data] : data))
      } catch (err: any) {
        if (isForbiddenError(err)) {
          setIsForbidden(true)
        } else {
          setError(getErrorMessage(err, "Unable to load audit logs."))
        }
        if (!append) {
          setLogs([])
        }
      } finally {
        setIsLoading(false)
      }
    },
    [appliedFilters, orgId]
  )

  useEffect(() => {
    void loadLogs(null, false)
  }, [loadLogs])

  const handleApply = () => {
    setAppliedFilters(filters)
  }

  const handleClear = () => {
    const cleared = {
      action: "",
      resourceType: "",
      resourceID: "",
      actorType: "",
      startAt: "",
      endAt: "",
    }
    setFilters(cleared)
    setAppliedFilters(cleared)
  }



  if (isForbidden) {
    return <ForbiddenState description="You do not have access to audit logs." />
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4 border-b border-border-subtle pb-6">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Audit Log</h1>
          <p className="text-text-muted text-sm max-w-2xl">
            A complete, immutable record of all billing, security, and administrative events.
            Used for compliance and dispute resolution.
          </p>
        </div>
        <div className="flex-shrink-0">
          <RestrictedFeature feature="audit_export" description="Exporting audit logs as CSV/JSON is available in Railzway Plus.">
            <Button variant="outline" className="h-9">
              <IconDownload className="mr-2 h-4 w-4 opacity-70" />
              Export
            </Button>
          </RestrictedFeature>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-bg-subtle/30 rounded-md p-4 border border-border-subtle space-y-3">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2 flex-1 min-w-[200px]">
            <IconSearch className="h-4 w-4 text-text-muted" />
            <Input
              className="h-9 bg-bg-surface"
              placeholder="Action (e.g. invoice.finalize)"
              value={filters.action}
              onChange={(event) => setFilters((prev) => ({ ...prev, action: event.target.value }))}
            />
          </div>
          <div className="flex items-center gap-2 flex-1 min-w-[200px]">
            <span className="text-xs text-text-muted font-mono">ID:</span>
            <Input
              className="h-9 bg-bg-surface font-mono"
              placeholder="Resource ID"
              value={filters.resourceID}
              onChange={(event) => setFilters((prev) => ({ ...prev, resourceID: event.target.value }))}
            />
          </div>
          <Select
            value={filters.resourceType || ALL_FILTER_VALUE}
            onValueChange={(value) => setFilters((prev) => ({ ...prev, resourceType: value === ALL_FILTER_VALUE ? "" : value }))}
          >
            <SelectTrigger className="h-9 w-[180px] bg-bg-surface">
              <SelectValue placeholder="Resource" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_FILTER_VALUE}>All Resources</SelectItem>
              {resourceTypeOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={filters.actorType || ALL_FILTER_VALUE}
            onValueChange={(value) => setFilters((prev) => ({ ...prev, actorType: value === ALL_FILTER_VALUE ? "" : value }))}
          >
            <SelectTrigger className="h-9 w-[140px] bg-bg-surface">
              <SelectValue placeholder="Actor" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_FILTER_VALUE}>All Actors</SelectItem>
              {actorTypeOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-border-subtle/50">
          <div className="flex items-center gap-2">
            <span className="text-xs text-text-muted flex items-center gap-1">
              <IconClock className="h-3 w-3" /> Time Range:
            </span>
            <Input
              type="datetime-local"
              className="h-8 w-auto min-w-[180px] bg-bg-surface text-xs"
              value={filters.startAt}
              onChange={(event) => setFilters((prev) => ({ ...prev, startAt: event.target.value }))}
            />
            <span className="text-text-muted text-xs">to</span>
            <Input
              type="datetime-local"
              className="h-8 w-auto min-w-[180px] bg-bg-surface text-xs"
              value={filters.endAt}
              onChange={(event) => setFilters((prev) => ({ ...prev, endAt: event.target.value }))}
            />
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={handleClear} className="h-8 text-text-muted hover:text-text-primary">
              Clear
            </Button>
            <Button size="sm" onClick={handleApply} className="h-8 px-4">
              Apply Filters
            </Button>
          </div>
        </div>
      </div>

      {isLoading && (
        <div className="py-12 flex justify-center text-text-muted text-sm">
          <span className="animate-pulse">Loading audit trail...</span>
        </div>
      )}
      {error && (
        <div className="p-4 rounded-md bg-status-error/10 border border-status-error/20 text-status-error text-sm">
          {error}
        </div>
      )}

      {!isLoading && !error && logs.length === 0 && (
        <div className="rounded-lg border border-dashed border-border-subtle p-12 text-center">
          <p className="text-text-muted text-sm">No audit events found matching the current criteria.</p>
        </div>
      )}

      {!error && logs.length > 0 && (
        <div className="rounded-md border border-border-subtle bg-bg-surface overflow-hidden">
          <Table>
            <TableHeader className="bg-bg-subtle/50">
              <TableRow>
                <TableHead className="w-[180px]">Timestamp</TableHead>
                <TableHead className="w-[200px]">Actor</TableHead>
                <TableHead className="w-[200px]">Action</TableHead>
                <TableHead className="w-[150px]">Resource</TableHead>
                <TableHead className="w-[150px]">Target ID</TableHead>
                <TableHead>Summary</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => {
                const createdAt = String(readField(log, ["created_at", "CreatedAt"]) ?? "")
                const action = String(readField(log, ["action", "Action"]) ?? "")
                const targetID = String(readField(log, ["target_id", "TargetID"]) ?? "")
                const targetHref = buildTargetLink(orgId, log)
                const resourceLabel = String(readField(log, ["target_type", "TargetType"]) ?? "")

                return (
                  <TableRow
                    key={String(readField(log, ["id", "ID"]) ?? createdAt)}
                    className="cursor-pointer hover:bg-bg-subtle/40 border-b border-border-subtle/50"
                    onClick={() => setSelectedLog(log)}
                  >
                    <TableCell className="text-text-muted text-xs font-mono align-top py-3">
                      {formatTimestamp(createdAt)}
                    </TableCell>
                    <TableCell className="align-top py-3">
                      <ActorCell log={log} />
                    </TableCell>
                    <TableCell className="align-top py-3">
                      <div className="font-medium text-sm text-text-primary">{humanizeAction(action)}</div>
                      <div className="text-xs text-text-muted font-mono mt-0.5 opacity-60">{action}</div>
                    </TableCell>
                    <TableCell className="text-text-primary text-xs align-top py-3 capitalize">
                      {resourceLabel.replace(/_/g, " ")}
                    </TableCell>
                    <TableCell className="align-top py-3">
                      {targetHref ? (
                        <Link
                          to={targetHref}
                          className="inline-flex items-center gap-1 font-mono text-xs text-accent-primary hover:underline"
                          onClick={(event) => event.stopPropagation()}
                        >
                          {targetID.slice(0, 12)}...
                          <IconExternalLink className="h-3 w-3" />
                        </Link>
                      ) : (
                        <span className="font-mono text-xs text-text-muted" title={targetID}>
                          {targetID ? `${targetID.slice(0, 12)}...` : "-"}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-text-muted text-xs align-top py-3">
                      {buildSummary(log)}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {hasMore && (
        <div className="flex items-center justify-center pt-4">
          <Button
            variant="ghost"
            size="sm"
            disabled={isLoading}
            onClick={() => loadLogs(pageToken, true)}
            className="text-text-muted hover:text-text-primary"
          >
            Load older events
          </Button>
        </div>
      )}

      {/* Drill-down Detail View */}
      <AuditLogDetailSheet
        open={!!selectedLog}
        onOpenChange={(open) => !open && setSelectedLog(null)}
        log={selectedLog}
      />
    </div>
  )
}
