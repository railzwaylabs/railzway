import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate, useParams } from "react-router-dom"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { useOrgIdParam, useOrgPath } from "../lib/org"
import type { Customer, Invoice, InvoiceItem, Organization } from "../lib/types"
import { InvoiceDetail, type InvoiceDetailData } from "@railzway/invoice-ui"

export default function InvoicesManage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const orgId = useOrgIdParam()

  const [invoice, setInvoice] = useState<Invoice | null>(null)
  const [items, setItems] = useState<InvoiceItem[]>([])
  const [organization, setOrganization] = useState<Organization | null>(null)
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  const [actionForm, setActionForm] = useState({
    reason: "",
    attachmentUrl: "",
    note: ""
  })

  const humanizeLineType = useCallback((value?: string): string => {
    if (!value) return t("invoices_manage.line_item")
    return value.replace(/_/g, " ").replace(/\b\w/g, (m) => m.toUpperCase())
  }, [t])

  const loadData = useCallback(async () => {
    if (!id) return
    try {
      setLoading(true)
      const resp = await api.invoices.get(id)
      setInvoice(resp.invoice)
      setItems(resp.items)
      const fetches: Promise<void>[] = []
      if (orgId) {
        fetches.push(api.organizations.get(orgId).then(setOrganization))
      }
      if (resp.invoice.customer_id) {
        fetches.push(api.customers.get(resp.invoice.customer_id).then(setCustomer))
      }
      if (fetches.length) {
        await Promise.all(fetches)
      }
    } catch (err) {
      toast.error(t("invoices_manage.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [id, orgId, t])

  useEffect(() => { void loadData() }, [loadData])

  const handleInvoiceAction = useCallback(async (action: "open" | "pay" | "void" | "markPaid") => {
    if (!id) return
    try {
      setActionLoading(true)
      const payload = {
        reason: actionForm.reason.trim() || undefined,
        attachment_url: actionForm.attachmentUrl.trim() || undefined,
        note: actionForm.note.trim() || undefined
      }

      if (action === "open") await api.invoices.open(id)
      else if (action === "pay") await api.invoices.pay(id, payload)
      else if (action === "void") await api.invoices.void(id, payload)
      else await api.invoices.markPaid(id, payload)

      toast.success(t("invoices_manage.toast.action_success", { action: action === "markPaid" ? t("invoices_manage.actions.mark_paid") : action }))
      setActionForm({ reason: "", attachmentUrl: "", note: "" })
      void loadData()
    } catch (err) {
      toast.error(t("invoices_manage.toast.action_failed", { action }), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, actionForm, loadData, t])

  const detailData = useMemo<InvoiceDetailData | null>(() => {
    if (!invoice) return null
    const lineItems = items.map((item) => {
      const title = item.description || humanizeLineType(item.line_type)
      const description = item.description && item.description !== title ? item.description : undefined
      return {
        id: item.id,
        title,
        description,
        quantity: item.quantity,
        unitAmountCents: item.unit_amount_cents,
        amountCents: item.amount_cents,
        currency: item.currency,
        periodStart: item.period_start,
        periodEnd: item.period_end,
        lineType: item.line_type
      }
    })
    return {
      invoice: {
        number: invoice.number,
        status: invoice.status,
        currency: invoice.currency,
        issueDate: invoice.issued_at ?? invoice.created_at,
        dueDate: invoice.due_at,
        periodStart: invoice.period_start,
        periodEnd: invoice.period_end,
        subtotalCents: invoice.subtotal_cents,
        taxCents: invoice.tax_cents,
        totalCents: invoice.total_cents,
        amountDueCents: invoice.amount_due_cents,
        amountPaidCents: invoice.amount_paid_cents
      },
      billedFrom: organization ? {
        name: organization.name,
        country: organization.country_code ?? undefined
      } : undefined,
      billedTo: customer ? {
        name: customer.name,
        email: customer.email
      } : undefined,
      lineItems
    }
  }, [invoice, items, organization, customer])

  const handleResend = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      const resp = await api.invoices.resend(id)
      if (resp.public_link?.url) {
        await navigator.clipboard.writeText(resp.public_link.url)
        toast.success(t("invoices_manage.toast.resend_copied"))
      } else if (resp.public_link?.token) {
        toast.success(t("invoices_manage.toast.resend_generated"))
      } else {
        toast.success(t("invoices_manage.toast.resend_queued"))
      }
    } catch (err) {
      toast.error(t("invoices_manage.toast.resend_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setActionLoading(false)
    }
  }, [id, t])

  if (loading) return <div className="page-content"><div className="loader" /></div>
  if (!invoice || !detailData) return <div className="page-content"><div className="inline-error">{t("invoices_manage.not_found")}</div></div>

  return (
    <div className="page-content">
      <InvoiceDetail
        data={detailData}
        variant="embedded"
        onBack={() => navigate(orgPath("/invoices"))}
        backLabel={t("invoices_manage.actions.back")}
        actions={
          <>
            <Button variant="outline" disabled>
              {t("invoices_manage.actions.download")}
            </Button>
            <Button variant="outline" onClick={handleResend} disabled={actionLoading} data-testid="invoices-resend-email">
              {t("invoices_manage.actions.resend")}
            </Button>
          </>
        }
      />

      <div className="action-panel">
        <div className="action-section">
          <div className="action-section-title">{t("invoices_manage.actions.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("invoices_manage.fields.reason")}</Label>
              <Input className="action-input" value={actionForm.reason} placeholder={t("invoices_manage.fields.reason_placeholder")}
                onChange={(e) => setActionForm(p => ({ ...p, reason: e.target.value }))} data-testid="invoices-manage-reason" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("invoices_manage.fields.attachment")}</Label>
              <Input className="action-input" value={actionForm.attachmentUrl} placeholder={t("invoices_manage.fields.attachment_placeholder")}
                onChange={(e) => setActionForm(p => ({ ...p, attachmentUrl: e.target.value }))} data-testid="invoices-manage-attachment" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("invoices_manage.fields.note")}</Label>
              <Input className="action-input" value={actionForm.note} placeholder={t("invoices_manage.fields.note_placeholder")}
                onChange={(e) => setActionForm(p => ({ ...p, note: e.target.value }))} data-testid="invoices-manage-note" />
            </div>
          </div>

          <div className="action-buttons" style={{ marginTop: 24, justifyContent: "flex-start", flexWrap: "wrap" }}>
            <Button variant="default" onClick={() => handleInvoiceAction("open")} disabled={actionLoading} data-testid="invoices-manage-open">{t("invoices_manage.actions.open")}</Button>
            <Button variant="default" style={{ backgroundColor: "hsl(var(--status-success))", color: "hsl(var(--text-inverse))" }} onClick={() => handleInvoiceAction("pay")} disabled={actionLoading} data-testid="invoices-manage-pay">{t("invoices_manage.actions.pay")}</Button>
            <Button variant="secondary" onClick={() => handleInvoiceAction("markPaid")} disabled={actionLoading} data-testid="invoices-manage-mark-paid">{t("invoices_manage.actions.mark_paid")}</Button>
            <Button variant="outline" style={{ color: "hsl(var(--status-error))" }} onClick={() => handleInvoiceAction("void")} disabled={actionLoading} data-testid="invoices-manage-void">{t("invoices_manage.actions.void")}</Button>
          </div>
        </div>
      </div>
    </div>
  )
}
