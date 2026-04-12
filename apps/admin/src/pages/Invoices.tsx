import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { Badge } from "../components/ui/badge"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { formatCurrency, formatNumber } from "../lib/format"
import { statusClass } from "../lib/status"
import { formatDate, shortID } from "../lib/display"
import type { Invoice, InvoicesListResponse, InvoicesSummary } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import FilterPanel from "../components/FilterPanel"
import { toast } from "../components/Toast"

function IconInvoice() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M4 4v12a2 2 0 002 2h8a2 2 0 002-2V7.5L11.5 3H6a2 2 0 00-2 2z" strokeLinecap="round" strokeLinejoin="round"/>
      <path d="M11 3v5h5M7 11h6M7 14h4" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

function IconCheck() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M5 8l2 2 4-4" strokeLinecap="round" strokeLinejoin="round"/>
      <circle cx="8" cy="8" r="6"/>
    </svg>
  )
}

function IconEdit() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M11 2.5l2.5 2.5M2 11.5L2 14h2.5L13.5 4.5l-2.5-2.5L2 11.5z" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

function IconAlert() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="8" r="7"/>
      <path d="M8 4v4M8 12h.01" strokeLinecap="round"/>
    </svg>
  )
}

function IconPlus() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M8 3v10M3 8h10" strokeLinecap="round"/>
    </svg>
  )
}

export default function Invoices() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [summary, setSummary] = useState<InvoicesSummary | null>(null)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const defaultFilters = useMemo(() => ({
    customerId: "", subscriptionId: "", status: "", number: "",
    periodStartFrom: "", periodStartTo: "", issuedFrom: "", issuedTo: "",
    createdFrom: "", createdTo: "", pageSize: 20
  }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const loadSummary = useCallback(async () => {
    try {
      setLoading(true); setError(null)
      const data = await api.invoices.summary()
      setSummary(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed_to_load")
    } finally { setLoading(false) }
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

  const searchSubscriptions = useCallback(async (query: string) => {
    const trimmed = query.trim()
    if (!trimmed) return []
    const resp = await api.subscriptions.list({ page_size: 50 })
    return resp.subscriptions
      .filter((sub) => sub.id.toLowerCase().includes(trimmed.toLowerCase()))
      .map((sub) => ({
        value: sub.id,
        label: `${sub.id.slice(0, 8)}… · ${sub.status}`
      }))
  }, [])

  const applyListResponse = useCallback((resp: InvoicesListResponse, reset: boolean) => {
    setInvoices((prev) => (reset ? resp.invoices : [...prev, ...resp.invoices]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadInvoices = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const token = reset ? undefined : nextTokenRef.current
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.invoices.list({
        page_token: token,
        page_size: f.pageSize,
        customer_id: f.customerId,
        subscription_id: f.subscriptionId,
        status: f.status,
        number: f.number,
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("invoices.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadSummary(); void loadInvoices(true) }, [loadInvoices, loadSummary])

  const columns = useMemo(() => [
    { key: "number", label: t("invoices.table.columns.invoice"), render: (r: Invoice) => (
      <div>
        <div style={{ fontWeight: 600 }}>{r.number}</div>
        <div className="muted">{shortID(r.id)}</div>
      </div>
    ) },
    { key: "status", label: t("invoices.table.columns.status"), width: "100px", render: (r: Invoice) => (
      <Badge className={`status-badge ${statusClass(r.status)}`}>{t(`invoices.status.${r.status}`, { defaultValue: r.status })}</Badge>
    ) },
    { key: "amount", label: t("invoices.table.columns.amount"), width: "110px", render: (r: Invoice) => (
      <div className="cell-mono" style={{ textAlign: "right" }}>
        {formatCurrency(r.amount_due_cents, r.currency)}
      </div>
    ) },
    { key: "customer", label: t("invoices.table.columns.customer"), width: "120px", render: (r: Invoice) => <span className="cell-mono">{shortID(r.customer_id)}</span> },
    { key: "period", label: t("invoices.table.columns.period"), width: "130px", render: (r: Invoice) => (
      <div>
        <span className="muted">{formatDate(r.period_start)}</span><br/>
        <span className="muted">{formatDate(r.period_end)}</span>
      </div>
    ) },
    { key: "actions", label: "", width: "100px", render: (r: Invoice) => (
      <div style={{ display: "flex", gap: "8px" }}>
        <Button variant="outline" size="sm" onClick={() => navigate(orgPath(`/invoices/${r.id}/manage`))} data-testid={`invoices-manage-${r.id}`}>{t("common.manage")}</Button>
      </div>
    ) },
  ], [navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader 
        icon={<IconInvoice />} 
        title={t("invoices.header.title")} 
        description={t("invoices.header.description")}
        actions={
          <Button variant="default" onClick={() => navigate(orgPath("/invoices/new"))} style={{ display: "flex", gap: 6, alignItems: "center" }}>
            <IconPlus /> {t("invoices.actions.generate")}
          </Button>
        }
      />

      {error ? <div className="inline-error">{error}</div> : null}

      <div className="stat-grid">
        <StatCard label={t("invoices.kpis.draft")} value={loading ? t("common.empty_dash") : formatNumber(summary?.draft ?? 0)} icon={<IconEdit />} accentColor="hsl(var(--text-muted))" />
        <StatCard label={t("invoices.kpis.open")} value={loading ? t("common.empty_dash") : formatNumber(summary?.open ?? 0)} icon={<IconAlert />} accentColor="hsl(var(--status-warning))" />
        <StatCard label={t("invoices.kpis.paid_amount")} value={loading ? t("common.empty_dash") : formatCurrency(summary?.paidCents ?? 0, "USD")} icon={<IconCheck />} accentColor="hsl(var(--status-success))" />
      </div>

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadInvoices(true)} data-testid="invoices-filters-apply">{t("common.apply_filters")}</Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => { setFilters(defaultFilters); setNextToken(undefined); void loadInvoices(true, defaultFilters) }} data-testid="invoices-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <AutoCompleteInput
              id="invoice-filter-customer"
              label={t("invoices.filters.customer")}
              value={filters.customerId}
              options={[]}
              placeholder={t("invoices.filters.customer_placeholder")}
              onSearch={searchCustomers}
              onChange={(value) => setFilters(p => ({ ...p, customerId: value }))}
            />
          </div>
          <div className="filter-field">
            <AutoCompleteInput
              id="invoice-filter-subscription"
              label={t("invoices.filters.subscription")}
              value={filters.subscriptionId}
              options={[]}
              placeholder={t("invoices.filters.subscription_placeholder")}
              onSearch={searchSubscriptions}
              onChange={(value) => setFilters(p => ({ ...p, subscriptionId: value }))}
            />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("invoices.filters.status")}</label>
            <Select
              value={toSelectValue(filters.status)}
              onValueChange={(value) => setFilters(p => ({ ...p, status: fromSelectValue(value) }))}
            >
              <SelectTrigger className="filter-select" data-testid="invoices-filter-status">
                <SelectValue placeholder={t("common.all")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.all")}</SelectItem>
                <SelectItem value="draft">{t("invoices.status.draft")}</SelectItem>
                <SelectItem value="open">{t("invoices.status.open")}</SelectItem>
                <SelectItem value="paid">{t("invoices.status.paid")}</SelectItem>
                <SelectItem value="void">{t("invoices.status.void")}</SelectItem>
                <SelectItem value="uncollectible">{t("invoices.status.uncollectible")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("invoices.filters.number")}</label>
            <Input className="filter-input" value={filters.number} onChange={(e) => setFilters(p => ({ ...p, number: e.target.value }))} data-testid="invoices-filter-number" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize} onChange={(e) => setFilters(p => ({ ...p, pageSize: Number(e.target.value||"20") }))} data-testid="invoices-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={invoices}
        loading={listLoading && invoices.length === 0}
        emptyTitle={t("invoices.table.empty_title")}
        emptyDesc={t("invoices.table.empty_desc")}
        footer={hasMore ? (
          <Button variant="secondary" size="sm" disabled={listLoading} onClick={() => loadInvoices(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
