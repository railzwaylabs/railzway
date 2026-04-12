import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { currencyHint } from "../lib/hints"
import { useCurrencies } from "../lib/reference"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { rfc3339Hint } from "../lib/display"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"

type EntryDraft = { accountCode: string; entryType: string; amountCents: string; currency: string; description: string }

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function LedgerCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies()
  const [actionLoading, setActionLoading] = useState(false)
  const [customerOptions, setCustomerOptions] = useState<Array<{ value: string; label: string }>>([])
  const [subscriptionOptions, setSubscriptionOptions] = useState<Array<{ value: string; label: string }>>([])
  const [invoiceOptions, setInvoiceOptions] = useState<Array<{ value: string; label: string }>>([])

  const [createForm, setCreateForm] = useState({
    currency: "USD", sourceType: "", sourceId: "", customerId: "", subscriptionId: "", invoiceId: "", occurredAt: "",
  })
  const [entries, setEntries] = useState<EntryDraft[]>([
    { accountCode: "", entryType: "debit", amountCents: "", currency: "USD", description: "" },
    { accountCode: "", entryType: "credit", amountCents: "", currency: "USD", description: "" },
  ])

  useEffect(() => {
    let active = true
    const load = async () => {
      try {
        const [cResp, sResp, iResp] = await Promise.all([
          api.customers.list({ page_size: 50 }),
          api.subscriptions.list({ page_size: 50 }),
          api.invoices.list({ page_size: 50 }),
        ])
        if (!active) return
        setCustomerOptions(cResp.customers.map((c) => ({ value: c.id, label: `${c.name} · ${c.email}` })))
        setSubscriptionOptions(sResp.subscriptions.map((s) => ({ value: s.id, label: `${s.id.slice(0, 8)}… · ${s.status}` })))
        setInvoiceOptions(iResp.invoices.map((i) => ({ value: i.id, label: `${i.number} · ${i.status}` })))
      } catch { /* ignore */ }
    }
    void load()
    return () => { active = false }
  }, [])

  const searchCustomers = useCallback(async (query: string) => {
    const trimmed = query.trim()
    if (!trimmed) return []
    const params = trimmed.includes("@")
      ? { page_size: 50, email: trimmed }
      : { page_size: 50, name: trimmed }
    const resp = await api.customers.list(params)
    return resp.customers.map((customer) => ({
      value: customer.id,
      label: `${customer.name} · ${customer.email}`
    }))
  }, [])

  const searchInvoices = useCallback(async (query: string) => {
    const trimmed = query.trim()
    if (!trimmed) return []
    const resp = await api.invoices.list({ page_size: 50, number: trimmed })
    return resp.invoices.map((invoice) => ({
      value: invoice.id,
      label: `${invoice.number} · ${invoice.status}`
    }))
  }, [])

  const handleAddEntry = () =>
    setEntries((p) => [...p, { accountCode: "", entryType: "debit", amountCents: "", currency: "USD", description: "" }])

  const handleEntryChange = (idx: number, field: keyof EntryDraft, value: string) =>
    setEntries((p) => p.map((e, i) => (i === idx ? { ...e, [field]: value } : e)))

  const handleCreate = useCallback(async () => {
    const payloadEntries = entries
      .filter((e) => e.accountCode.trim() && e.amountCents.trim())
      .map((e) => ({
        account_code: e.accountCode.trim(), entry_type: e.entryType.trim(),
        amount_cents: Number.parseInt(e.amountCents, 10), currency: e.currency.trim(),
        description: e.description.trim() || undefined,
      }))
    if (payloadEntries.length < 2) { toast.error(t("ledger_create.validation.entries_min")); return }
    try {
      setActionLoading(true)
      const resp = await api.ledger.createTransaction({
        currency: createForm.currency.trim(), source_type: createForm.sourceType.trim(),
        source_id: createForm.sourceId.trim(), customer_id: createForm.customerId.trim() || undefined,
        subscription_id: createForm.subscriptionId.trim() || undefined,
        invoice_id: createForm.invoiceId.trim() || undefined,
        occurred_at: createForm.occurredAt.trim() || undefined, entries: payloadEntries,
      })
      toast.success(t("ledger_create.toast.posted"), resp.id)
      navigate(orgPath("/ledger"))
    } catch (err) {
      toast.error(t("ledger_create.toast.post_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [createForm, entries, navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader 
        title={t("ledger_create.header.title")} 
        description={t("ledger_create.header.description")}
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/ledger"))}
        style={{ cursor: "pointer" }}
      />

      <div className="panel" style={{ maxWidth: 840 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">{t("ledger_create.sections.details")}</div>
          <div className="action-fields">
            <div className="action-field">
              <AutoCompleteInput
                id="ledger-create-currency"
                label={<>{t("ledger_create.fields.currency")} <HelpHint text={currencyHint} /></>}
                value={createForm.currency}
                options={currencyOptions}
                placeholder={currenciesLoading ? t("common.loading") : undefined}
                onChange={(value) => setCreateForm((p) => ({ ...p, currency: value }))}
              />
            </div>
            <div className="action-field"><label className="action-label">{t("ledger_create.fields.source_type")}</label>
              <Input className="action-input" value={createForm.sourceType} placeholder={t("ledger_create.fields.source_type_placeholder")} onChange={(e) => setCreateForm((p) => ({ ...p, sourceType: e.target.value }))} data-testid="ledger-create-source-type" /></div>
            <div className="action-field"><label className="action-label">{t("ledger_create.fields.source_id")}</label>
              <Input className="action-input" value={createForm.sourceId} onChange={(e) => setCreateForm((p) => ({ ...p, sourceId: e.target.value }))} data-testid="ledger-create-source-id" /></div>
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <AutoCompleteInput id="ledger-customer-id" label={t("ledger_create.fields.customer")} value={createForm.customerId} options={customerOptions}
                placeholder={t("ledger_create.fields.customer_placeholder")} onSearch={searchCustomers}
                onChange={(v) => setCreateForm((p) => ({ ...p, customerId: v }))} />
            </div>
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <AutoCompleteInput id="ledger-subscription-id" label={t("ledger_create.fields.subscription")} value={createForm.subscriptionId} options={subscriptionOptions}
                placeholder={t("ledger_create.fields.subscription_placeholder")} onChange={(v) => setCreateForm((p) => ({ ...p, subscriptionId: v }))} />
            </div>
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <AutoCompleteInput id="ledger-invoice-id" label={t("ledger_create.fields.invoice")} value={createForm.invoiceId} options={invoiceOptions}
                placeholder={t("ledger_create.fields.invoice_placeholder")} onSearch={searchInvoices}
                onChange={(v) => setCreateForm((p) => ({ ...p, invoiceId: v }))} />
            </div>
            <div className="action-field"><label className="action-label">{t("ledger_create.fields.occurred_at")} <HelpHint text={rfc3339Hint} /></label>
              <Input className="action-input" type="datetime-local" value={createForm.occurredAt} onChange={(e) => setCreateForm((p) => ({ ...p, occurredAt: e.target.value }))} data-testid="ledger-create-occurred-at" /></div>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("ledger_create.sections.entries")}</div>
          {entries.map((entry, idx) => (
            <div key={idx} className="action-fields" style={{ marginBottom: 12, padding: "12px 0", borderBottom: "1px solid var(--line)" }}>
              <div className="action-field"><label className="action-label">{t("ledger_create.entries.account_code")}</label>
                <Input className="action-input" value={entry.accountCode} onChange={(e) => handleEntryChange(idx, "accountCode", e.target.value)} data-testid={`ledger-entry-${idx}-account`} /></div>
              <div className="action-field"><label className="action-label">{t("ledger_create.entries.entry_type")}</label>
                <Select value={entry.entryType} onValueChange={(value) => handleEntryChange(idx, "entryType", value)}>
                  <SelectTrigger className="action-select" data-testid={`ledger-entry-${idx}-type`}>
                    <SelectValue placeholder={t("ledger_create.entries.entry_type_placeholder")} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="debit">{t("ledger_create.entries.entry_type_options.debit")}</SelectItem>
                    <SelectItem value="credit">{t("ledger_create.entries.entry_type_options.credit")}</SelectItem>
                  </SelectContent>
                </Select></div>
              <div className="action-field"><label className="action-label">{t("ledger_create.entries.amount")}</label>
                <Input className="action-input" type="number" value={entry.amountCents} onChange={(e) => handleEntryChange(idx, "amountCents", e.target.value)} data-testid={`ledger-entry-${idx}-amount`} /></div>
              <div className="action-field">
                <AutoCompleteInput
                  id={`ledger-entry-${idx}-currency`}
                  label={t("ledger_create.entries.currency")}
                  value={entry.currency}
                  options={currencyOptions}
                  placeholder={currenciesLoading ? t("common.loading") : undefined}
                  onChange={(value) => handleEntryChange(idx, "currency", value)}
                />
              </div>
              <div className="action-field" style={{ gridColumn: "span 2" }}><label className="action-label">{t("ledger_create.entries.description")}</label>
                <Input className="action-input" value={entry.description} onChange={(e) => handleEntryChange(idx, "description", e.target.value)} data-testid={`ledger-entry-${idx}-description`} /></div>
            </div>
          ))}
          <div className="action-buttons" style={{ justifyContent: "flex-start", marginTop: 24 }}>
            <Button variant="secondary" onClick={handleAddEntry} data-testid="ledger-add-entry">{t("ledger_create.actions.add_entry")}</Button>
            <Button variant="default" disabled={actionLoading} onClick={handleCreate} data-testid="ledger-post-transaction">
              {actionLoading ? t("ledger_create.actions.posting") : t("ledger_create.actions.post")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
