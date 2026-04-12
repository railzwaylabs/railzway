import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { useParams, useNavigate } from "react-router-dom"
import PageHeader from "../components/PageHeader"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import AutoCompleteInput from "../components/AutoCompleteInput"
import type { Feature } from "../lib/types"

export default function FeaturesEdit() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  
  const [feature, setFeature] = useState<Feature | null>(null)
  const [form, setForm] = useState({
    name: "",
    description: "",
    active: true
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadFeature = useCallback(async () => {
    if (!id) return
    try {
      setLoading(true)
      const data = await api.features.get(id)
      setFeature(data)
      setForm({
        name: data.name,
        description: data.description || "",
        active: data.active
      })
    } catch (err) {
      setError(t("features_edit.toast.load_failed") || "Failed to load feature")
    } finally {
      setLoading(false)
    }
  }, [id, t])

  useEffect(() => { void loadFeature() }, [loadFeature])

  const handleSubmit = async () => {
    if (!id) return
    try {
      setSaving(true)
      setError(null)
      await api.features.update(id, {
        name: form.name,
        description: form.description || undefined,
        active: form.active
      })
      navigate(orgPath("/features"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("features_edit.toast.update_failed"))
    } finally {
      setSaving(false)
    }
  }

  const disabled = saving || !form.name

  if (loading) return <div className="page-content">{t("common.loading")}</div>
  if (!feature) return <div className="page-content">{t("not_found.title")}</div>

  return (
    <div className="page-content">
      <PageHeader
        title={t("features_edit.header.title_with_name", { name: feature.code })}
        description={t("features.header.description")}
      />

      <div className="panel" style={{ maxWidth: 720 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">{t("features_create.sections.feature")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("common.name") || "Name"} *</Label>
              <Input
                className="action-input"
                value={form.name}
                onChange={(e) => setForm(p => ({ ...p, name: e.target.value }))}
              />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_create.fields.description")}</Label>
              <Input
                className="action-input"
                value={form.description}
                onChange={(e) => setForm(p => ({ ...p, description: e.target.value }))}
              />
            </div>
            <div className="action-field">
              <AutoCompleteInput
                id="feature-active"
                label={<>{t("plans.table.columns.status")}</>}
                value={form.active ? "true" : "false"}
                options={[
                  {value: "true", label: t("plans.table.status.active")},
                  {value: "false", label: t("plans.table.status.inactive")}
                ]}
                onChange={(v) => setForm(p => ({ ...p, active: v === "true" }))}
              />
            </div>
          </div>
          {error && <div className="inline-error">{error}</div>}
          <div className="action-buttons">
            <Button onClick={handleSubmit} disabled={disabled}>
              {saving ? t("common.saving") : t("common.save_changes")}
            </Button>
            <Button variant="secondary" onClick={() => navigate(orgPath("/features"))} disabled={saving}>
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function setFormUnsafe(fn: (p: any) => any) {
  // Helper
}
