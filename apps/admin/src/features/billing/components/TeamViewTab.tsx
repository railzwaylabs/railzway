import { Loader2, Users, Briefcase, Clock, AlertTriangle, CheckCircle } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { useTeamView } from "../hooks/useIA"
import { formatCurrency } from "../utils/formatting"

export function TeamViewTab() {
  const { data, isLoading, error } = useTeamView()

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
        Failed to load team workload data
      </div>
    )
  }

  const { team_members = [], summary } = data || {}

  return (
    <div className="space-y-8">
      {/* Summary Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border/60 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Total Active
            </CardTitle>
            <Briefcase className="h-4 w-4 text-blue-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{summary?.total_active_assignments || 0}</div>
            <p className="text-[10px] text-muted-foreground mt-1">Assignments currently claimed</p>
          </CardContent>
        </Card>

        <Card className="border-border/60 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Total Exposure
            </CardTitle>
            <CheckCircle className="h-4 w-4 text-emerald-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatCurrency(summary?.total_exposure || 0, summary?.currency)}
            </div>
            <p className="text-[10px] text-muted-foreground mt-1">Value under management</p>
          </CardContent>
        </Card>

        <Card className="border-border/60 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Avg. Age
            </CardTitle>
            <Clock className="h-4 w-4 text-amber-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{summary?.average_assignment_age_minutes || 0}m</div>
            <p className="text-[10px] text-muted-foreground mt-1">Global average response time</p>
          </CardContent>
        </Card>

        <Card className="border-border/60 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Escalations
            </CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{summary?.escalation_count || 0}</div>
            <p className="text-[10px] text-muted-foreground mt-1">Active SLA breaches</p>
          </CardContent>
        </Card>
      </div>

      {/* Team Table */}
      <div className="space-y-4">
        <h3 className="text-sm font-semibold flex items-center gap-2">
          <Users className="h-4 w-4" />
          Operator Workloads
        </h3>

        <div className="rounded-xl border border-border/60 bg-card shadow-sm overflow-hidden">
          <Table>
            <TableHeader className="bg-muted/30">
              <TableRow className="hover:bg-transparent">
                <TableHead className="font-medium">Team Member</TableHead>
                <TableHead className="font-medium">Active Tasks</TableHead>
                <TableHead className="font-medium">Owned Exposure</TableHead>
                <TableHead className="font-medium">Avg. Task Age</TableHead>
                <TableHead className="font-medium">Escalations</TableHead>
                <TableHead className="text-right font-medium">Resolutions (30d)</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {team_members.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                    No team member activity recorded.
                  </TableCell>
                </TableRow>
              ) : (
                team_members.map((member) => (
                  <TableRow key={member.user_id}>
                    <TableCell className="font-medium">{member.user_name}</TableCell>
                    <TableCell>{member.active_assignments}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {formatCurrency(member.total_exposure, summary?.currency)}
                    </TableCell>
                    <TableCell>{member.average_assignment_age_minutes}m</TableCell>
                    <TableCell>
                      <span className={member.escalation_count > 0 ? "text-red-600 font-bold" : ""}>
                        {member.escalation_count}
                      </span>
                    </TableCell>
                    <TableCell className="text-right">{member.resolved_count_30d}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  )
}
