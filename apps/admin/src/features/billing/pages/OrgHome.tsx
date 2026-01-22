import {
  Activity,
  ArrowUpRight,
  CheckCircle2,
  Clock,
  CreditCard,
  FileText,
  AlertCircle,
  Zap,
} from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { formatDistanceToNow } from "date-fns"
import { getBillingCycles, getBillingActivity } from "../api/dashboard"

export default function OrgHome() {
  const { orgId } = useParams()

  const { data: cyclesData, isLoading: isLoadingCycles } = useQuery({
    queryKey: ["billing", "cycles", orgId],
    queryFn: getBillingCycles,
  })

  const { data: activityData, isLoading: isLoadingActivity } = useQuery({
    queryKey: ["billing", "activity", orgId],
    queryFn: () => getBillingActivity(10),
    refetchInterval: 30000,
  })

  // Derive current cycle from the list (assuming first is current)
  // In a real app, this logic might be more complex or explicit from backend
  const currentCycle = cyclesData?.cycles?.[0]

  if (isLoadingCycles || isLoadingActivity) {
    return (
      <div className="space-y-6 max-w-7xl mx-auto p-6">
        <Skeleton className="h-10 w-48" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Home</h1>
          <p className="text-muted-foreground text-sm">System command center · {new Date().toLocaleDateString()}</p>
        </div>
      </div>

      {/* HUD Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">

        {/* 1. Cycle Health */}
        <Card className="border-l-4 border-l-primary/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Current Cycle</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {currentCycle ? (
              <>
                <div className="text-2xl font-bold">{currentCycle.period}</div>
                <div className="flex items-center gap-2 mt-1">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${currentCycle.status === 'active' ? 'bg-emerald-100 text-emerald-800' : 'bg-gray-100 text-gray-800'
                    }`}>
                    {currentCycle.status}
                  </span>
                  <span className="text-xs text-muted-foreground">Ending soon</span>
                </div>

                <div className="mt-4 space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Accrued Revenue</span>
                    <span className="font-mono font-medium">${(currentCycle.total_revenue / 100).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Pending Invoices</span>
                    <span className="font-mono font-medium">{currentCycle.invoice_count}</span>
                  </div>
                </div>
              </>
            ) : (
              <div className="py-6 text-center text-muted-foreground text-sm">
                No active billing cycle found.
                <Button variant="link" className="mt-2 h-auto p-0 block mx-auto text-primary">Configure Schedule</Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* 2. Ops Health (Mocked for now as per plan, waiting for BillingOperations API) */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Operations Pulse</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <AlertCircle className="h-4 w-4 text-emerald-500" />
                  <span className="text-sm">System Status</span>
                </div>
                <span className="text-sm font-medium text-emerald-600">Healthy</span>
              </div>
              <div className="h-px bg-border" />
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-2xl font-bold">0</div>
                  <div className="text-xs text-muted-foreground">Failed Payments</div>
                </div>
                <div>
                  <div className="text-2xl font-bold">0</div>
                  <div className="text-xs text-muted-foreground">Overdue Invoices</div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 3. Quick Actions */}
        <Card className="bg-muted/10 border-dashed">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Quick Actions</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <Link to={`/orgs/${orgId}/billing/overview`} className="flex items-center justify-between p-2 rounded hover:bg-muted transition-colors text-sm group">
              <span className="flex items-center gap-2">
                <FileText className="h-4 w-4 text-muted-foreground group-hover:text-primary" />
                View Reports
              </span>
              <ArrowUpRight className="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
            </Link>
            <Link to={`/orgs/${orgId}/billing/operations`} className="flex items-center justify-between p-2 rounded hover:bg-muted transition-colors text-sm group">
              <span className="flex items-center gap-2">
                <Zap className="h-4 w-4 text-muted-foreground group-hover:text-primary" />
                Operations Inbox
              </span>
              <ArrowUpRight className="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
            </Link>
            <Link to={`/orgs/${orgId}/customers`} className="flex items-center justify-between p-2 rounded hover:bg-muted transition-colors text-sm group">
              <span className="flex items-center gap-2">
                <CreditCard className="h-4 w-4 text-muted-foreground group-hover:text-primary" />
                Manage Customers
              </span>
              <ArrowUpRight className="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
            </Link>
          </CardContent>
        </Card>
      </div>

      {/* Activity Feed */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Activity</CardTitle>
          <CardDescription>Live stream of billing engine events</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-6">
            {activityData?.activity?.map((group, gIndex) => (
              <div key={gIndex} className="space-y-4">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">{group.title}</h4>
                <div className="space-y-4">
                  {group.activities.map((item, i) => (
                    <div key={i} className="flex items-start gap-4 group">
                      <div className="mt-1 h-2 w-2 rounded-full bg-primary/20 group-hover:bg-primary transition-colors shrink-0" />
                      <div className="flex-1 space-y-1">
                        <p className="text-sm font-medium leading-none">{item.message}</p>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <span className="font-mono bg-muted px-1 rounded">{item.action}</span>
                          <span>·</span>
                          <span>{formatDistanceToNow(new Date(item.occurred_at), { addSuffix: true })}</span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
            {(!activityData?.activity || activityData.activity.length === 0) && (
              <div className="text-center py-8 text-muted-foreground">
                <CheckCircle2 className="h-8 w-8 mx-auto mb-2 opacity-20" />
                <p>No recent activity recorded.</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
