import { Link, useParams, useNavigate } from "react-router-dom"
import { Plus } from "lucide-react"
import { useState, useEffect } from "react"

import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { getErrorMessage } from "@/lib/api-errors"
import { admin } from "@/api/client"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty"

export default function OrgTestClocksList() {
  const { orgId } = useParams()
  const navigate = useNavigate()
  const [testClocks, setTestClocks] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!orgId) return
    setLoading(true)
    admin.get("/test-clocks")
      .then((res) => {
        setTestClocks(res.data?.data ?? [])
      })
      .catch((err) => {
        console.error(getErrorMessage(err, "Failed to fetch test clocks"))
      })
      .finally(() => setLoading(false))
  }, [orgId])

  if (!loading && testClocks.length === 0) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center rounded-lg border border-dashed p-8 text-center animate-in fade-in-50">
        <div className="mx-auto flex max-w-[420px] flex-col items-center justify-center text-center">
          <Empty>
            <EmptyHeader>
              <EmptyTitle>Simulate billing scenarios through time</EmptyTitle>
              <EmptyDescription>
                Simplify testing and check that your integration will work exactly as you expected by advancing time in a simulated environment.
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button asChild className="mt-4">
                <Link to={`/orgs/${orgId}/test-clocks/create`}>
                  <Plus className="mr-2 h-4 w-4" />
                  New simulation
                </Link>
              </Button>
            </EmptyContent>
          </Empty>

          <div className="mt-16 grid grid-cols-3 gap-8 text-center">
            <div className="space-y-2">
              <div className="mx-auto flex h-8 w-8 items-center justify-center rounded-full bg-accent-primary/10 text-sm font-bold text-accent-primary">1</div>
              <h4 className="text-sm font-medium">Create a simulation to set the test clock</h4>
              <p className="text-xs text-text-muted">Then set up your scenario to test.</p>
            </div>
            <div className="space-y-2">
              <div className="mx-auto flex h-8 w-8 items-center justify-center rounded-full bg-accent-primary/10 text-sm font-bold text-accent-primary">2</div>
              <h4 className="text-sm font-medium">Advance the test clock</h4>
              <p className="text-xs text-text-muted">This will simulate events through time.</p>
            </div>
            <div className="space-y-2">
              <div className="mx-auto flex h-8 w-8 items-center justify-center rounded-full bg-accent-primary/10 text-sm font-bold text-accent-primary">3</div>
              <h4 className="text-sm font-medium">Review test objects and events in Dashboard</h4>
              <p className="text-xs text-text-muted">You can make changes and advance the clock again.</p>
            </div>
          </div>
        </div>
      </div>
    )
  }

  const formatTime = (iso: string) => {
    return new Date(iso).toLocaleString(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    })
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-medium">Test Clocks</h2>
        <Button size="sm" asChild>
          <Link to={`/orgs/${orgId}/test-clocks/create`}>
            <Plus className="mr-2 h-4 w-4" />
            New simulation
          </Link>
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Simulated Time</TableHead>
              <TableHead>Created At</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {testClocks.map((clock) => (
              <TableRow
                key={clock.id}
                className="cursor-pointer hover:bg-bg-surface-2"
                onClick={() => navigate(`/orgs/${orgId}/test-clocks/${clock.id}`)}
              >
                <TableCell className="font-medium">{clock.name}</TableCell>
                <TableCell>
                  <Badge variant="outline">{clock.status}</Badge>
                </TableCell>
                <TableCell>{formatTime(clock.current_time)}</TableCell>
                <TableCell>{formatTime(clock.created_at)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
