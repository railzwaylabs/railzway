import { Badge } from "./ui/badge"
import { Input } from "./ui/input"
import { Label } from "./ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select"

export type PlanFeatureDraft = {
  feature_id: string
  code: string
  name: string
  feature_type: string
  meter_id?: string
  active: boolean
  inherited: boolean
  enabled: boolean
  limit_numeric: string
  limit_unit: string
  reset_period: string
}

export function buildDraftPlanFeatures(
  available: Array<{ id: string; code: string; name: string; feature_type: string; meter_id?: string; active: boolean }>,
  existing?: Array<{ id: string; enabled: boolean; limit_numeric?: number; limit_unit?: string; reset_period: string }>
): PlanFeatureDraft[] {
  const overrideByID = new Map((existing ?? []).map((item) => [item.id, item]))
  return available
    .slice()
    .sort((a, b) => a.name.localeCompare(b.name))
    .map((feature) => {
      const override = overrideByID.get(feature.id)
      return {
        feature_id: feature.id,
        code: feature.code,
        name: feature.name,
        feature_type: feature.feature_type,
        meter_id: feature.meter_id,
        active: feature.active,
        inherited: !override,
        enabled: override?.enabled ?? true,
        limit_numeric: override?.limit_numeric != null ? String(override.limit_numeric) : "",
        limit_unit: override?.limit_unit ?? "",
        reset_period: override?.reset_period ?? "none",
      }
    })
}

type Props = {
  rows: PlanFeatureDraft[]
  disabled?: boolean
  t: (key: string, options?: Record<string, unknown>) => string
  onChange: (rows: PlanFeatureDraft[]) => void
}

export default function PlanFeatureEditor({ rows, disabled, t, onChange }: Props) {
  const updateRow = (featureID: string, patch: Partial<PlanFeatureDraft>) => {
    onChange(rows.map((row) => row.feature_id === featureID ? { ...row, ...patch, inherited: false } : row))
  }

  return (
    <div style={{ display: "grid", gap: 12 }}>
      {rows.map((row) => (
        <div
          key={row.feature_id}
          data-testid={`plan-feature-row-${row.code}`}
          style={{
            border: "1px solid var(--border)",
            borderRadius: 10,
            padding: 14,
            background: row.active ? "var(--panel)" : "var(--muted)",
            opacity: row.active ? 1 : 0.72
          }}
        >
          <div style={{ display: "flex", justifyContent: "space-between", gap: 16, alignItems: "flex-start", marginBottom: 12 }}>
            <div>
              <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                <div style={{ fontWeight: 600 }}>{row.name}</div>
                <Badge className="badge-muted">{row.feature_type}</Badge>
                {row.meter_id ? <Badge className="badge-info">{t("plans_edit.plan_features.metered")}</Badge> : null}
                {row.inherited ? <Badge className="badge-muted">{t("plans_edit.plan_features.inherited")}</Badge> : null}
                {!row.active ? <Badge className="badge-muted">{t("plans_edit.status.inactive")}</Badge> : null}
              </div>
              <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>{row.code}</div>
            </div>
            <label style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 110, justifyContent: "flex-end" }}>
              <input
                type="checkbox"
                checked={row.enabled}
                data-testid={`plan-feature-enabled-${row.code}`}
                disabled={disabled || !row.active}
                onChange={(e) => {
                  const enabled = e.target.checked
                  updateRow(row.feature_id, {
                    enabled,
                    ...(enabled ? {} : { limit_numeric: "", limit_unit: "", reset_period: "none" })
                  })
                }}
              />
              <span style={{ fontSize: 13 }}>{t("plans_edit.plan_features.enabled")}</span>
            </label>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1.1fr) minmax(0, 0.9fr) minmax(0, 1fr)", gap: 12 }}>
            <div>
              <Label className="action-label">{t("plans_edit.plan_features.limit")}</Label>
              <Input
                className="action-input"
                data-testid={`plan-feature-limit-${row.code}`}
                inputMode="decimal"
                type="text"
                placeholder={t("plans_edit.plan_features.limit_placeholder")}
                value={row.limit_numeric}
                disabled={disabled || !row.enabled || !row.active}
                onChange={(e) => updateRow(row.feature_id, { limit_numeric: e.target.value })}
              />
            </div>
            <div>
              <Label className="action-label">{t("plans_edit.plan_features.limit_unit")}</Label>
              <Input
                className="action-input"
                data-testid={`plan-feature-limit-unit-${row.code}`}
                placeholder={t("plans_edit.plan_features.limit_unit_placeholder")}
                value={row.limit_unit}
                disabled={disabled || !row.enabled || !row.active}
                onChange={(e) => updateRow(row.feature_id, { limit_unit: e.target.value })}
              />
            </div>
            <div>
              <Label className="action-label">{t("plans_edit.plan_features.reset_period")}</Label>
              <Select
                value={row.reset_period}
                onValueChange={(value) => updateRow(row.feature_id, { reset_period: value })}
                disabled={disabled || !row.enabled || !row.active}
              >
                <SelectTrigger className="action-select" data-testid={`plan-feature-reset-period-${row.code}`}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t("plans_edit.plan_features.reset_options.none")}</SelectItem>
                  <SelectItem value="day">{t("plans_edit.plan_features.reset_options.day")}</SelectItem>
                  <SelectItem value="month">{t("plans_edit.plan_features.reset_options.month")}</SelectItem>
                  <SelectItem value="billing_period">{t("plans_edit.plan_features.reset_options.billing_period")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
