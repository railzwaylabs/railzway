import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import type { LedgerTransaction, LedgerTransactionsResponse } from "../lib/types"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { formatDate, normalizeDate, rfc3339Hint } from "../lib/display"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"

function IconLedger() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="2" width="16" height="16" rx="2"/>
      <path d="M6 7h8M6 10h8M6 13h5" strokeLinecap="round"/>
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

export default function Ledger() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [transactions, setTransactions] = useState<LedgerTransaction[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [listLoading, setListLoading] = useState(false)
  const defaultFilters = useMemo(() => ({
    sourceType: "", sourceId: "", customerId: "", subscriptionId: "", invoiceId: "",
    occurredFrom: "", occurredTo: "", pageSize: 20,
  }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const applyListResponse = useCallback((resp: LedgerTransactionsResponse, reset: boolean) => {
    setTransactions((prev) => (reset ? resp.transactions : [...prev, ...resp.transactions]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
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

  const searchInvoices = useCallback(async (query: string) => {
    const trimmed = query.trim()
    if (!trimmed) return []
    const resp = await api.invoices.list({ page_size: 50, number: trimmed })
    return resp.invoices.map((invoice) => ({
      value: invoice.id,
      label: `${invoice.number} · ${invoice.status}`
    }))
  }, [])

  const loadTransactions = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.ledger.listTransactions({
        page_token: reset ? undefined : nextTokenRef.current, page_size: f.pageSize,
        source_type: f.sourceType, source_id: f.sourceId,
        customer_id: f.customerId, subscription_id: f.subscriptionId, invoice_id: f.invoiceId,
        occurred_from: normalizeDate(f.occurredFrom), occurred_to: normalizeDate(f.occurredTo),
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("ledger.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadTransactions(true) }, [loadTransactions])

  const txColumns = useMemo(() => [
    { key: "source_type", label: t("ledger.table.columns.source_type"), render: (r: LedgerTransaction) => <strong>{r.source_type}</strong> },
    { key: "source_id", label: t("ledger.table.columns.source_id"), render: (r: LedgerTransaction) => <span className="cell-mono">{r.source_id ? r.source_id.slice(0, 12) + "…" : t("common.empty_dash")}</span> },
    { key: "currency", label: t("ledger.table.columns.currency"), width: "100px", render: (r: LedgerTransaction) => <span className="cell-mono">{r.currency}</span> },
    { key: "customer_id", label: t("ledger.table.columns.customer"), render: (r: LedgerTransaction) => <span className="cell-mono">{r.customer_id ? `${r.customer_id.slice(0, 8)}…` : t("common.empty_dash")}</span> },
    { key: "occurred_at", label: t("ledger.table.columns.occurred_at"), width: "135px", render: (r: LedgerTransaction) => <span className="muted">{formatDate(r.occurred_at)}</span> },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader 
        icon={<IconLedger />} 
        title={t("ledger.header.title")} 
        description={t("ledger.header.description")} 
        actions={
          <Button variant="default" onClick={() => navigate(orgPath("/ledger/new"))} style={{ display: "flex", gap: 6, alignItems: "center" }} data-testid="ledger-new-button">
            <IconPlus /> {t("ledger.actions.new")}
          </Button>
        }
      />

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadTransactions(true)} data-testid="ledger-filters-apply">{t("common.apply_filters")}</Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadTransactions(true, defaultFilters)
            }} data-testid="ledger-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field"><label className="filter-label">{t("ledger.filters.source_type")}</label>
            <Input className="filter-input" value={filters.sourceType} onChange={(e) => setFilters((p) => ({ ...p, sourceType: e.target.value }))} data-testid="ledger-filter-source-type" /></div>
          <div className="filter-field"><label className="filter-label">{t("ledger.filters.source_id")}</label>
            <Input className="filter-input" value={filters.sourceId} onChange={(e) => setFilters((p) => ({ ...p, sourceId: e.target.value }))} data-testid="ledger-filter-source-id" /></div>
          <div className="filter-field">
            <AutoCompleteInput
              id="ledger-filter-customer"
              label={t("ledger.filters.customer")}
              value={filters.customerId}
              options={[]}
              placeholder={t("ledger.filters.customer_placeholder")}
              onSearch={searchCustomers}
              onChange={(value) => setFilters((p) => ({ ...p, customerId: value }))}
            />
          </div>
          <div className="filter-field">
            <AutoCompleteInput
              id="ledger-filter-subscription"
              label={t("ledger.filters.subscription")}
              value={filters.subscriptionId}
              options={[]}
              placeholder={t("ledger.filters.subscription_placeholder")}
              onSearch={searchSubscriptions}
              onChange={(value) => setFilters((p) => ({ ...p, subscriptionId: value }))}
            />
          </div>
          <div className="filter-field">
            <AutoCompleteInput
              id="ledger-filter-invoice"
              label={t("ledger.filters.invoice")}
              value={filters.invoiceId}
              options={[]}
              placeholder={t("ledger.filters.invoice_placeholder")}
              onSearch={searchInvoices}
              onChange={(value) => setFilters((p) => ({ ...p, invoiceId: value }))}
            />
          </div>
          <div className="filter-field"><label className="filter-label">{t("ledger.filters.occurred_from")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="datetime-local" value={filters.occurredFrom} onChange={(e) => setFilters((p) => ({ ...p, occurredFrom: e.target.value }))} data-testid="ledger-filter-occurred-from" /></div>
          <div className="filter-field"><label className="filter-label">{t("ledger.filters.occurred_to")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="datetime-local" min={filters.occurredFrom || undefined} value={filters.occurredTo} onChange={(e) => setFilters((p) => ({ ...p, occurredTo: e.target.value }))} data-testid="ledger-filter-occurred-to" /></div>
          <div className="filter-field"><label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="ledger-filter-page-size" /></div>
        </div>
      </FilterPanel>

      <DataTable
        columns={txColumns as Parameters<typeof DataTable>[0]["columns"]}
        data={transactions}
        loading={listLoading && transactions.length === 0}
        emptyTitle={t("ledger.table.empty_title")}
        emptyDesc={t("ledger.table.empty_desc")}
        footer={hasMore ? (
          <Button variant="secondary" size="sm" disabled={listLoading} onClick={() => loadTransactions(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
