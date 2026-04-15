import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { formatDate } from "../lib/display"
import { Button } from "../components/ui/button"
import { api } from "../lib/api"
import type { APIKey } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogFooter, DialogTitle, DialogTrigger } from "../components/ui/dialog"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"

function IconKey() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4" />
    </svg>
  )
}

function IconCopy() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  )
}

export default function ApiKeys() {
  const { t } = useTranslation()
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Create state
  const [createOpen, setCreateOpen] = useState(false)
  const [createForm, setCreateForm] = useState({ name: "" })
  const [creating, setCreating] = useState(false)
  const [newKey, setNewKey] = useState<string | null>(null)
  const [revokeTarget, setRevokeTarget] = useState<APIKey | null>(null)
  const [revoking, setRevoking] = useState(false)

  const loadKeys = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await api.apiKeys.list()
      setKeys(data.keys || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load")
      toast.error(t("settings.api_keys.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => { void loadKeys() }, [loadKeys])

  const handleCreate = async () => {
    if (!createForm.name.trim()) return
    try {
      setCreating(true)
      const resp = await api.apiKeys.create({
        name: createForm.name,
        key_type: "secret",
        scopes: ["*"]
      })
      setNewKey(resp.key || "Error: key not returned")
      await loadKeys()
    } catch (err) {
      toast.error(t("settings.api_keys.toast.create_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setCreating(false)
    }
  }

  const handleRevoke = async (id: string) => {
    try {
      setRevoking(true)
      await api.apiKeys.revoke(id)
      setRevokeTarget(null)
      toast.success(t("settings.api_keys.toast.revoked"))
      await loadKeys()
    } catch (err) {
      toast.error(t("settings.api_keys.toast.revoke_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setRevoking(false)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success(t("settings.api_keys.toast.copied"))
  }

  const columns = useMemo(() => [
    {
      key: "name", label: t("settings.api_keys.columns.name"),
      render: (row: APIKey) => (
        <div>
          <div style={{ fontWeight: 600 }}>{row.name}</div>
          <div className="muted" style={{ fontSize: "11px" }}>{row.id}</div>
        </div>
      ),
    },
    {
      key: "prefix", label: t("settings.api_keys.columns.prefix"), width: "140px",
      render: (row: APIKey) => <code className="cell-mono" style={{ fontSize: "12px" }}>{row.key_prefix}...</code>
    },
    {
      key: "status", label: t("settings.api_keys.columns.status"), width: "100px",
      render: (row: APIKey) => (
        <span className={`badge ${row.status === 'active' ? 'badge-success' : 'badge-neutral'}`}>
          {row.status}
        </span>
      )
    },
    {
      key: "created_at", label: t("common.created") || "Created", width: "130px",
      render: (row: APIKey) => <span className="muted">{formatDate(row.created_at)}</span>
    },
    {
      key: "actions", label: "", width: "80px", className: "col-actions",
      render: (row: APIKey) => (
        row.status === 'active' ? (
          <Button variant="ghost" size="sm" onClick={() => setRevokeTarget(row)} style={{ color: "var(--status-danger)" }}>
            {t("settings.api_keys.actions.revoke")}
          </Button>
        ) : null
      ),
    },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconKey />}
        title={t("settings.api_keys.title")}
        description={t("settings.api_keys.description")}
        actions={
          <Dialog open={createOpen} onOpenChange={(open) => { setCreateOpen(open); if (!open) { setNewKey(null); setCreateForm({ name: "" }) } }}>
            <DialogTrigger asChild>
              <Button style={{ display: "flex", alignItems: "center", gap: 6 }}>
                + {t("settings.api_keys.actions.create")}
              </Button>
            </DialogTrigger>
            <DialogContent style={{ maxWidth: 480 }}>
              <DialogHeader>
                <DialogTitle>{t("settings.api_keys.actions.create")}</DialogTitle>
                <DialogDescription>
                  {t("settings.api_keys.created.description")}
                </DialogDescription>
              </DialogHeader>

              {newKey ? (
                <div style={{ display: "flex", flexDirection: "column", gap: "16px", marginTop: "16px" }}>
                  <div style={{ padding: "16px", background: "var(--bg-subtle)", borderRadius: "8px", border: "1px solid var(--border-color)", position: "relative" }}>
                    <code style={{ display: "block", padding: "8px", paddingTop: "0", fontSize: "14px", wordBreak: "break-all", fontFamily: "monospace", color: "var(--text-main)" }}>
                      {newKey}
                    </code>
                    <div style={{ display: "flex", justifyContent: "flex-end", marginTop: "12px" }}>
                       <Button variant="secondary" size="sm" onClick={() => copyToClipboard(newKey)} style={{ gap: "6px" }}>
                         <IconCopy /> {t("settings.api_keys.created.copy")}
                       </Button>
                    </div>
                  </div>
                  <div style={{ padding: "12px", background: "rgba(234, 179, 8, 0.1)", border: "1px solid rgba(234, 179, 8, 0.3)", borderRadius: "6px", fontSize: "13px", color: "var(--status-warning)" }}>
                    {t("settings.api_keys.created.description")}
                  </div>
                  <Button onClick={() => setCreateOpen(false)}>{t("common.done") || "Done"}</Button>
                </div>
              ) : (
                <div style={{ display: "flex", flexDirection: "column", gap: "20px", marginTop: "20px" }}>
                  <div className="action-field" style={{ display: "flex", flexDirection: "column", gap: "8px" }}>
                    <Label htmlFor="name">{t("settings.api_keys.fields.name")}</Label>
                    <Input
                      id="name"
                      autoFocus
                      value={createForm.name}
                      onChange={(e) => setCreateForm({ name: e.target.value })}
                      placeholder={t("settings.api_keys.placeholders.name")}
                    />
                  </div>
                  <DialogFooter style={{ marginTop: "8px" }}>
                    <Button variant="secondary" onClick={() => setCreateOpen(false)}>{t("common.cancel")}</Button>
                    <Button onClick={handleCreate} disabled={creating || !createForm.name.trim()}>
                      {creating ? t("common.creating") : t("settings.api_keys.actions.create")}
                    </Button>
                  </DialogFooter>
                </div>
              )}
            </DialogContent>
          </Dialog>
        }
      />

      <DataTable
        columns={columns as any}
        data={keys}
        loading={loading}
        emptyTitle={t("settings.api_keys.empty_title")}
        emptyDesc={t("settings.api_keys.empty_desc")}
      />

      <Dialog open={Boolean(revokeTarget)} onOpenChange={(open) => { if (!open) setRevokeTarget(null) }}>
        <DialogContent style={{ maxWidth: 480 }}>
          <DialogHeader>
            <DialogTitle>{t("settings.api_keys.actions.revoke")}</DialogTitle>
            <DialogDescription>
              {revokeTarget
                ? `Revoke API key "${revokeTarget.name}"? This action cannot be undone.`
                : t("common.confirm_action")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter style={{ marginTop: "8px" }}>
            <Button variant="secondary" onClick={() => setRevokeTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => revokeTarget ? void handleRevoke(revokeTarget.id) : undefined}
              disabled={revoking}
            >
              {revoking ? (t("common.loading") || "Revoking...") : t("settings.api_keys.actions.revoke")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
