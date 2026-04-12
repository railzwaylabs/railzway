import { useCallback, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function MetersCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [actionLoading, setActionLoading] = useState(false)
  const [createForm, setCreateForm] = useState({ code: "", name: "", aggregation: "", unit: "", active: true })

  const validation = useMemo(() => {
    const e: string[] = []
    if (!createForm.code.trim()) e.push(t("meters_create.validation.code_required"))
    if (!createForm.name.trim()) e.push(t("meters_create.validation.name_required"))
    if (!createForm.aggregation.trim()) e.push(t("meters_create.validation.aggregation_required"))
    if (!createForm.unit.trim()) e.push(t("meters_create.validation.unit_required"))
    return e
  }, [createForm, t])

  const handleCreate = useCallback(async () => {
    try {
      setActionLoading(true)
      const resp = await api.meters.create({
        code: createForm.code.trim(), name: createForm.name.trim(),
        aggregation: createForm.aggregation.trim(), unit: createForm.unit.trim(), active: createForm.active,
      })
      toast.success(t("meters_create.toast.created"), resp.id)
      navigate(orgPath(`/meters/${resp.id}/edit`))
    } catch (err) {
      toast.error(t("meters_create.toast.create_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [createForm, navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader
        title={t("meters_create.header.title")}
        description={t("meters_create.header.description")}
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/meters"))}
        style={{ cursor: "pointer" }}
      />
      <div className="panel" style={{ maxWidth: 640 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-fields">
            <div className="action-field">
            <label className="action-label">{t("meters_create.fields.code")}</label>
            <Input className="action-input" value={createForm.code} autoFocus
                onChange={(e) => setCreateForm((p) => ({ ...p, code: e.target.value }))} data-testid="meters-create-code" />
          </div>
          <div className="action-field">
            <label className="action-label">{t("meters_create.fields.name")}</label>
            <Input className="action-input" value={createForm.name}
                onChange={(e) => setCreateForm((p) => ({ ...p, name: e.target.value }))} data-testid="meters-create-name" />
          </div>
          <div className="action-field">
            <label className="action-label">{t("meters_create.fields.aggregation")}</label>
            <Select
              value={toSelectValue(createForm.aggregation)}
              onValueChange={(value) => setCreateForm((p) => ({ ...p, aggregation: fromSelectValue(value) }))}
            >
              <SelectTrigger className="action-select" data-testid="meters-create-aggregation">
                <SelectValue placeholder={t("meters_create.fields.aggregation_placeholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("meters_create.fields.aggregation_placeholder")}</SelectItem>
                <SelectItem value="sum">{t("meters_create.fields.aggregation_options.sum")}</SelectItem>
                <SelectItem value="count">{t("meters_create.fields.aggregation_options.count")}</SelectItem>
                <SelectItem value="max">{t("meters_create.fields.aggregation_options.max")}</SelectItem>
                <SelectItem value="last">{t("meters_create.fields.aggregation_options.last")}</SelectItem>
                <SelectItem value="avg">{t("meters_create.fields.aggregation_options.avg")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="action-field">
            <label className="action-label">{t("meters_create.fields.unit")}</label>
            <Input
              className="action-input"
              list="meter-units"
              value={createForm.unit}
              placeholder={t("meters_create.fields.unit_placeholder")}
              onChange={(e) => setCreateForm((p) => ({ ...p, unit: e.target.value }))}
              data-testid="meters-create-unit"
            />
            <datalist id="meter-units">
              <option value="requests" />
              <option value="tokens" />
              <option value="characters" />
              <option value="seconds" />
              <option value="minutes" />
              <option value="hours" />
              <option value="days" />
              <option value="rows" />
              <option value="mb" />
              <option value="gb" />
              <option value="tb" />
            </datalist>
          </div>
          <div className="action-field">
            <label className="action-label">{t("meters_create.fields.status")}</label>
            <Select
              value={createForm.active ? "true" : "false"}
              onValueChange={(value) => setCreateForm((p) => ({ ...p, active: value === "true" }))}
            >
              <SelectTrigger className="action-select" data-testid="meters-create-active">
                <SelectValue placeholder={t("meters_create.fields.status_placeholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="true">{t("meters_create.fields.status_options.active")}</SelectItem>
                <SelectItem value="false">{t("meters_create.fields.status_options.inactive")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          </div>
          {validation.length > 0 ? <div className="inline-error">{validation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="outline" onClick={() => navigate(orgPath("/meters"))} data-testid="meters-create-cancel">{t("common.cancel")}</Button>
            <Button variant="default" disabled={actionLoading || validation.length > 0} onClick={handleCreate} data-testid="meters-create-submit">
              {actionLoading ? t("common.creating") : t("meters_create.actions.create")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
