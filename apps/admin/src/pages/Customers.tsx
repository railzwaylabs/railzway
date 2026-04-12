import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { formatDate, normalizeDate, rfc3339Hint, shortID } from "../lib/display"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { api } from "../lib/api"
import { useCurrencies } from "../lib/reference"
import { useOrgPath } from "../lib/org"
import { formatNumber } from "../lib/format"
import { currencyHint } from "../lib/hints"
import type { Customer, CustomersListResponse, CustomersSummary } from "../lib/types"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import { toast } from "../components/Toast"

function IconUsers() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="7" cy="6" r="3" />
      <path d="M1 17c0-3.866 2.686-6 6-6s6 2.134 6 6" strokeLinecap="round" />
      <path d="M14 10c2 .4 4 1.6 4 4" strokeLinecap="round" />
      <circle cx="16" cy="6" r="2" />
    </svg>
  )
}

function IconActive() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="8" r="6" />
      <path d="M5 8l2 2 4-4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function IconRisk() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M8 2L14.5 13H1.5L8 2z" strokeLinejoin="round" />
      <path d="M8 6v3M8 11v.5" strokeLinecap="round" />
    </svg>
  )
}

function IconNRR() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 11L6 7l3 2 5-5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function IconPlus() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M8 3v10M3 8h10" strokeLinecap="round" />
    </svg>
  )
}


export default function Customers() {
  const { t } = useTranslation()
  const orgPath = useOrgPath()
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies()
  const [summary, setSummary] = useState<CustomersSummary | null>(null)
  const [customers, setCustomers] = useState<Customer[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const defaultFilters = useMemo(() => ({
    name: "", email: "", currency: "", createdFrom: "", createdTo: "", pageSize: 20
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
      const data = await api.customers.summary()
      setSummary(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed_to_load")
    } finally { setLoading(false) }
  }, [])

  const applyListResponse = useCallback((resp: CustomersListResponse, reset: boolean) => {
    setCustomers((prev) => (reset ? resp.customers : [...prev, ...resp.customers]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadCustomers = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const token = reset ? undefined : nextTokenRef.current
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.customers.list({
        page_token: token, page_size: f.pageSize,
        name: f.name, email: f.email, currency: f.currency,
        created_from: normalizeDate(f.createdFrom), created_to: normalizeDate(f.createdTo),
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("customers.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadSummary(); void loadCustomers(true) }, [loadCustomers, loadSummary])

  const nrrLabel = loading ? t("common.loading_dash") : `${(summary?.nrr_pct ?? 0).toFixed(1)}%`

  const columns = useMemo(() => [
    {
      key: "name", label: t("customers.table.columns.customer"),
      render: (row: Customer) => (
        <div>
          <div style={{ fontWeight: 600 }}>{row.name}</div>
          <div className="muted">{row.email}</div>
        </div>
      ),
    },
    {
      key: "currency", label: t("customers.table.columns.currency"), width: "100px",
      render: (row: Customer) => row.currency ? <span className="cell-mono">{row.currency}</span> : <span className="muted">{t("common.empty_dash")}</span>
    },
    {
      key: "external_id", label: t("customers.table.columns.external_id"), width: "120px",
      render: (row: Customer) => <span className="cell-mono">{shortID(row.external_id)}</span>
    },
    {
      key: "created_at", label: t("customers.table.columns.created"), width: "130px",
      render: (row: Customer) => <span className="muted">{formatDate(row.created_at)}</span>
    },
    {
      key: "actions", label: "", width: "80px", className: "col-actions",
      render: (row: Customer) => (
        <Button asChild variant="secondary" size="sm">
          <Link to={orgPath(`/customers/${row.id}/edit`)} data-testid={`customers-edit-${row.id}`}>{t("common.edit")}</Link>
        </Button>
      ),
    },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconUsers />}
        title={t("customers.header.title")}
        description={t("customers.header.description")}
        actions={
          <Button asChild>
            <Link to={orgPath("/customers/new")} style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <IconPlus /> {t("customers.actions.new")}
            </Link>
          </Button>
        }
      />

      {error ? <div className="inline-error" style={{ padding: "12px 16px", background: "var(--status-danger-bg)", borderRadius: 12 }}>{error}</div> : null}

      {/* KPI Cards */}
      <div className="stat-grid">
        <StatCard
          label={t("customers.kpis.active.label")}
          value={loading ? t("common.empty_dash") : formatNumber(summary?.active ?? 0)}
          icon={<IconActive />}
          accentColor="hsl(var(--status-success))"
        />
        <StatCard
          label={t("customers.kpis.at_risk.label")}
          value={loading ? t("common.empty_dash") : formatNumber(summary?.at_risk ?? 0)}
          icon={<IconRisk />}
          accentColor="hsl(var(--status-error))"
        />
        <StatCard
          label={t("customers.kpis.nrr.label")}
          value={nrrLabel}
          icon={<IconNRR />}
          sub={t("customers.kpis.nrr.sub")}
          accentColor="hsl(var(--accent-primary))"
        />
      </div>

      {/* Filters */}
      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadCustomers(true)} data-testid="customers-filters-apply">
              {listLoading ? t("common.searching") : t("common.apply_filters")}
            </Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadCustomers(true, defaultFilters)
            }} data-testid="customers-filters-reset">
              {t("common.reset")}
            </Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <label className="filter-label">{t("customers.filters.name")}</label>
            <Input className="filter-input" value={filters.name}
              onChange={(e) => setFilters((p) => ({ ...p, name: e.target.value }))} data-testid="customers-filter-name" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("customers.filters.email")}</label>
            <Input className="filter-input" value={filters.email}
              onChange={(e) => setFilters((p) => ({ ...p, email: e.target.value }))} data-testid="customers-filter-email" />
          </div>
          <div className="filter-field">
            <AutoCompleteInput
              id="customers-filter-currency"
              label={<>{t("customers.filters.currency")} <HelpHint text={currencyHint} /></>}
              value={filters.currency}
              options={currencyOptions}
              placeholder={currenciesLoading ? t("common.loading") : t("customers.filters.currency_placeholder")}
              onChange={(value) => setFilters((p) => ({ ...p, currency: value }))}
            />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("customers.filters.created_from")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" value={filters.createdFrom}
              onChange={(e) => setFilters((p) => ({ ...p, createdFrom: e.target.value }))}
              data-testid="customers-filter-created-from" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("customers.filters.created_to")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" min={filters.createdFrom || undefined} value={filters.createdTo}
              onChange={(e) => setFilters((p) => ({ ...p, createdTo: e.target.value }))}
              data-testid="customers-filter-created-to" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="customers-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      {/* Table */}
      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={customers}
        loading={listLoading && customers.length === 0}
        emptyTitle={t("customers.table.empty_title")}
        emptyDesc={t("customers.table.empty_desc")}
        footer={hasMore ? (
          <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => loadCustomers(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
