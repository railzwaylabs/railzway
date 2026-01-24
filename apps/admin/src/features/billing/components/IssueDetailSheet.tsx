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
import { FileText, User, Calendar, Clock, AlertTriangle, CheckCircle2 } from "lucide-react"
import { formatCurrency, formatAssignmentAge } from "../utils/formatting"
import { cn } from "@/lib/utils"

interface IssueDetailSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  issue: any // Type this strictly if possible, using 'any' for speed to match existing implicit types
}

export function IssueDetailSheet({ open, onOpenChange, issue }: IssueDetailSheetProps) {
  if (!issue) return null

  const isOverdue = (issue.days_overdue || issue.days_overdue_at_claim) > 0
  const amount = issue.amount_due ?? issue.amount_due_at_claim ?? 0
  const currency = issue.currency || "USD"

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[400px] sm:w-[540px] flex flex-col h-full bg-background border-l">
        <SheetHeader className="pb-6 border-b">
          <div className="flex items-center gap-2 mb-2">
            <Badge variant="outline" className="capitalize">
              {issue.entity_type}
            </Badge>
            <Badge
              variant="secondary"
              className={cn(
                "capitalize",
                issue.risk_category === "high_exposure" ? "bg-indigo-100 text-indigo-700" :
                  issue.risk_category === "failed_payment" ? "bg-red-100 text-red-700" :
                    "bg-slate-100 text-slate-700"
              )}
            >
              {(issue.risk_category || issue.category || "General").replace(/_/g, " ")}
            </Badge>
          </div>
          <SheetTitle className="text-xl font-mono">
            {issue.entity_name}
          </SheetTitle>
          <SheetDescription>
            {issue.customer_name || issue.entity_name}
          </SheetDescription>
        </SheetHeader>

        <ScrollArea className="flex-1 -mx-6 px-6 py-6">
          <div className="space-y-8">
            {/* Snapshot Section */}
            <section className="space-y-4">
              <h3 className="text-sm font-semibold tracking-wide text-muted-foreground uppercase">
                Financial Snapshot
              </h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="p-4 rounded-lg bg-muted/40 border">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                    <FileText className="h-4 w-4" /> Amount Due
                  </div>
                  <div className="text-2xl font-bold tabular-nums">
                    {formatCurrency(amount, currency)}
                  </div>
                </div>
                <div className={cn(
                  "p-4 rounded-lg border",
                  isOverdue ? "bg-amber-500/5 border-amber-500/20" : "bg-muted/40 border-muted"
                )}>
                  <div className={cn("flex items-center gap-2 text-sm mb-1", isOverdue ? "text-amber-700" : "text-muted-foreground")}>
                    <Clock className="h-4 w-4" /> Age
                  </div>
                  <div className={cn("text-2xl font-bold tabular-nums", isOverdue ? "text-amber-700" : "")}>
                    {issue.days_overdue ?? issue.days_overdue_at_claim ?? 0} days
                  </div>
                </div>
              </div>
            </section>

            <Separator />

            {/* Context Section */}
            <section className="space-y-4">
              <h3 className="text-sm font-semibold tracking-wide text-muted-foreground uppercase">
                Operational Context
              </h3>
              <div className="space-y-3">
                <div className="flex justify-between items-center text-sm">
                  <span className="flex items-center gap-2 text-muted-foreground">
                    <User className="h-4 w-4" /> Customer
                  </span>
                  <span className="font-medium max-w-[200px] truncate">{issue.customer_name || "N/A"}</span>
                </div>
                <div className="flex justify-between items-center text-sm">
                  <span className="flex items-center gap-2 text-muted-foreground">
                    <Calendar className="h-4 w-4" /> Created At
                  </span>
                  <span className="font-mono text-xs">
                    {new Date(issue.created_at || issue.claimed_at).toLocaleDateString()}
                  </span>
                </div>
                {issue.claimed_at && (
                  <div className="flex justify-between items-center text-sm">
                    <span className="flex items-center gap-2 text-purple-600 font-medium">
                      <CheckCircle2 className="h-4 w-4" /> Claimed
                    </span>
                    <span className="text-muted-foreground">
                      {formatAssignmentAge(issue.claimed_at)} ago
                    </span>
                  </div>
                )}
              </div>
            </section>

            <Separator />

            {/* Read-Only Notice */}
            <div className="bg-slate-50 border border-slate-200 rounded-md p-4 text-sm text-slate-600 flex gap-3 items-start">
              <AlertTriangle className="h-5 w-5 shrink-0 text-slate-400" />
              <div>
                <p className="font-medium text-slate-900 mb-1">Immutable Ledger Record</p>
                <p>
                  This billing event is final. You cannot modify the amount or dates.
                  Resolution actions (Credit Note, Void, Collection) must be performed via specific workflows.
                </p>
              </div>
            </div>
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
