import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate, useParams } from "react-router-dom"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import type { Meter } from "../lib/types"
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

export default function MetersEdit() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const orgPath = useOrgPath()

  const [meter, setMeter] = useState<Meter | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  const [updateForm, setUpdateForm] = useState({ name: "", aggregation: "", unit: "", active: "" })

  const loadData = useCallback(async () => {
    if (!id) return
    try {
      setLoading(true)
      let found: Meter | undefined
      let pageToken: string | undefined
      while (!found) {
        const resp = await api.meters.list({ page_token: pageToken, page_size: 50 })
        found = resp.meters.find(m => m.id === id)
        pageToken = resp.next_page_token
        if (!resp.has_more && !pageToken) break
      }

      if (found) {
        setMeter(found)
        setUpdateForm({ name: found.name, aggregation: found.aggregation, unit: found.unit, active: String(found.active) })
      }
    } catch (err) {
      toast.error(t("meters_edit.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [id, t])

  useEffect(() => { void loadData() }, [loadData])

  const validation = useMemo(() => {
    const e: string[] = []
    const has = Boolean(updateForm.name.trim() || updateForm.aggregation.trim() || updateForm.unit.trim() || updateForm.active)
    if (!has) e.push(t("meters_edit.validation.no_changes"))
    return e
  }, [updateForm, t])

  const handleUpdate = useCallback(async () => {
    if (!id) return
    const payload: Record<string, unknown> = {}
    if (updateForm.name.trim()) payload.name = updateForm.name.trim()
    if (updateForm.aggregation.trim()) payload.aggregation = updateForm.aggregation.trim()
    if (updateForm.unit.trim()) payload.unit = updateForm.unit.trim()
    if (updateForm.active !== "") payload.active = updateForm.active === "true"
    try {
      setActionLoading(true)
      const resp = await api.meters.update(id, payload)
      toast.success(t("meters_edit.toast.updated"), resp.id)
      void loadData()
    } catch (err) {
      toast.error(t("meters_edit.toast.update_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, updateForm, loadData, t])

  if (loading) return <div className="page-content"><div className="loader" /></div>

  return (
    <div className="page-content">
      <PageHeader 
        title={meter ? t("meters_edit.header.title_with_name", { name: meter.name }) : t("meters_edit.header.title")} 
        description={meter ? `${meter.code} — ${meter.aggregation} by ${meter.unit}` : ""} 
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/meters"))}
        style={{ cursor: "pointer" }}
      />
      
      <div className="action-panel">
        <div className="action-section">
          <div className="action-section-title">{t("meters_edit.section.title")}</div>
          <div className="action-fields">
            <div className="action-field">
            <label className="action-label">{t("meters_edit.fields.name")}</label>
            <Input className="action-input" value={updateForm.name}
                onChange={(e) => setUpdateForm((p) => ({ ...p, name: e.target.value }))} data-testid="meters-edit-name" />
          </div>
          <div className="action-field">
            <label className="action-label">{t("meters_edit.fields.aggregation")}</label>
            <Select
              value={toSelectValue(updateForm.aggregation)}
              onValueChange={(value) => setUpdateForm((p) => ({ ...p, aggregation: fromSelectValue(value) }))}
            >
              <SelectTrigger className="action-select" data-testid="meters-edit-aggregation">
                <SelectValue placeholder={t("common.no_change")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.no_change")}</SelectItem>
                <SelectItem value="sum">{t("meters_edit.fields.aggregation_options.sum")}</SelectItem>
                <SelectItem value="count">{t("meters_edit.fields.aggregation_options.count")}</SelectItem>
                <SelectItem value="max">{t("meters_edit.fields.aggregation_options.max")}</SelectItem>
                <SelectItem value="last">{t("meters_edit.fields.aggregation_options.last")}</SelectItem>
                <SelectItem value="avg">{t("meters_edit.fields.aggregation_options.avg")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="action-field">
            <label className="action-label">{t("meters_edit.fields.unit")}</label>
            <Input
              className="action-input"
              list="meter-units"
              value={updateForm.unit}
              placeholder={t("meters_edit.fields.unit_placeholder")}
              onChange={(e) => setUpdateForm((p) => ({ ...p, unit: e.target.value }))}
              data-testid="meters-edit-unit"
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
            <label className="action-label">{t("meters_edit.fields.status")}</label>
            <Select
              value={toSelectValue(updateForm.active)}
              onValueChange={(value) => setUpdateForm((p) => ({ ...p, active: fromSelectValue(value) }))}
            >
              <SelectTrigger className="action-select" data-testid="meters-edit-active">
                <SelectValue placeholder={t("common.no_change")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.no_change")}</SelectItem>
                <SelectItem value="true">{t("meters_edit.fields.status_options.active")}</SelectItem>
                <SelectItem value="false">{t("meters_edit.fields.status_options.inactive")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        {validation.length > 0 ? <div className="inline-error">{validation.join(" ")}</div> : null}
        <div className="action-buttons">
          <Button variant="default" disabled={actionLoading || validation.length > 0} onClick={handleUpdate} data-testid="meters-edit-submit">
            {actionLoading ? t("common.updating") : t("meters_edit.actions.update")}
          </Button>
        </div>
        </div>
      </div>
    </div>
  )
}
