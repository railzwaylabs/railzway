import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { normalizeDate, rfc3339Hint } from "../lib/display"
import { useOrgPath } from "../lib/org"

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function InvoicesCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [actionLoading, setActionLoading] = useState(false)
  const [subscriptionOptions, setSubscriptionOptions] = useState<Array<{ value: string; label: string }>>([])
  
  const [generateForm, setGenerateForm] = useState({
    subscriptionId: "",
    periodStart: "",
    periodEnd: "",
    issueAt: "",
    dueAt: ""
  })

  useEffect(() => {
    async function loadOpts() {
      try {
        const resp = await api.subscriptions.list({ page_size: 100 })
        setSubscriptionOptions(resp.subscriptions.map(s => ({ value: s.id, label: s.id })))
      } catch (err) { /* ignore */ }
    }
    void loadOpts()
  }, [])

  const validation = useMemo(() => {
    const errors: string[] = []
    if (!generateForm.subscriptionId.trim()) errors.push(t("invoices_create.validation.subscription_required"))
    if (!generateForm.periodStart.trim()) errors.push(t("invoices_create.validation.period_start_required"))
    if (!generateForm.periodEnd.trim()) errors.push(t("invoices_create.validation.period_end_required"))
    if (generateForm.periodStart && generateForm.periodEnd) {
      const start = new Date(generateForm.periodStart)
      const end = new Date(generateForm.periodEnd)
      if (!isNaN(start.getTime()) && !isNaN(end.getTime()) && end < start) {
        errors.push(t("invoices_create.validation.period_end_after"))
      }
    }
    return errors
  }, [generateForm, t])

  const handleGenerate = useCallback(async () => {
    try {
      setActionLoading(true)
      const resp = await api.invoices.generate({
        subscription_id: generateForm.subscriptionId.trim(),
        period_start: normalizeDate(generateForm.periodStart),
        period_end: normalizeDate(generateForm.periodEnd),
        issue_at: generateForm.issueAt.trim() ? normalizeDate(generateForm.issueAt) : undefined,
        due_at: generateForm.dueAt.trim() ? normalizeDate(generateForm.dueAt) : undefined
      })
      toast.success(t("invoices_create.toast.generated"), resp.number)
      navigate(orgPath(`/invoices/${resp.id}/manage`))
    } catch (err) {
      toast.error(t("invoices_create.toast.generate_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setActionLoading(false)
    }
  }, [generateForm, navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader
        title={t("invoices_create.header.title")}
        description={t("invoices_create.header.description")}
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/invoices"))}
        style={{ cursor: "pointer" }}
      />
      
      <div className="panel" style={{ maxWidth: 640 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-fields">
            <div className="action-field">
              <AutoCompleteInput
                id="invoice-subscription-id"
                label={t("invoices_create.fields.subscription_label")}
                value={generateForm.subscriptionId}
                options={subscriptionOptions}
                placeholder={t("invoices_create.fields.subscription_placeholder")}
                onChange={(value) => setGenerateForm((prev) => ({ ...prev, subscriptionId: value }))}
              />
            </div>
            
            <div className="action-field">
              <Label className="action-label">{t("invoices_create.fields.period_start")} <HelpHint text={rfc3339Hint}/></Label>
              <Input className="action-input" type="date" value={generateForm.periodStart} placeholder={t("invoices_create.fields.period_start_placeholder")}
                  onChange={(e) => setGenerateForm((prev) => ({ ...prev, periodStart: e.target.value }))} data-testid="invoices-create-period-start" />
            </div>

            <div className="action-field">
              <Label className="action-label">{t("invoices_create.fields.period_end")} <HelpHint text={rfc3339Hint}/></Label>
              <Input className="action-input" type="date" min={generateForm.periodStart || undefined} value={generateForm.periodEnd} placeholder={t("invoices_create.fields.period_end_placeholder")}
                  onChange={(e) => setGenerateForm((prev) => ({ ...prev, periodEnd: e.target.value }))} data-testid="invoices-create-period-end" />
            </div>

            <div className="action-field">
              <Label className="action-label">{t("invoices_create.fields.issue_at")} <HelpHint text={rfc3339Hint}/></Label>
              <Input className="action-input" type="datetime-local" value={generateForm.issueAt} placeholder={t("invoices_create.fields.optional_placeholder")}
                  onChange={(e) => setGenerateForm((prev) => ({ ...prev, issueAt: e.target.value }))} data-testid="invoices-create-issue-at" />
            </div>

            <div className="action-field">
              <Label className="action-label">{t("invoices_create.fields.due_at")} <HelpHint text={rfc3339Hint}/></Label>
              <Input className="action-input" type="datetime-local" min={generateForm.issueAt || undefined} value={generateForm.dueAt} placeholder={t("invoices_create.fields.optional_placeholder")}
                  onChange={(e) => setGenerateForm((prev) => ({ ...prev, dueAt: e.target.value }))} data-testid="invoices-create-due-at" />
            </div>
          </div>
          
          {validation.length > 0 && <div className="inline-error">{validation.join(" ")}</div>}
          
          <div className="action-buttons">
            <Button variant="outline" onClick={() => navigate(orgPath("/invoices"))} data-testid="invoices-create-cancel">{t("common.cancel")}</Button>
            <Button variant="default" disabled={actionLoading || validation.length > 0} onClick={handleGenerate} data-testid="invoices-create-submit">
              {actionLoading ? t("invoices_create.actions.generating") : t("invoices_create.actions.generate")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
