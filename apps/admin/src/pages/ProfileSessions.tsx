import { useCallback, useEffect, useMemo, useState } from "react"
import PageHeader from "../components/PageHeader"
import DataTable from "../components/DataTable"
import { Button } from "../components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../components/ui/dialog"
import { toast } from "../components/Toast"
import { api } from "../lib/api"
import { clearOrgId } from "../lib/auth"
import { formatDateTime } from "../lib/display"
import type { AdminSession } from "../lib/types"

function IconSessions() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7">
      <rect x="3" y="4" width="18" height="12" rx="2" />
      <path d="M8 20h8M12 16v4" strokeLinecap="round" />
    </svg>
  )
}

function statusForSession(session: AdminSession): { label: string; className: string } {
  if (session.revokedAt) return { label: "Revoked", className: "badge-neutral" }
  const expiresAt = new Date(session.expiresAt)
  if (!Number.isNaN(expiresAt.getTime()) && expiresAt.getTime() <= Date.now()) {
    return { label: "Expired", className: "badge-muted" }
  }
  if (session.current) return { label: "Current", className: "badge-success" }
  return { label: "Active", className: "badge-info" }
}

function describeDevice(userAgent?: string): string {
  const source = (userAgent || "").toLowerCase()
  if (!source) return "Unknown device"

  let browser = "Browser"
  if (source.includes("edg/")) browser = "Edge"
  else if (source.includes("chrome/") && !source.includes("edg/")) browser = "Chrome"
  else if (source.includes("safari/") && !source.includes("chrome/")) browser = "Safari"
  else if (source.includes("firefox/")) browser = "Firefox"

  let platform = "device"
  if (source.includes("mac os x")) platform = "macOS"
  else if (source.includes("windows")) platform = "Windows"
  else if (source.includes("iphone") || source.includes("ipad")) platform = "iOS"
  else if (source.includes("android")) platform = "Android"
  else if (source.includes("linux")) platform = "Linux"

  return `${browser} on ${platform}`
}

export default function ProfileSessions() {
  const [sessions, setSessions] = useState<AdminSession[]>([])
  const [loading, setLoading] = useState(true)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [revokingOthers, setRevokingOthers] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmAction, setConfirmAction] = useState<{ type: "session"; session: AdminSession } | { type: "others" } | null>(null)

  const loadSessions = useCallback(async () => {
    try {
      setLoading(true)
      const resp = await api.auth.listSessions()
      setSessions(resp.sessions || [])
    } catch (err) {
      toast.error("Failed to load sessions", err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadSessions()
  }, [loadSessions])

  const handleRevoke = useCallback((session: AdminSession) => {
    setConfirmAction({ type: "session", session })
    setConfirmOpen(true)
  }, [])

  const handleRevokeOthers = useCallback(() => {
    setConfirmAction({ type: "others" })
    setConfirmOpen(true)
  }, [])

  const confirmRevoke = useCallback(async () => {
    if (!confirmAction) return

    if (confirmAction.type === "session") {
      const { session } = confirmAction
      try {
        setBusyId(session.id)
        const resp = await api.auth.revokeSession(session.id)
        setConfirmOpen(false)
        setConfirmAction(null)
        if (resp.revokedCurrent) {
          clearOrgId()
          window.location.assign("/")
          return
        }
        toast.success("Session revoked")
        await loadSessions()
      } catch (err) {
        toast.error("Failed to revoke session", err instanceof Error ? err.message : undefined)
      } finally {
        setBusyId(null)
      }
      return
    }

    try {
      setRevokingOthers(true)
      const resp = await api.auth.revokeOtherSessions()
      setConfirmOpen(false)
      setConfirmAction(null)
      toast.success(`Revoked ${resp.revokedCount ?? 0} other session(s)`)
      await loadSessions()
    } catch (err) {
      toast.error("Failed to revoke other sessions", err instanceof Error ? err.message : undefined)
    } finally {
      setRevokingOthers(false)
    }
  }, [confirmAction, loadSessions])

  const activeCount = useMemo(
    () => sessions.filter((session) => !session.revokedAt).length,
    [sessions]
  )

  const columns = useMemo(
    () => [
      {
        key: "device",
        label: "Device",
        render: (session: AdminSession) => (
          <div>
            <div style={{ fontWeight: 600, display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
              <span>{describeDevice(session.userAgent)}</span>
              {session.current ? <span className="badge badge-success">Current</span> : null}
            </div>
            <div className="muted" style={{ fontSize: "12px", marginTop: 4 }}>
              {session.userAgent || "User agent unavailable"}
            </div>
          </div>
        ),
      },
      {
        key: "ipAddress",
        label: "IP",
        width: "140px",
        render: (session: AdminSession) => <span className="cell-mono">{session.ipAddress || "—"}</span>,
      },
      {
        key: "status",
        label: "Status",
        width: "110px",
        render: (session: AdminSession) => {
          const status = statusForSession(session)
          return <span className={`badge ${status.className}`}>{status.label}</span>
        },
      },
      {
        key: "lastSeenAt",
        label: "Seen",
        width: "220px",
        render: (session: AdminSession) => (
          <div>
            <div>{formatDateTime(session.lastSeenAt)}</div>
            <div className="muted" style={{ fontSize: "12px", marginTop: 4 }}>
              Created {formatDateTime(session.createdAt)}
            </div>
          </div>
        ),
      },
      {
        key: "expiresAt",
        label: "Expires",
        width: "180px",
        render: (session: AdminSession) => <span>{formatDateTime(session.expiresAt)}</span>,
      },
      {
        key: "actions",
        label: "",
        width: "110px",
        className: "col-actions",
        render: (session: AdminSession) => (
          !session.revokedAt ? (
            <Button
              size="sm"
              variant={session.current ? "destructive" : "ghost"}
              disabled={busyId === session.id}
              onClick={() => void handleRevoke(session)}
            >
              {session.current ? "Log out" : "Revoke"}
            </Button>
          ) : null
        ),
      },
    ],
    [busyId, handleRevoke]
  )

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconSessions />}
        title="My Sessions"
        description={`Review active admin sessions, device details, and IP addresses. ${activeCount} session(s) currently active.`}
        actions={
          <Button variant="outline" onClick={() => void handleRevokeOthers()} disabled={revokingOthers}>
            {revokingOthers ? "Revoking..." : "Revoke Other Sessions"}
          </Button>
        }
      />

      <DataTable
        columns={columns as any}
        data={sessions}
        loading={loading}
        emptyTitle="No sessions"
        emptyDesc="Your admin sessions will appear here after login."
      />

      <Dialog
        open={confirmOpen}
        onOpenChange={(open) => {
          setConfirmOpen(open)
          if (!open) setConfirmAction(null)
        }}
      >
        <DialogContent style={{ maxWidth: 480 }}>
          <DialogHeader>
            <DialogTitle>
              {confirmAction?.type === "others" ? "Revoke other sessions" : confirmAction?.type === "session" && confirmAction.session.current ? "Log out current session" : "Revoke session"}
            </DialogTitle>
            <DialogDescription>
              {confirmAction?.type === "others"
                ? "This will sign out every other admin session for your account and keep this current session active."
                : confirmAction?.type === "session" && confirmAction.session.current
                  ? "This will sign out your current admin session immediately."
                  : "This will revoke the selected admin session immediately."}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter style={{ marginTop: 24 }}>
            <Button variant="secondary" onClick={() => setConfirmOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => void confirmRevoke()}
              disabled={Boolean(busyId) || revokingOthers}
            >
              {confirmAction?.type === "others" ? "Revoke Other Sessions" : confirmAction?.type === "session" && confirmAction.session.current ? "Log out" : "Revoke Session"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
