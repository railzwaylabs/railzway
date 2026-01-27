import { useCallback, useEffect, useState } from "react"
import { Link, useParams, useNavigate } from "react-router-dom"
import {
  Clock,
  Pencil,
  Info,
  ChevronDown,
  User,
  Plus,
  Check,
  X
} from "lucide-react"
import { format, addDays, formatDistanceToNow } from "date-fns"

import { admin } from "@/api/client"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Spinner } from "@/components/ui/spinner"
import { Input } from "@/components/ui/input"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu"
import { getErrorMessage } from "@/lib/api-errors"

type TestClock = {
  id: string
  name: string
  current_time: string
  status: "active" | "advancing" | "paused"
  created_at: string
}

export default function OrgTestClockDetailPage() {
  const { orgId, clockId } = useParams()
  const navigate = useNavigate()
  const [clock, setClock] = useState<TestClock | null>(null)
  const [loading, setLoading] = useState(true)
  const [advancing, setAdvancing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Rename state
  const [isEditing, setIsEditing] = useState(false)
  const [newName, setNewName] = useState("")
  const [savingName, setSavingName] = useState(false)

  // Finish state
  const [finishing, setFinishing] = useState(false)

  const fetchClock = useCallback(async () => {
    if (!clockId) return
    try {
      const res = await admin.get(`/test-clocks/${clockId}`)
      setClock(res.data?.data)
      setNewName(res.data?.data?.name)
    } catch (err) {
      setError(getErrorMessage(err, "Failed to load test clock"))
    } finally {
      setLoading(false)
    }
  }, [clockId])

  useEffect(() => {
    fetchClock()
  }, [fetchClock])

  const handleAdvance = async (seconds: number) => {
    if (!clockId) return
    setAdvancing(true)
    setError(null)
    try {
      const res = await admin.post(`/test-clocks/${clockId}/advance`, { seconds })
      setClock(res.data?.data)
    } catch (err) {
      setError(getErrorMessage(err, "Failed to advance time"))
    } finally {
      setAdvancing(false)
    }
  }

  const handleRename = async () => {
    if (!clockId || !newName.trim()) return
    setSavingName(true)
    setError(null)
    try {
      const res = await admin.patch(`/test-clocks/${clockId}`, { name: newName })
      setClock(res.data?.data)
      setIsEditing(false)
    } catch (err) {
      setError(getErrorMessage(err, "Failed to rename test clock"))
    } finally {
      setSavingName(false)
    }
  }

  const handleFinishSimulation = async () => {
    if (!clockId) return
    if (!confirm("Are you sure you want to finish this simulation? This action cannot be undone.")) return

    setFinishing(true)
    setError(null)
    try {
      await admin.delete(`/test-clocks/${clockId}`)
      navigate(`/orgs/${orgId}/subscriptions?tab=test-clocks`)
    } catch (err) {
      setError(getErrorMessage(err, "Failed to finish simulation"))
      setFinishing(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Spinner />
      </div>
    )
  }

  if (!clock) {
    return (
      <div className="p-4">
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>Test clock not found.</AlertDescription>
        </Alert>
        <Button asChild className="mt-4" variant="outline">
          <Link to={`/orgs/${orgId}/subscriptions?tab=test-clocks`}>
            Back to list
          </Link>
        </Button>
      </div>
    )
  }

  const creationDate = new Date(clock.created_at)
  // Assuming 30 days expiration for test clocks as standard practice
  const expirationDate = addDays(creationDate, 30)
  const daysUntilExpiration = formatDistanceToNow(expirationDate)

  const formatSimulatedTime = (iso: string) => {
    // Format: January 25, 2026 at 12:11 AM GMT+7
    return format(new Date(iso), "MMMM d, yyyy 'at' h:mm a 'GMT'X")
  }

  return (
    <div className="space-y-8 max-w-5xl">
      {/* Header Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-1 text-sm text-accent-primary">
              <Link to={`/orgs/${orgId}/subscriptions?tab=test-clocks`} className="hover:underline">
                Test clocks
              </Link>
            </div>
            <div className="flex items-center gap-2 h-10">
              {isEditing ? (
                <div className="flex items-center gap-2">
                  <Input
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    className="max-w-[300px] h-9 text-lg font-bold"
                    autoFocus
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleRename()
                      if (e.key === "Escape") {
                        setIsEditing(false)
                        setNewName(clock.name)
                      }
                    }}
                  />
                  <Button size="icon-sm" onClick={handleRename} disabled={savingName}>
                    {savingName ? <Spinner className="size-4" /> : <Check className="size-4" />}
                  </Button>
                  <Button size="icon-sm" variant="ghost" onClick={() => { setIsEditing(false); setNewName(clock.name) }}>
                    <X className="size-4" />
                  </Button>
                </div>
              ) : (
                <>
                  <h1 className="text-3xl font-bold tracking-tight">{clock.name}</h1>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="text-text-muted hover:text-text-primary"
                    onClick={() => setIsEditing(true)}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                </>
              )}
            </div>
          </div>
          <Button
            variant="destructive"
            size="sm"
            onClick={handleFinishSimulation}
            disabled={finishing}
          >
            {finishing && <Spinner className="mr-2 h-4 w-4" />}
            Finish simulation
          </Button>
        </div>

        <div className="flex items-center gap-8 text-sm">
          <div className="space-y-1">
            <span className="text-text-muted block">Created on</span>
            <span className="font-medium text-text-primary underline decoration-dashed underline-offset-4">
              {format(creationDate, "MMM d, yyyy")}
            </span>
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1 text-text-muted">
              <span>Expires in</span>
              <Info className="h-3 w-3" />
            </div>
            <span className="font-medium text-text-primary">
              {daysUntilExpiration}
            </span>
          </div>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Simulation Banner */}
      <div className="rounded-md bg-[#FFF9C4] p-4 flex items-center justify-between border border-[#FBC02D]/20 text-[#F57F17]">
        <div className="flex items-center gap-3">
          <Clock className="h-5 w-5" />
          <span className="font-medium">
            The clock time is {formatSimulatedTime(clock.current_time)}
          </span>
        </div>
        <div className="flex items-center gap-4 text-sm">
          <button className="text-[#F57F17] hover:underline font-medium">
            Learn more
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="bg-white border-[#FBC02D]/30 text-text-primary hover:bg-white/90">
                {advancing ? <Spinner className="h-4 w-4 mr-2" /> : null}
                Advance time
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => handleAdvance(3600)}>
                Advance 1 hour
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleAdvance(86400)}>
                Advance 1 day
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleAdvance(86400 * 30)}>
                Advance 30 days
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Clock Objects Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">Clock objects</h2>
            <p className="text-sm text-text-muted">
              These objects are tied to the time and existence of this clock.
            </p>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                Add <ChevronDown className="ml-2 h-4 w-4 text-text-muted" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem>
                Add customer
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="py-16 flex flex-col items-center justify-center text-center">
          <div className="h-12 w-12 rounded-full bg-bg-surface-2 flex items-center justify-center mb-4">
            <User className="h-6 w-6 text-text-muted" />
          </div>
          <h3 className="text-sm font-medium text-text-muted mb-1">No clock objects</h3>
          <p className="text-sm text-text-muted mb-6">
            Create a simulated customer to get started.
          </p>
          <Button variant="outline" size="sm">
            <Plus className="mr-2 h-4 w-4" />
            Add first customer
          </Button>
        </div>
      </div>
    </div>
  )
}
