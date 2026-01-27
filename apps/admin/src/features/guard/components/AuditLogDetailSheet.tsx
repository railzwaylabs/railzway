import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
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
      <SheetContent className="w-[400px] sm:w-[600px] flex flex-col h-full bg-background border-l">
        <SheetHeader className="pb-6 border-b">
          <div className="flex items-center gap-2 mb-2">
            <Badge variant="outline" className="font-mono text-xs uppercase tracking-wider">
              {id.slice(0, 8)}
            </Badge>
            <Badge variant="secondary" className="capitalize">
              {actorType === "api_key" ? "API Key" : actorType}
            </Badge>
          </div>
          <SheetTitle className="text-xl font-mono break-all">
            {action}
          </SheetTitle>
          <SheetDescription className="flex items-center gap-2">
            <IconClock className="h-3.5 w-3.5" />
            {formatTimestamp(createdAt)}
          </SheetDescription>
        </SheetHeader>

        <ScrollArea className="flex-1 -mx-6 px-6 py-6">
          <div className="space-y-8">
            {/* Actor Context */}
            <section className="space-y-4">
              <h3 className="text-xs font-bold text-text-muted uppercase tracking-wider flex items-center gap-2">
                <IconUser className="h-4 w-4" /> Authenticated Actor
              </h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="bg-bg-subtle/30 p-3 rounded-md border border-border-subtle">
                  <span className="text-xs text-text-muted icon-center gap-1 mb-1">Actor ID</span>
                  <div className="font-mono text-xs break-all">{actorID}</div>
                </div>
                <div className="bg-bg-subtle/30 p-3 rounded-md border border-border-subtle">
                  <span className="text-xs text-text-muted icon-center gap-1 mb-1">IP Address</span>
                  <div className="font-mono text-xs">{ipAddress}</div>
                </div>
              </div>
            </section>

            <Separator />

            {/* Resource Context */}
            <section className="space-y-4">
              <h3 className="text-xs font-bold text-text-muted uppercase tracking-wider flex items-center gap-2">
                <IconHash className="h-4 w-4" /> Affected Resource
              </h3>
              <div className="space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-text-muted">Type</span>
                  <span className="font-medium capitalize">{targetType.replace(/_/g, " ")}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-text-muted">ID</span>
                  <span className="font-mono text-xs">{targetID}</span>
                </div>
              </div>
            </section>

            <Separator />

            {detailRows.length > 0 && (
              <section className="space-y-4">
                <h3 className="text-xs font-bold text-text-muted uppercase tracking-wider flex items-center gap-2">
                  <IconActivity className="h-4 w-4" /> Event Details
                </h3>
                <div className="space-y-3">
                  {detailRows.map((row) => (
                    <div key={row.label} className="flex justify-between text-sm gap-6">
                      <span className="text-text-muted">{row.label}</span>
                      <span className="font-medium text-right break-all">{row.value}</span>
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
              <h3 className="text-xs font-bold text-text-muted uppercase tracking-wider flex items-center gap-2">
                <IconCode className="h-4 w-4" /> Raw Metadata
              </h3>
              <div className="rounded-md border border-border-subtle bg-bg-subtle/10 p-4">
                <pre className="text-[10px] font-mono whitespace-pre-wrap break-all text-text-secondary">
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
