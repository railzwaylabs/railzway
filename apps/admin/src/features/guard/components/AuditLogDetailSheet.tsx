import {
  Sheet,
  SheetContent,
} from "@/components/ui/sheet"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { ScrollArea } from "@/components/ui/scroll-area"
import { IconClock, IconUser, IconHash, IconActivity, IconCode } from "@tabler/icons-react"

type AuditLog = Record<string, unknown>

// Helper helpers reused from main page
const readField = (item: any, fields: string[]) => {
  if (!item) return undefined
  for (const field of fields) {
    if (field in item) return item[field]
  }
  return undefined
}

const formatTimestamp = (value?: string | null) => {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZoneName: "short",
  }).format(date)
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

interface AuditLogDetailSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  log: AuditLog | null
}

export function AuditLogDetailSheet({ open, onOpenChange, log }: AuditLogDetailSheetProps) {
  if (!log) return null

  const id = String(readField(log, ["id", "ID"]) ?? "-")
  const createdAt = String(readField(log, ["created_at", "CreatedAt"]) ?? "")
  const action = String(readField(log, ["action", "Action"]) ?? "")
  const actorType = String(readField(log, ["actor_type", "ActorType"]) ?? "").toLowerCase()
  const actorID = String(readField(log, ["actor_id", "ActorID"]) ?? "-")
  const ipAddress = String(readField(log, ["ip_address", "IPAddress"]) ?? "-")

  const targetType = String(readField(log, ["target_type", "TargetType"]) ?? "")
  const targetID = String(readField(log, ["target_id", "TargetID"]) ?? "-")

  const metadata = getMetadata(log)
  const fromStatus = readMetadataValue(metadata, ["from_status", "fromStatus", "previous_status", "prev_status"])
  const toStatus = readMetadataValue(metadata, ["to_status", "toStatus", "next_status"])
  const status = readMetadataValue(metadata, ["status", "current_status"])
  const reason = readMetadataValue(metadata, ["reason", "failure_reason"])
  const error = readMetadataValue(metadata, ["error", "error_message"])
  const amount = readMetadataValue(metadata, ["amount", "amount_value"])
  const currency = readMetadataValue(metadata, ["currency", "currency_code"])
  const occurredAt = readMetadataValue(metadata, ["at", "occurred_at", "occurredAt", "timestamp"])

  const diffBefore = metadata.before ?? metadata.old_values
  const diffAfter = metadata.after ?? metadata.new_values
  const hasDiff = typeof diffBefore !== 'undefined' || typeof diffAfter !== 'undefined'
  const statusLabel = (typeof fromStatus === "string" && typeof toStatus === "string" && fromStatus && toStatus)
    ? `${fromStatus} → ${toStatus}`
    : (typeof status === "string" && status.trim() ? status : undefined)

  const detailRows = [
    { label: "Status", value: statusLabel },
    { label: "Reason", value: typeof reason === "string" ? reason : undefined },
    { label: "Error", value: typeof error === "string" ? error : undefined },
    { label: "Amount", value: typeof amount === "string" || typeof amount === "number" ? String(amount) : undefined },
    { label: "Currency", value: typeof currency === "string" ? currency : undefined },
    { label: "Occurred At", value: typeof occurredAt === "string" ? formatTimestamp(occurredAt) : undefined },
  ].filter((row) => row.value)

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[400px] sm:w-[600px] flex flex-col h-full bg-bg-primary border-l overflow-hidden p-0 gap-0">
        {/* Sticky Header with Glassmorphism */}
        <div className="sticky top-0 z-20 bg-bg-primary/80 backdrop-blur-md border-b px-6 py-6 transition-all duration-200">
          <div className="flex items-center gap-2 mb-3">
            <Badge variant="outline" className="font-mono text-[10px] uppercase tracking-widest bg-white/50 border-border-subtle text-text-muted px-2">
              {id.slice(0, 8)}
            </Badge>
            <Badge variant="secondary" className="capitalize text-[10px] font-bold px-2 rounded-full border-none bg-accent-primary/10 text-accent-primary">
              {actorType === "api_key" ? "API Key" : actorType}
            </Badge>
          </div>
          <h2 className="text-xl font-mono font-bold tracking-tight text-text-primary mb-1 break-all">
            {action}
          </h2>
          <div className="flex items-center gap-2 text-text-muted text-xs">
            <IconClock className="h-3.5 w-3.5 opacity-60" />
            <span className="opacity-80 font-medium">{formatTimestamp(createdAt)}</span>
          </div>
        </div>

        <ScrollArea className="flex-1 min-h-0 w-full">
          <div className="px-6 py-10 space-y-12 pb-32">
            {/* Actor Context */}
            <section className="space-y-5">
              <h3 className="text-[11px] font-bold text-text-muted uppercase tracking-[0.1em] flex items-center gap-2">
                <IconUser className="h-4 w-4 opacity-50" /> Authenticated Actor
              </h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="bg-bg-subtle/50 p-5 rounded-xl border border-border-subtle shadow-sm transition-all hover:border-border-strong hover:shadow-md">
                  <span className="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1.5 opacity-70">Actor ID</span>
                  <div className="font-mono text-xs break-all text-text-primary leading-relaxed">{actorID}</div>
                </div>
                <div className="bg-bg-subtle/50 p-5 rounded-xl border border-border-subtle shadow-sm transition-all hover:border-border-strong hover:shadow-md">
                  <span className="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1.5 opacity-70">IP Address</span>
                  <div className="font-mono text-xs text-text-primary leading-relaxed">{ipAddress}</div>
                </div>
              </div>
            </section>

            <Separator />

            {/* Resource Context */}
            <section className="space-y-4">
              <h3 className="text-[11px] font-bold text-text-muted uppercase tracking-[0.1em] flex items-center gap-2">
                <IconHash className="h-4 w-4 opacity-50" /> Affected Resource
              </h3>
              <div className="bg-bg-subtle/50 p-5 rounded-xl border border-border-subtle shadow-sm flex flex-col gap-4">
                <div className="flex justify-between items-center text-sm">
                  <span className="text-[10px] font-bold text-text-muted uppercase tracking-widest opacity-70">Type</span>
                  <Badge variant="secondary" className="font-medium capitalize text-text-primary px-2.5 py-0.5 rounded-full border-none bg-accent-primary/10 text-accent-primary">
                    {targetType.replace(/_/g, " ")}
                  </Badge>
                </div>
                <div className="flex flex-col gap-1.5">
                  <span className="text-[10px] font-bold text-text-muted uppercase tracking-widest opacity-70">Resource ID</span>
                  <div className="font-mono text-xs text-text-primary bg-white/50 p-2.5 rounded border border-border-subtle break-all">
                    {targetID}
                  </div>
                </div>
              </div>
            </section>

            <Separator />

            {detailRows.length > 0 && (
              <section className="space-y-4">
                <h3 className="text-[11px] font-bold text-text-muted uppercase tracking-[0.1em] flex items-center gap-2">
                  <IconActivity className="h-4 w-4 opacity-50" /> Event Details
                </h3>
                <div className="bg-bg-subtle/50 p-5 rounded-xl border border-border-subtle shadow-sm divide-y divide-border-subtle/50">
                  {detailRows.map((row) => (
                    <div key={row.label} className="flex justify-between items-center py-3 first:pt-0 last:pb-0 gap-6">
                      <span className="text-[10px] font-bold text-text-muted uppercase tracking-widest opacity-70">{row.label}</span>
                      <span className="text-sm font-medium text-text-primary text-right break-all">{row.value}</span>
                    </div>
                  ))}
                </div>
              </section>
            )}

            {detailRows.length > 0 && <Separator />}

            {/* State Change Diff */}
            {hasDiff && (
              <section className="space-y-4">
                <h3 className="text-xs font-bold text-text-muted uppercase tracking-wider flex items-center gap-2">
                  <IconActivity className="h-4 w-4" /> State Change
                </h3>
                <div className="grid grid-cols-1 gap-4">
                  <div className="rounded border border-border-subtle overflow-hidden">
                    <div className="bg-bg-subtle/50 px-3 py-1.5 border-b border-border-subtle text-xs font-medium text-text-muted">
                      Before
                    </div>
                    <pre className="p-3 bg-bg-subtle/10 text-[10px] font-mono overflow-x-auto text-text-muted">
                      {diffBefore ? JSON.stringify(diffBefore, null, 2) : "null"}
                    </pre>
                  </div>
                  <div className="rounded border border-border-subtle overflow-hidden">
                    <div className="bg-bg-subtle/50 px-3 py-1.5 border-b border-border-subtle text-xs font-medium text-text-primary">
                      After
                    </div>
                    <pre className="p-3 bg-bg-subtle/10 text-[10px] font-mono overflow-x-auto text-text-primary">
                      {diffAfter ? JSON.stringify(diffAfter, null, 2) : "null"}
                    </pre>
                  </div>
                </div>
              </section>
            )}

            {/* Full Metadata */}
            <section className="space-y-4">
              <h3 className="text-[11px] font-bold text-text-muted uppercase tracking-[0.1em] flex items-center gap-2">
                <IconCode className="h-4 w-4 opacity-50" /> Raw Metadata
              </h3>
              <div className="rounded-xl border border-border-subtle bg-bg-surface p-5 shadow-inner">
                <pre className="text-[11px] font-mono whitespace-pre-wrap break-all text-text-primary leading-relaxed">
                  {JSON.stringify(metadata, null, 2)}
                </pre>
              </div>
            </section>
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
