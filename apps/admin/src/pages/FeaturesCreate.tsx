import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import PageHeader from "../components/PageHeader"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import AutoCompleteInput from "../components/AutoCompleteInput"

export default function FeaturesCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()

  const [form, setForm] = useState({
    code: "",
    name: "",
    description: "",
    featureType: "boolean",
    meterId: ""
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const featureTypes = [
    { value: "boolean", label: t("features.type.boolean") },
    { value: "metered", label: t("features.type.metered") }
  ]

  const handleSubmit = async () => {
    try {
      setSaving(true)
      setError(null)
      await api.features.create({
        code: form.code,
        name: form.name,
        description: form.description || undefined,
        feature_type: form.featureType,
        meter_id: form.featureType === "metered" && form.meterId ? form.meterId : undefined,
        active: true
      })
      navigate(orgPath("/features"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("features_create.toast.create_failed"))
    } finally {
      setSaving(false)
    }
  }

  const disabled = saving || !form.code || !form.name

  return (
    <div className="page-content">
      <PageHeader
        title={t("features_create.header.title")}
        description={t("features_create.header.description")}
      />

      <div className="panel" style={{ maxWidth: 720 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">{t("features_create.sections.feature")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("plans_create.fields.plan_code")} *</Label>
              <Input
                className="action-input"
                value={form.code}
                onChange={(e) => setForm(p => ({ ...p, code: e.target.value }))}
                placeholder={t("features_create.placeholders.code")}
              />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("common.name") || "Name"} *</Label>
              <Input
                className="action-input"
                value={form.name}
                onChange={(e) => setForm(p => ({ ...p, name: e.target.value }))}
                placeholder={t("features_create.placeholders.name")}
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
                id="feature-type"
                label={<>{t("plans_edit.price_fields.type")} *</>}
                value={form.featureType}
                options={featureTypes}
                onChange={(v) => setForm(p => ({ ...p, featureType: v }))}
              />
            </div>
            {form.featureType === "metered" && (
              <div className="action-field">
                <Label className="action-label">{t("plans_edit.price_fields.meter")}</Label>
                <Input
                  className="action-input"
                  value={form.meterId}
                  onChange={(e) => setForm(p => ({ ...p, meterId: e.target.value }))}
                  placeholder={t("features_create.placeholders.meter_hint")}
                />
              </div>
            )}
          </div>
          {error && <div className="inline-error">{error}</div>}
          <div className="action-buttons">
            <Button onClick={handleSubmit} disabled={disabled}>
              {saving ? t("common.creating") : t("features_create.actions.save")}
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
  // Helper for AutoCompleteInput which might not be strictly typed in some contexts
  // but here we just use the setForm state setter
}
