import { useCallback, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { Textarea } from "../components/ui/textarea"

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function TaxesCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState({
    code: "",
    name: "",
    percentage: "",
    inclusive: false,
    active: true,
    metadata: ""
  })

  const validation = useMemo(() => {
    const errors: string[] = []
    if (!form.code.trim()) errors.push(t("taxes_create.validation.code_required"))
    if (!form.name.trim()) errors.push(t("taxes_create.validation.name_required"))
    if (!form.percentage.trim()) {
      errors.push(t("taxes_create.validation.rate_required"))
    } else {
      const parsed = Number.parseFloat(form.percentage)
      if (Number.isNaN(parsed)) errors.push(t("taxes_create.validation.rate_number"))
      if (!Number.isNaN(parsed) && parsed < 0) errors.push(t("taxes_create.validation.rate_min"))
    }
    if (form.metadata.trim()) {
      try {
        const parsed = JSON.parse(form.metadata)
        if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
          errors.push(t("taxes_create.validation.metadata_object"))
        }
      } catch {
        errors.push(t("taxes_create.validation.metadata_invalid"))
      }
    }
    return errors
  }, [form, t])

  const handleSubmit = useCallback(async () => {
    try {
      setSaving(true)
      const percentage = Number.parseFloat(form.percentage)
      const payload: {
        code: string
        name: string
        percentage: number
        inclusive: boolean
        active: boolean
        metadata?: Record<string, unknown>
      } = {
        code: form.code.trim(),
        name: form.name.trim(),
        percentage,
        inclusive: form.inclusive,
        active: form.active
      }
      if (form.metadata.trim()) {
        payload.metadata = JSON.parse(form.metadata) as Record<string, unknown>
      }
      const resp = await api.taxes.create(payload)
      toast.success(t("taxes_create.toast.created"), resp.id)
      navigate(orgPath("/taxes"))
    } catch (err) {
      toast.error(t("taxes_create.toast.create_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [form, navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader
        title={t("taxes_create.header.title")}
        description={t("taxes_create.header.description")}
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/taxes"))}
        style={{ cursor: "pointer" }}
      />
      <div className="panel" style={{ maxWidth: 720 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("taxes_create.fields.code")}</label>
              <Input
                className="action-input"
                value={form.code}
                onChange={(e) => setForm((p) => ({ ...p, code: e.target.value }))}
                data-testid="taxes-create-code"
              />
            </div>
            <div className="action-field">
              <label className="action-label">{t("taxes_create.fields.name")}</label>
              <Input
                className="action-input"
                value={form.name}
                onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
                data-testid="taxes-create-name"
              />
            </div>
            <div className="action-field">
              <label className="action-label">{t("taxes_create.fields.rate")}</label>
              <Input
                className="action-input"
                type="number"
                min="0"
                step="0.01"
                value={form.percentage}
                onChange={(e) => setForm((p) => ({ ...p, percentage: e.target.value }))}
                data-testid="taxes-create-percentage"
              />
            </div>
            <div className="action-field">
              <label className="action-label">{t("taxes_create.fields.inclusive")}</label>
              <Select
                value={form.inclusive ? "true" : "false"}
                onValueChange={(value) => setForm((p) => ({ ...p, inclusive: value === "true" }))}
              >
                <SelectTrigger className="action-select" data-testid="taxes-create-inclusive">
                  <SelectValue placeholder={t("taxes_create.fields.inclusive_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="false">{t("taxes_create.fields.inclusive_options.exclusive")}</SelectItem>
                  <SelectItem value="true">{t("taxes_create.fields.inclusive_options.inclusive")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field">
              <label className="action-label">{t("taxes_create.fields.status")}</label>
              <Select
                value={form.active ? "true" : "false"}
                onValueChange={(value) => setForm((p) => ({ ...p, active: value === "true" }))}
              >
                <SelectTrigger className="action-select" data-testid="taxes-create-active">
                  <SelectValue placeholder={t("taxes_create.fields.status_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="true">{t("taxes_create.fields.status_options.active")}</SelectItem>
                  <SelectItem value="false">{t("taxes_create.fields.status_options.inactive")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field" style={{ gridColumn: "1 / -1" }}>
              <label className="action-label">
                {t("taxes_create.fields.metadata")} <HelpHint text={t("taxes_create.fields.metadata_hint")} />
              </label>
              <Textarea
                className="action-input"
                rows={4}
                placeholder={t("taxes_create.fields.metadata_placeholder")}
                value={form.metadata}
                onChange={(e) => setForm((p) => ({ ...p, metadata: e.target.value }))}
                data-testid="taxes-create-metadata"
              />
            </div>
          </div>
          {validation.length > 0 ? <div className="inline-error">{validation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="outline" onClick={() => navigate(orgPath("/taxes"))} data-testid="taxes-create-cancel">{t("common.cancel")}</Button>
            <Button
              variant="default"
              disabled={saving || validation.length > 0}
              onClick={handleSubmit}
              data-testid="taxes-create-submit"
            >
              {saving ? t("common.creating") : t("taxes_create.actions.create")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
