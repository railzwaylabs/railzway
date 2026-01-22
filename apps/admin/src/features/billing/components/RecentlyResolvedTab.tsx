import { Loader2, History, CheckCircle, XCircle, AlertCircle } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { useRecentlyResolved } from "../hooks/useIA"
import { formatCurrency, formatDateTime } from "../utils/formatting"
import { cn } from "@/lib/utils"

export function RecentlyResolvedTab() {
  const { data, isLoading, error } = useRecentlyResolved()

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-64 items-center justify-center text-destructive">
        Failed to load resolution history
      </div>
    )
  }

  const items = data?.items || []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Showing resolved items from the last 30 days
        </p>
      </div>

      <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
        <Table>
          <TableHeader className="bg-muted/30">
            <TableRow className="hover:bg-transparent">
              <TableHead className="font-medium">Entity</TableHead>
              <TableHead className="font-medium">Outcome</TableHead>
              <TableHead className="font-medium">Resolution / Reason</TableHead>
              <TableHead className="font-medium">Resolved By</TableHead>
              <TableHead className="font-medium">Amount</TableHead>
              <TableHead className="text-right font-medium">Completed</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                  <div className="flex flex-col items-center gap-2">
                    <History className="h-8 w-8 opacity-20" />
                    <span>No recently resolved items found.</span>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              items.map((item) => {
                const isResolved = item.status === "resolved"
                const isEscalated = item.status === "escalated"
                const isReleased = item.status === "released"

                return (
                  <TableRow key={item.assignment_id} className="group">
                    <TableCell>
                      <div className="flex flex-col">
                        <span className="font-medium">{item.entity_name}</span>
                        <span className="text-[10px] text-muted-foreground uppercase font-mono tracking-wider">
                          {item.entity_type}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className={cn(
                          "flex w-fit items-center gap-1.5 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wider",
                          isResolved && "border-emerald-500/30 bg-emerald-500/10 text-emerald-600",
                          isEscalated && "border-red-500/30 bg-red-500/10 text-red-600",
                          isReleased && "border-amber-500/30 bg-amber-500/10 text-amber-600"
                        )}
                      >
                        {isResolved && <CheckCircle className="h-3 w-3" />}
                        {isEscalated && <AlertCircle className="h-3 w-3" />}
                        {isReleased && <XCircle className="h-3 w-3" />}
                        {item.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="max-w-[200px] truncate text-sm text-muted-foreground" title={item.resolution || item.release_reason}>
                        {item.resolution || item.release_reason || "—"}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm font-medium">
                      {item.resolved_by === "system" ? (
                        <span className="text-muted-foreground italic">System</span>
                      ) : (
                        item.resolved_by
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-xs tabular-nums">
                      {formatCurrency(item.amount_due_at_claim, item.currency)}
                    </TableCell>
                    <TableCell className="text-right text-xs text-muted-foreground">
                      {formatDateTime(item.resolved_at)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
