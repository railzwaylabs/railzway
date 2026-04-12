import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { api } from "../lib/api"
import type { AppDefinition, AppInstallation } from "../lib/types"
import PageHeader from "../components/PageHeader"
import DataTable from "../components/DataTable"
import { Button } from "../components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from "../components/ui/dialog"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { Input } from "../components/ui/input"
import { toast } from "../components/Toast"
import { formatDate } from "../lib/display"

function IconApps() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="3" y="3" width="6" height="6" rx="1.5"/>
      <rect x="11" y="3" width="6" height="6" rx="1.5"/>
      <rect x="3" y="11" width="6" height="6" rx="1.5"/>
      <rect x="11" y="11" width="6" height="6" rx="1.5"/>
    </svg>
  )
}

export default function Apps() {
  const { t } = useTranslation()
  const [catalog, setCatalog] = useState<AppDefinition[]>([])
  const [installations, setInstallations] = useState<AppInstallation[]>([])
  const [loading, setLoading] = useState(true)
  const [mutating, setMutating] = useState(false)
  const [authSelection, setAuthSelection] = useState<Record<string, string>>({})
  const [selectedAppId, setSelectedAppId] = useState<string | null>(null)
  const [selectedInstallationId, setSelectedInstallationId] = useState<string | null>(null)
  const [formAuthMethod, setFormAuthMethod] = useState("")
  const [credentials, setCredentials] = useState<Record<string, string>>({})
  const [connecting, setConnecting] = useState(false)

  const loadAll = useCallback(async () => {
    try {
      setLoading(true)
      const [catalogResp, installResp] = await Promise.all([
        api.apps.catalog(),
        api.apps.installations()
      ])
      setCatalog(catalogResp.apps || [])
      setInstallations(installResp.installations || [])
    } catch (err) {
      toast.error(t("apps.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void loadAll() }, [loadAll])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const status = params.get("status")
    if (status === "stripe_connected") {
      toast.success(t("apps.toast_stripe_connected"), t("apps.toast_stripe_connected_desc"))
      params.delete("status")
      const next = params.toString()
      const url = next ? `${window.location.pathname}?${next}` : window.location.pathname
      window.history.replaceState({}, "", url)
    }
  }, [])

  const installedByApp = useMemo(() => {
    const map = new Map<string, AppInstallation>()
    for (const inst of installations) {
      map.set(inst.app_id, inst)
    }
    return map
  }, [installations])

  const catalogById = useMemo(() => {
    const map = new Map<string, AppDefinition>()
    for (const app of catalog) {
      map.set(app.id, app)
    }
    return map
  }, [catalog])

  const selectedInstallation = useMemo(
    () => installations.find((inst) => inst.id === selectedInstallationId) ?? null,
    [installations, selectedInstallationId]
  )
  const selectedCatalog = useMemo(
    () => {
      if (selectedInstallation) {
        return catalogById.get(selectedInstallation.app_id) ?? null
      }
      if (selectedAppId) {
        return catalogById.get(selectedAppId) ?? null
      }
      return null
    },
    [catalogById, selectedAppId, selectedInstallation]
  )

  const requiredCredentialKeys = useMemo(
    () => selectedCatalog?.credentials_schema?.[formAuthMethod] ?? [],
    [formAuthMethod, selectedCatalog]
  )
  const missingRequiredKeys = useMemo(
    () => requiredCredentialKeys.filter((key) => !credentials[key]?.trim()),
    [credentials, requiredCredentialKeys]
  )

  const resolveAuthMethod = useCallback((app: AppDefinition) => {
    const methods = app.auth_methods ?? []
    if (methods.length === 0) return "api_keys"
    return authSelection[app.id] ?? methods[0]
  }, [authSelection])

  useEffect(() => {
    if (!selectedInstallation && !selectedAppId) {
      setFormAuthMethod("")
      setCredentials({})
      return
    }
    const methods = selectedCatalog?.auth_methods ?? []
    const initialMethod = selectedInstallation
      ? (selectedInstallation.auth_method || methods[0] || "api_keys")
      : (methods[0] || "api_keys")
    setFormAuthMethod(initialMethod)
    const required = selectedCatalog?.credentials_schema?.[initialMethod] ?? []
    const next: Record<string, string> = {}
    required.forEach((key) => { next[key] = "" })
    setCredentials(next)
  }, [selectedAppId, selectedInstallation, selectedCatalog])

  useEffect(() => {
    if (!selectedCatalog) return
    const required = selectedCatalog.credentials_schema?.[formAuthMethod] ?? []
    setCredentials((prev) => {
      const next: Record<string, string> = {}
      required.forEach((key) => {
        next[key] = prev[key] ?? ""
      })
      return next
    })
  }, [formAuthMethod, selectedCatalog])

  const formatCredentialLabel = useCallback((key: string) => {
    const map: Record<string, string> = {
      publishable_key: t("apps.dialog.publishable_key"),
      secret_key: t("apps.dialog.secret_key"),
      webhook_secret: t("apps.dialog.webhook_secret"),
      api_key: t("apps.dialog.api_key"),
      username: t("apps.dialog.username"),
      password: t("apps.dialog.password"),
      host: t("apps.dialog.host"),
      port: t("apps.dialog.port")
    }
    return map[key] ?? key.replace(/_/g, " ")
  }, [t])

  const formatCredentialPlaceholder = useCallback((key: string) => {
    const map: Record<string, string> = {
      publishable_key: t("apps.dialog.publishable_placeholder"),
      secret_key: t("apps.dialog.secret_placeholder"),
      webhook_secret: t("apps.dialog.webhook_placeholder"),
      api_key: t("apps.dialog.api_key_placeholder"),
      username: t("apps.dialog.username_placeholder"),
      password: t("apps.dialog.password_placeholder"),
      host: t("apps.dialog.host_placeholder"),
      port: t("apps.dialog.port_placeholder")
    }
    return map[key] ?? ""
  }, [t])

  const resolveCredentialInputType = useCallback((key: string) => {
    if (key === "secret_key" || key === "webhook_secret" || key === "api_key" || key === "password") {
      return "password"
    }
    return "text"
  }, [])

  const handleInstall = useCallback((app: AppDefinition) => {
    setSelectedInstallationId(null)
    setSelectedAppId(app.id)
  }, [])

  const handleToggle = useCallback(async (inst: AppInstallation) => {
    try {
      setMutating(true)
      const next = inst.status === "active" ? "disabled" : "active"
      await api.apps.updateInstallation(inst.id, { status: next })
      toast.success(t("apps.toast_installation_updated"), `${inst.app_id} → ${next}`)
      await loadAll()
    } catch (err) {
      toast.error(t("apps.toast_update_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setMutating(false)
    }
  }, [loadAll])


  const handleSaveConfig = useCallback(async () => {
    const appId = selectedInstallation?.app_id ?? selectedAppId
    if (!appId) return
    try {
      setMutating(true)
      if (appId === "payment.stripe" && formAuthMethod === "oauth2" && !selectedInstallation) {
        toast.error(t("apps.validation.oauth_required"))
        return
      }
      const payload: { auth_method?: string; credentials?: Record<string, unknown> } = {}
      if (formAuthMethod) {
        payload.auth_method = formAuthMethod
      }
      const required = requiredCredentialKeys
      if (required.length > 0) {
        const creds: Record<string, unknown> = {}
        required.forEach((key) => {
          const value = credentials[key]?.trim()
          if (value) {
            creds[key] = value
          }
        })
        if (missingRequiredKeys.length > 0) {
          toast.error(t("apps.validation.credentials_required"))
          return
        }
        payload.credentials = creds
      }
      if (selectedInstallation) {
        await api.apps.updateInstallation(selectedInstallation.id, payload)
        toast.success(t("apps.toast_config_saved"), selectedInstallation.app_id)
      } else {
        await api.apps.install({ app_id: appId, auth_method: formAuthMethod, credentials: payload.credentials })
        toast.success(t("apps.toast_installed"), `${appId} (${formAuthMethod})`)
      }
      await loadAll()
      setSelectedInstallationId(null)
      setSelectedAppId(null)
    } catch (err) {
      toast.error(t("apps.toast_config_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setMutating(false)
    }
  }, [formAuthMethod, loadAll, selectedAppId, selectedInstallation, t, credentials, selectedCatalog, requiredCredentialKeys, missingRequiredKeys])

  const handleStripeConnect = useCallback(async () => {
    try {
      if (!selectedInstallation) return
      setConnecting(true)
      if (formAuthMethod !== "oauth2") {
        await api.apps.updateInstallation(selectedInstallation.id, { auth_method: "oauth2" })
        setFormAuthMethod("oauth2")
      }
      const resp = await api.apps.startStripeOAuth()
      if (resp?.url) {
        window.location.href = resp.url
      }
    } catch (err) {
      toast.error(t("apps.toast_stripe_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setConnecting(false)
    }
  }, [formAuthMethod, selectedInstallation])

  const catalogColumns = useMemo(() => [
    { key: "name", label: t("apps.catalog.columns.app"), render: (r: AppDefinition) => (
      <div>
        <div style={{ fontWeight: 600 }}>{r.name}</div>
        <div className="muted">{r.description}</div>
      </div>
    ) },
    { key: "category", label: t("apps.catalog.columns.category"), width: "120px", render: (r: AppDefinition) => (
      <span className="badge badge-info">{r.category}</span>
    ) },
    { key: "provider", label: t("apps.catalog.columns.provider"), width: "140px", render: (r: AppDefinition) => <span className="cell-mono">{r.provider}</span> },
    { key: "auth", label: t("apps.catalog.columns.auth"), width: "160px", render: (r: AppDefinition) => {
      const methods = r.auth_methods ?? []
      if (methods.length <= 1) {
        const label = (methods[0] ?? "api_keys").replace(/_/g, " ")
        return <span className="badge badge-muted">{label}</span>
      }
      const value = resolveAuthMethod(r)
      return (
        <Select
          value={value}
          onValueChange={(val) => setAuthSelection((prev) => ({ ...prev, [r.id]: val }))}
        >
          <SelectTrigger className="h-8">
            <SelectValue placeholder={t("apps.catalog.select_auth")} />
          </SelectTrigger>
          <SelectContent>
            {methods.map((method) => (
              <SelectItem key={method} value={method}>
                {method.replace(/_/g, " ")}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    } },
    { key: "status", label: t("apps.catalog.columns.status"), width: "120px", render: (r: AppDefinition) => (
      <span className={`badge ${r.status === "active" ? "badge-success" : "badge-muted"}`}>{t(`apps.status.${r.status}`, { defaultValue: r.status })}</span>
    ) },
    { key: "actions", label: "", width: "140px", className: "col-actions", render: (r: AppDefinition) => {
      const installed = installedByApp.get(r.id)
      return (
        <Button
          variant={installed ? "secondary" : "default"}
          size="sm"
          disabled={mutating}
          onClick={() => (installed ? handleToggle(installed) : handleInstall(r))}
        >
          {installed
            ? (installed.status === "active" ? t("common.disable") : t("common.enable"))
            : t("apps.catalog.install")}
        </Button>
      )
    } },
  ], [authSelection, handleInstall, handleToggle, installedByApp, mutating, resolveAuthMethod, t])

  const installColumns = useMemo(() => [
    { key: "app_id", label: t("apps.installed.columns.app"), render: (r: AppInstallation) => (
      <div>
        <div style={{ fontWeight: 600 }}>{r.app_id}</div>
        <div className="muted">{r.status} · {r.auth_method || "api_keys"}</div>
      </div>
    ) },
    { key: "status", label: t("apps.installed.columns.status"), width: "120px", render: (r: AppInstallation) => (
      <span className={`badge ${r.status === "active" ? "badge-success" : "badge-muted"}`}>{t(`apps.status.${r.status}`, { defaultValue: r.status })}</span>
    ) },
    { key: "updated_at", label: t("apps.installed.columns.updated"), width: "140px", render: (r: AppInstallation) => <span className="muted">{formatDate(r.updated_at)}</span> },
    { key: "actions", label: "", width: "180px", className: "col-actions", render: (r: AppInstallation) => (
      <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
        <Button variant="secondary" size="sm" disabled={mutating} onClick={() => handleToggle(r)}>
          {r.status === "active" ? t("common.disable") : t("common.enable")}
        </Button>
        <Button variant="default" size="sm" onClick={() => setSelectedInstallationId(r.id)}>
          {t("apps.installed.configure")}
        </Button>
      </div>
    ) },
  ], [handleToggle, mutating, t])

  return (
    <div className="page-content">
      <PageHeader icon={<IconApps />} title={t("apps.catalog.title")} description={t("apps.catalog.description")} />

      <DataTable
        columns={catalogColumns as Parameters<typeof DataTable>[0]["columns"]}
        data={catalog}
        loading={loading}
        emptyTitle={t("apps.catalog.empty_title")}
        emptyDesc={t("apps.catalog.empty_desc")}
      />

      <div style={{ height: 24 }} />

      <PageHeader title={t("apps.installed.title")} description={t("apps.installed.description")} />
      <DataTable
        columns={installColumns as Parameters<typeof DataTable>[0]["columns"]}
        data={installations}
        loading={loading}
        emptyTitle={t("apps.installed.empty_title")}
        emptyDesc={t("apps.installed.empty_desc")}
      />

      <Dialog
        open={Boolean(selectedInstallation || selectedAppId)}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedInstallationId(null)
            setSelectedAppId(null)
          }
        }}
      >
        {(selectedInstallation || selectedAppId) && (
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("apps.dialog.title", { app: selectedCatalog?.name ?? selectedInstallation?.app_id ?? selectedAppId })}</DialogTitle>
              <DialogDescription>{t("apps.dialog.description")}</DialogDescription>
            </DialogHeader>
            <div style={{ display: "grid", gap: 16, marginTop: 16 }}>
              <div className="action-field">
                <label className="action-label">{t("apps.dialog.auth_method")}</label>
                <Select value={formAuthMethod} onValueChange={(value) => setFormAuthMethod(value)}>
                  <SelectTrigger className="action-select">
                    <SelectValue placeholder={t("apps.dialog.select_auth_method")} />
                  </SelectTrigger>
                  <SelectContent>
                    {(selectedCatalog?.auth_methods ?? [selectedInstallation?.auth_method ?? "api_keys"]).map((method) => (
                      <SelectItem key={method} value={method}>
                        {method.replace(/_/g, " ")}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {selectedCatalog?.id === "payment.stripe" && formAuthMethod === "oauth2" && (
                <div className="card" style={{ padding: 16, background: "var(--surface-subtle)" }}>
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <div>
                      <div style={{ fontWeight: 600 }}>{t("apps.dialog.stripe.title")}</div>
                      <div className="muted">{t("apps.dialog.stripe.description")}</div>
                    </div>
                    <Button disabled={connecting} onClick={handleStripeConnect}>
                      {connecting ? t("apps.dialog.stripe.redirecting") : t("apps.dialog.stripe.connect")}
                    </Button>
                  </div>
                </div>
              )}

              {requiredCredentialKeys.length > 0 && (
                <div className="action-fields">
                  {requiredCredentialKeys.map((key) => (
                    <div className="action-field">
                      <label className="action-label">
                        {formatCredentialLabel(key)} <span className="req">*</span>
                      </label>
                      <Input
                        className="action-input"
                        type={resolveCredentialInputType(key)}
                        placeholder={formatCredentialPlaceholder(key)}
                        value={credentials[key] ?? ""}
                        onChange={(event) => setCredentials((prev) => ({ ...prev, [key]: event.target.value }))}
                      />
                    </div>
                  ))}
                </div>
              )}

              {selectedCatalog?.id === "payment.stripe" && formAuthMethod === "oauth2" && (
                <div className="muted" style={{ fontSize: 12 }}>
                  {t("apps.dialog.stripe.oauth_hint")}
                </div>
              )}
              <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
                <DialogClose asChild>
                  <Button variant="secondary">{t("common.cancel")}</Button>
                </DialogClose>
                <Button
                  disabled={mutating || (selectedCatalog?.id === "payment.stripe" && formAuthMethod === "oauth2") || missingRequiredKeys.length > 0}
                  onClick={handleSaveConfig}
                >
                  {t("apps.dialog.save")}
                </Button>
              </div>
            </div>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}
