import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { api } from "../lib/api"
import type { APIKey, FeatureFlag, FeatureFlagUpsertRequest, SettingsSummary } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import { toast } from "../components/Toast"
import { useOrgIdParam } from "../lib/org"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"

type Scope = "org" | "global"

function IconSettings() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="10" cy="10" r="2.5"/>
      <path d="M10 2v2M10 16v2M2 10h2M16 10h2M4.22 4.22l1.42 1.42M14.36 14.36l1.42 1.42M4.22 15.78l1.42-1.42M14.36 5.64l1.42-1.42"/>
    </svg>
  )
}

export default function Settings() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [settings, setSettings] = useState<SettingsSummary | null>(null)
  const [flags, setFlags] = useState<FeatureFlag[]>([])
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [newKey, setNewKey] = useState({
    name: "",
    keyType: "secret",
    scopes: "usage:write,usage:read",
    allowedIps: "",
    allowedDomains: ""
  })
  const [createdKey, setCreatedKey] = useState<APIKey | null>(null)
  const [loading, setLoading] = useState(true)
  const [savingKey, setSavingKey] = useState<string | null>(null)
  const [scope, setScope] = useState<Scope>("org")
  const [actorId, setActorId] = useState(() => localStorage.getItem("actor_id") ?? "admin")
  const [draftRollouts, setDraftRollouts] = useState<Record<string, number>>({})
  const orgId = useOrgIdParam()

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const [settingsSummary, featureFlags, apiKeysResp] = await Promise.all([
        api.settings.summary(),
        api.featureFlags.list(),
        api.apiKeys.list(),
      ])
      setSettings(settingsSummary)
      setFlags(featureFlags.flags)
      setApiKeys(apiKeysResp.keys || [])
    } catch (err) {
      toast.error(t("settings.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setLoading(false) }
  }, [t])

  useEffect(() => { void load() }, [load])
  useEffect(() => { localStorage.setItem("actor_id", actorId) }, [actorId])

  const sortedFlags = useMemo(() => [...flags].sort((a, b) => a.key.localeCompare(b.key)), [flags])

  const handleToggle = useCallback(async (flag: FeatureFlag, enabled: boolean) => {
    const rollout = draftRollouts[flag.key] ?? flag.rollout
    const payload: FeatureFlagUpsertRequest = {
      key: flag.key, enabled, rollout, actor_id: actorId,
      org_id: scope === "global" ? "" : undefined,
    }
    try {
      setSavingKey(flag.key)
      await api.featureFlags.upsert(payload)
      toast.success(t("settings.toast.flag_updated", { key: flag.key, status: enabled ? t("settings.status.enabled") : t("settings.status.disabled") }))
      void load()
    } catch (err) {
      toast.error(t("settings.toast.flag_update_failed"), err instanceof Error ? err.message : undefined)
    } finally { setSavingKey(null) }
  }, [actorId, draftRollouts, load, scope, t])

  const handleSaveRollout = useCallback(async (flag: FeatureFlag) => {
    const rollout = draftRollouts[flag.key] ?? flag.rollout
    const payload: FeatureFlagUpsertRequest = {
      key: flag.key, enabled: flag.enabled, rollout, actor_id: actorId,
      org_id: scope === "global" ? "" : undefined,
    }
    try {
      setSavingKey(flag.key)
      await api.featureFlags.upsert(payload)
      toast.success(t("settings.toast.rollout_saved", { key: flag.key }))
      void load()
    } catch (err) {
      toast.error(t("settings.toast.rollout_failed"), err instanceof Error ? err.message : undefined)
    } finally { setSavingKey(null) }
  }, [actorId, draftRollouts, load, scope, t])

  const handleRolloutChange = (key: string, value: string) => {
    const parsed = Number.parseInt(value, 10)
    if (!isNaN(parsed)) {
      setDraftRollouts((p) => ({ ...p, [key]: Math.min(100, Math.max(0, parsed)) }))
    }
  }

  const handleEditInvoiceFormat = useCallback(() => {
    if (!orgId) {
      toast.error(t("settings.toast.org_not_selected"))
      return
    }
    navigate(`/organizations/${orgId}/edit#invoice-formats`)
  }, [navigate, orgId, t])

  const handleCreateKey = useCallback(async () => {
    if (!newKey.name.trim()) {
      toast.error(t("settings.api_keys.validation.name_required"))
      return
    }
    try {
      setSavingKey("api_keys_create")
      const payload = {
        name: newKey.name.trim(),
        key_type: newKey.keyType,
        scopes: newKey.scopes.split(",").map((v) => v.trim()).filter(Boolean),
        allowed_ips: newKey.allowedIps.split(",").map((v) => v.trim()).filter(Boolean),
        allowed_domains: newKey.allowedDomains.split(",").map((v) => v.trim()).filter(Boolean)
      }
      const resp = await api.apiKeys.create(payload)
      setCreatedKey(resp)
      setNewKey({
        name: "",
        keyType: newKey.keyType,
        scopes: newKey.scopes,
        allowedIps: "",
        allowedDomains: ""
      })
      toast.success(t("settings.api_keys.toast.created"))
      void load()
    } catch (err) {
      toast.error(t("settings.api_keys.toast.create_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSavingKey(null)
    }
  }, [load, newKey, t])

  const handleRevokeKey = useCallback(async (key: APIKey) => {
    try {
      setSavingKey(key.id)
      await api.apiKeys.revoke(key.id)
      toast.success(t("settings.api_keys.toast.revoked"))
      void load()
    } catch (err) {
      toast.error(t("settings.api_keys.toast.revoke_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSavingKey(null)
    }
  }, [load, t])

  const handleCopyKey = async () => {
    if (!createdKey?.key) return
    await navigator.clipboard.writeText(createdKey.key)
    toast.success(t("settings.api_keys.toast.copied"))
  }

  const flagColumns = [
    { key: "key", label: t("settings.flags.columns.key"),
      render: (r: FeatureFlag) => (
        <div>
          <div style={{ fontWeight: 600 }}>{r.key}</div>
          <span className="muted" style={{ fontSize: "0.75rem" }}>{t("settings.flags.columns.source", { source: r.source })}</span>
        </div>
      ) },
    { key: "status", label: t("settings.flags.columns.status"), width: "100px",
      render: (r: FeatureFlag) => (
        <span className={`badge ${r.enabled ? "badge-success" : "badge-muted"}`}>
          {r.enabled ? t("settings.status.enabled") : t("settings.status.disabled")}
        </span>
      ) },
    { key: "rollout", label: t("settings.flags.columns.rollout"), width: "140px",
      render: (r: FeatureFlag) => (
        <input
          className="action-input"
          type="number" min={0} max={100}
          value={draftRollouts[r.key] ?? r.rollout}
          onChange={(e) => handleRolloutChange(r.key, e.target.value)}
          style={{ width: 80 }}
        />
      ) },
    { key: "actions", label: "", width: "160px", className: "col-actions",
      render: (r: FeatureFlag) => {
        const isSaving = savingKey === r.key
        return (
          <div style={{ display: "flex", gap: 6 }}>
            <Button size="sm" variant="secondary" disabled={isSaving}
              onClick={() => handleToggle(r, !r.enabled)}>
              {r.enabled ? t("common.disable") : t("common.enable")}
            </Button>
            <Button size="sm" variant="default" disabled={isSaving}
              onClick={() => handleSaveRollout(r)}>
              {t("common.save")}
            </Button>
          </div>
        )
      } },
  ]

  const apiKeyColumns = [
    { key: "name", label: t("settings.api_keys.columns.name"), render: (r: APIKey) => (
      <div>
        <div style={{ fontWeight: 600 }}>{r.name}</div>
        <span className="muted" style={{ fontSize: "0.75rem" }}>{r.key_type}</span>
      </div>
    ) },
    { key: "prefix", label: t("settings.api_keys.columns.prefix"), width: "140px",
      render: (r: APIKey) => <span className="cell-mono">{r.key_prefix}</span> },
    { key: "scopes", label: t("settings.api_keys.columns.scopes"),
      render: (r: APIKey) => r.scopes.length ? r.scopes.join(", ") : <span className="muted">{t("common.empty_dash")}</span> },
    { key: "status", label: t("settings.api_keys.columns.status"), width: "120px",
      render: (r: APIKey) => (
        <span className={`badge ${r.status === "active" ? "badge-success" : "badge-muted"}`}>{r.status}</span>
      ) },
    { key: "actions", label: "", width: "140px", className: "col-actions",
      render: (r: APIKey) => (
        <Button size="sm" variant="secondary" disabled={savingKey === r.id || r.status !== "active"} onClick={() => handleRevokeKey(r)}>
          {t("settings.api_keys.actions.revoke")}
        </Button>
      ) }
  ]

  return (
    <div className="page-content">
      <PageHeader icon={<IconSettings />} title={t("settings.header.title")} description={t("settings.header.description")} />

      <div className="stat-grid">
        <StatCard label={t("settings.cards.api_keys")} value={loading ? t("common.loading_dash") : String(settings?.apiKeys ?? 0)} accentColor="hsl(var(--accent-primary))" />
        <StatCard label={t("settings.cards.invoice_format")} value={loading ? t("common.loading_dash") : (settings?.invoiceFormat || t("common.empty_dash"))} accentColor="var(--accent-strong)" />
        <StatCard label={t("settings.cards.timezone")} value={loading ? t("common.loading_dash") : (settings?.timezone || t("common.empty_dash"))} accentColor="hsl(var(--status-success))" />
      </div>

      <div className="action-panel">
        <div className="action-panel-header">
          <span className="panel-title">{t("settings.invoice_format.title")}</span>
        </div>
        <div className="action-section">
          <p className="muted" style={{ margin: "0 0 12px" }}>
            {t("settings.invoice_format.description")}
          </p>
          <div className="action-buttons">
            <Button variant="default" onClick={handleEditInvoiceFormat}>
              {t("settings.invoice_format.edit")}
            </Button>
          </div>
        </div>
      </div>

      <div className="action-panel">
        <div className="action-panel-header">
          <span className="panel-title">{t("settings.api_keys.title")}</span>
        </div>
        <div className="action-section">
          <p className="muted" style={{ margin: "0 0 12px" }}>
            {t("settings.api_keys.description")}
          </p>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("settings.api_keys.fields.name")}</label>
              <Input className="action-input" value={newKey.name} onChange={(e) => setNewKey((p) => ({ ...p, name: e.target.value }))} data-testid="settings-api-key-name" />
            </div>
            <div className="action-field">
              <label className="action-label">{t("settings.api_keys.fields.type")}</label>
              <Select value={newKey.keyType} onValueChange={(value) => setNewKey((p) => ({ ...p, keyType: value }))}>
                <SelectTrigger className="action-select">
                  <SelectValue placeholder={t("settings.api_keys.fields.type_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="secret">{t("settings.api_keys.types.secret")}</SelectItem>
                  <SelectItem value="public">{t("settings.api_keys.types.public")}</SelectItem>
                  <SelectItem value="webhook">{t("settings.api_keys.types.webhook")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <label className="action-label">{t("settings.api_keys.fields.scopes")}</label>
              <Input className="action-input" value={newKey.scopes} onChange={(e) => setNewKey((p) => ({ ...p, scopes: e.target.value }))} data-testid="settings-api-key-scopes" />
            </div>
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <label className="action-label">{t("settings.api_keys.fields.allowed_ips")}</label>
              <Input className="action-input" value={newKey.allowedIps} onChange={(e) => setNewKey((p) => ({ ...p, allowedIps: e.target.value }))} />
            </div>
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <label className="action-label">{t("settings.api_keys.fields.allowed_domains")}</label>
              <Input className="action-input" value={newKey.allowedDomains} onChange={(e) => setNewKey((p) => ({ ...p, allowedDomains: e.target.value }))} />
            </div>
          </div>
          <div className="action-buttons" style={{ marginTop: 12 }}>
            <Button variant="default" disabled={savingKey === "api_keys_create"} onClick={handleCreateKey} data-testid="settings-api-key-submit">
              {t("settings.api_keys.actions.create")}
            </Button>
          </div>
          {createdKey?.key ? (
            <div className="card" style={{ marginTop: 16, padding: 16 }}>
              <div style={{ fontWeight: 600 }}>{t("settings.api_keys.created.title")}</div>
              <div className="muted" style={{ marginTop: 4 }}>{t("settings.api_keys.created.description")}</div>
              <div className="action-fields" style={{ marginTop: 12 }}>
                <div className="action-field" style={{ gridColumn: "span 2" }}>
                  <label className="action-label">{t("settings.api_keys.created.key_label")}</label>
                  <Input className="action-input" value={createdKey.key} readOnly data-testid="settings-api-key-value" />
                </div>
              </div>
              <div className="action-buttons" style={{ marginTop: 8 }}>
                <Button variant="secondary" onClick={handleCopyKey} data-testid="settings-api-key-copy">{t("settings.api_keys.created.copy")}</Button>
              </div>
            </div>
          ) : null}
        </div>
      </div>

      <DataTable
        columns={apiKeyColumns as Parameters<typeof DataTable>[0]["columns"]}
        data={apiKeys}
        loading={loading}
        emptyTitle={t("settings.api_keys.empty_title")}
        emptyDesc={t("settings.api_keys.empty_desc")}
        keyExtractor={(r) => r.id}
      />

      {/* Feature Flags */}
      <div className="action-panel">
        <div className="action-panel-header"><span className="panel-title">{t("settings.flags.title")}</span></div>
        <div className="action-section">
          <p className="muted" style={{ margin: "0 0 12px" }}>
            {t("settings.flags.description")}
          </p>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("settings.flags.scope_label")}</label>
              <Select value={scope} onValueChange={(value) => setScope(value as Scope)}>
                <SelectTrigger className="action-select">
                  <SelectValue placeholder={t("settings.flags.scope_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="org">{t("settings.flags.scope_options.org")}</SelectItem>
                  <SelectItem value="global">{t("settings.flags.scope_options.global")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field">
              <label className="action-label">{t("settings.flags.actor_label")}</label>
              <Input className="action-input" value={actorId} onChange={(e) => setActorId(e.target.value)} />
            </div>
          </div>
        </div>
      </div>

      <DataTable
        columns={flagColumns as Parameters<typeof DataTable>[0]["columns"]}
        data={sortedFlags}
        loading={loading}
        emptyTitle={t("settings.flags.empty_title")}
        emptyDesc={t("settings.flags.empty_desc")}
        keyExtractor={(r) => r.key}
      />
    </div>
  )
}
