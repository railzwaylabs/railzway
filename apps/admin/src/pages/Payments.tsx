import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { api } from "../lib/api"
import { formatCurrency, formatNumber } from "../lib/format"
import { statusClass } from "../lib/status"
import type { Payment, PaymentsListResponse, PaymentsSummary } from "../lib/types"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import { toast } from "../components/Toast"
import { formatDate, normalizeDate, rfc3339Hint, shortID } from "../lib/display"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"
import { Badge } from "../components/ui/badge"

function IconPayment() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1.5" y="4" width="17" height="12" rx="2"/>
      <path d="M1.5 8h17" strokeLinecap="round"/>
      <path d="M5 12.5h3" strokeLinecap="round" strokeWidth="2"/>
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

function IconAlert() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="8" r="7"/>
      <path d="M8 4v4M8 12h.01" strokeLinecap="round"/>
    </svg>
  )
}

function IconRefresh() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M13.5 4.5L11 2M13.5 4.5L11 7" strokeLinecap="round" strokeLinejoin="round"/>
      <path d="M2.5 11.5L5 9M2.5 11.5L5 14" strokeLinecap="round" strokeLinejoin="round"/>
      <path d="M3.5 5A4.5 4.5 0 0113.5 4.5M12.5 11A4.5 4.5 0 012.5 11.5" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function Payments() {
  const { t } = useTranslation()
  const [summary, setSummary] = useState<PaymentsSummary | null>(null)
  const [payments, setPayments] = useState<Payment[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const defaultFilters = useMemo(() => ({
    customerId: "", invoiceId: "", status: "", provider: "", createdFrom: "", createdTo: "", pageSize: 20,
  }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const loadSummary = useCallback(async () => {
    try { setLoading(true); const d = await api.payments.summary(); setSummary(d) }
    catch { /* ignore */ } finally { setLoading(false) }
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

  const applyListResponse = useCallback((resp: PaymentsListResponse, reset: boolean) => {
    setPayments((prev) => (reset ? resp.payments : [...prev, ...resp.payments]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadPayments = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.payments.list({
        page_token: reset ? undefined : nextTokenRef.current, page_size: f.pageSize,
        customer_id: f.customerId, invoice_id: f.invoiceId, status: f.status, provider: f.provider,
        created_from: normalizeDate(f.createdFrom), created_to: normalizeDate(f.createdTo),
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("payments.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadSummary(); void loadPayments(true) }, [loadPayments, loadSummary])

  const columns = useMemo(() => [
    { key: "id", label: t("payments.table.columns.payment_id"), render: (r: Payment) => <span className="cell-mono">{shortID(r.id)}</span> },
    { key: "customer_id", label: t("payments.table.columns.customer"), render: (r: Payment) => <span className="cell-mono">{shortID(r.customer_id)}</span> },
    { key: "provider", label: t("payments.table.columns.provider"), width: "110px", render: (r: Payment) => <span className="muted">{r.provider || t("common.empty_dash")}</span> },
    { key: "amount_cents", label: t("payments.table.columns.amount"), width: "120px",
      render: (r: Payment) => (
        <span className="cell-mono">{formatCurrency(r.amount_cents, r.currency)}</span>
      ) },
    { key: "status", label: t("payments.table.columns.status"), width: "120px",
      render: (r: Payment) => <Badge className={`status-badge ${statusClass(r.status)}`}>{t(`payments.status.${r.status}`, { defaultValue: r.status })}</Badge> },
    { key: "created_at", label: t("payments.table.columns.created"), width: "130px",
      render: (r: Payment) => <span className="muted">{formatDate(r.created_at)}</span> },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader icon={<IconPayment />} title={t("payments.header.title")} description={t("payments.header.description")} />

      <div className="stat-grid">
        <StatCard label={t("payments.kpis.collected")} value={loading ? t("common.empty_dash") : formatCurrency(summary?.collectedCents ?? 0)} icon={<IconCheck />} accentColor="hsl(var(--status-success))" />
        <StatCard label={t("payments.kpis.failed")} value={loading ? t("common.empty_dash") : formatNumber(summary?.failed ?? 0)} icon={<IconAlert />} accentColor="hsl(var(--status-error))" />
        <StatCard label={t("payments.kpis.retries")} value={loading ? t("common.empty_dash") : formatNumber(summary?.retries ?? 0)} icon={<IconRefresh />} accentColor="hsl(var(--status-warning))" />
      </div>

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadPayments(true)} data-testid="payments-filters-apply">
              {listLoading ? t("common.searching") : t("common.apply_filters")}
            </Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadPayments(true, defaultFilters)
            }} data-testid="payments-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <AutoCompleteInput
              id="payments-filter-customer"
              label={t("payments.filters.customer")}
              value={filters.customerId}
              options={[]}
              placeholder={t("payments.filters.customer_placeholder")}
              onSearch={searchCustomers}
              onChange={(value) => setFilters((p) => ({ ...p, customerId: value }))}
            />
          </div>
          <div className="filter-field">
            <AutoCompleteInput
              id="payments-filter-invoice"
              label={t("payments.filters.invoice")}
              value={filters.invoiceId}
              options={[]}
              placeholder={t("payments.filters.invoice_placeholder")}
              onSearch={searchInvoices}
              onChange={(value) => setFilters((p) => ({ ...p, invoiceId: value }))}
            />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("payments.filters.status")}</label>
            <Select
              value={toSelectValue(filters.status)}
              onValueChange={(value) => setFilters((p) => ({ ...p, status: fromSelectValue(value) }))}
            >
              <SelectTrigger className="filter-select" data-testid="payments-filter-status">
                <SelectValue placeholder={t("common.all")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.all")}</SelectItem>
                <SelectItem value="pending">{t("payments.status.pending")}</SelectItem>
                <SelectItem value="succeeded">{t("payments.status.succeeded")}</SelectItem>
                <SelectItem value="failed">{t("payments.status.failed")}</SelectItem>
                <SelectItem value="refunded">{t("payments.status.refunded")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("payments.filters.provider")}</label>
            <Input className="filter-input" value={filters.provider}
              onChange={(e) => setFilters((p) => ({ ...p, provider: e.target.value }))} data-testid="payments-filter-provider" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("payments.filters.created_from")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" value={filters.createdFrom}
              onChange={(e) => setFilters((p) => ({ ...p, createdFrom: e.target.value }))} data-testid="payments-filter-created-from" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("payments.filters.created_to")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" min={filters.createdFrom || undefined} value={filters.createdTo}
              onChange={(e) => setFilters((p) => ({ ...p, createdTo: e.target.value }))} data-testid="payments-filter-created-to" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="payments-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={payments}
        loading={listLoading && payments.length === 0}
        emptyTitle={t("payments.table.empty_title")}
        emptyDesc={t("payments.table.empty_desc")}
        footer={hasMore ? (
          <Button variant="secondary" size="sm" disabled={listLoading} onClick={() => loadPayments(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
