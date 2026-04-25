import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { api } from "../lib/api"
import type { FeatureFlag, FeatureFlagUpsertRequest } from "../lib/types"

type Scope = "org" | "global"

function IconFeatureFlags() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M3 6h8" strokeLinecap="round" />
      <path d="M3 14h14" strokeLinecap="round" />
      <circle cx="13" cy="6" r="2.5" />
      <circle cx="8" cy="14" r="2.5" />
    </svg>
  )
}

export default function FeatureFlags() {
  const { t } = useTranslation()
  const [flags, setFlags] = useState<FeatureFlag[]>([])
  const [loading, setLoading] = useState(true)
  const [savingKey, setSavingKey] = useState<string | null>(null)
  const [scope, setScope] = useState<Scope>("org")
  const [actorId, setActorId] = useState(() => localStorage.getItem("actor_id") ?? "admin")
  const [draftRollouts, setDraftRollouts] = useState<Record<string, number>>({})

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const featureFlags = await api.featureFlags.list()
      setFlags(featureFlags.flags)
    } catch (err) {
      toast.error(t("settings.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    localStorage.setItem("actor_id", actorId)
  }, [actorId])

  const sortedFlags = useMemo(() => [...flags].sort((a, b) => a.key.localeCompare(b.key)), [flags])

  const buildPayload = useCallback(
    (flag: FeatureFlag, enabled: boolean, rollout: number): FeatureFlagUpsertRequest => ({
      key: flag.key,
      enabled,
      rollout,
      actor_id: actorId,
      org_id: scope === "global" ? "" : undefined,
    }),
    [actorId, scope],
  )

  const handleToggle = useCallback(async (flag: FeatureFlag, enabled: boolean) => {
    const rollout = draftRollouts[flag.key] ?? flag.rollout
    try {
      setSavingKey(flag.key)
      await api.featureFlags.upsert(buildPayload(flag, enabled, rollout))
      toast.success(
        t("settings.toast.flag_updated", {
          key: flag.key,
          status: enabled ? t("settings.status.enabled") : t("settings.status.disabled"),
        }),
      )
      void load()
    } catch (err) {
      toast.error(t("settings.toast.flag_update_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSavingKey(null)
    }
  }, [buildPayload, draftRollouts, load, t])

  const handleSaveRollout = useCallback(async (flag: FeatureFlag) => {
    const rollout = draftRollouts[flag.key] ?? flag.rollout
    try {
      setSavingKey(flag.key)
      await api.featureFlags.upsert(buildPayload(flag, flag.enabled, rollout))
      toast.success(t("settings.toast.rollout_saved", { key: flag.key }))
      void load()
    } catch (err) {
      toast.error(t("settings.toast.rollout_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSavingKey(null)
    }
  }, [buildPayload, draftRollouts, load, t])

  const handleRolloutChange = useCallback((key: string, value: string) => {
    const parsed = Number.parseInt(value, 10)
    if (!Number.isNaN(parsed)) {
      setDraftRollouts((current) => ({
        ...current,
        [key]: Math.min(100, Math.max(0, parsed)),
      }))
    }
  }, [])

  const columns = [
    {
      key: "key",
      label: t("settings.flags.columns.key"),
      render: (row: FeatureFlag) => (
        <div>
          <div style={{ fontWeight: 600 }}>{row.key}</div>
          <span className="muted" style={{ fontSize: "0.75rem" }}>
            {t("settings.flags.columns.source", { source: row.source })}
          </span>
        </div>
      ),
    },
    {
      key: "status",
      label: t("settings.flags.columns.status"),
      width: "100px",
      render: (row: FeatureFlag) => (
        <span className={`badge ${row.enabled ? "badge-success" : "badge-muted"}`}>
          {row.enabled ? t("settings.status.enabled") : t("settings.status.disabled")}
        </span>
      ),
    },
    {
      key: "rollout",
      label: t("settings.flags.columns.rollout"),
      width: "140px",
      render: (row: FeatureFlag) => (
        <input
          className="action-input"
          type="number"
          min={0}
          max={100}
          value={draftRollouts[row.key] ?? row.rollout}
          onChange={(event) => handleRolloutChange(row.key, event.target.value)}
          style={{ width: 80 }}
        />
      ),
    },
    {
      key: "actions",
      label: "",
      width: "160px",
      className: "col-actions",
      render: (row: FeatureFlag) => {
        const isSaving = savingKey === row.key
        return (
          <div style={{ display: "flex", gap: 6 }}>
            <Button size="sm" variant="secondary" disabled={isSaving} onClick={() => handleToggle(row, !row.enabled)}>
              {row.enabled ? t("common.disable") : t("common.enable")}
            </Button>
            <Button size="sm" variant="default" disabled={isSaving} onClick={() => handleSaveRollout(row)}>
              {t("common.save")}
            </Button>
          </div>
        )
      },
    },
  ]

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconFeatureFlags />}
        title={t("routes.feature_flags.label")}
        description={t("routes.feature_flags.desc")}
      />

      <div className="action-panel">
        <div className="action-panel-header">
          <span className="panel-title">{t("settings.flags.title")}</span>
        </div>
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
              <Input className="action-input" value={actorId} onChange={(event) => setActorId(event.target.value)} />
            </div>
          </div>
        </div>
      </div>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={sortedFlags}
        loading={loading}
        emptyTitle={t("settings.flags.empty_title")}
        emptyDesc={t("settings.flags.empty_desc")}
        keyExtractor={(row) => row.key}
      />
    </div>
  )
}
