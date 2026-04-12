import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { formatNumber } from "../lib/format"
import type { TaxRate, TaxRatesListResponse, TaxesSummary } from "../lib/types"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import { toast } from "../components/Toast"
import { Badge } from "../components/ui/badge"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"
import { formatDate, rfc3339Hint } from "../lib/display"
import { statusClass } from "../lib/status"

function IconTax() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M5 15L15 5M6.5 5.5h-3v3M16.5 14.5h-3v3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}
function IconProfiles() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="2" width="12" height="12" rx="2"/>
      <path d="M5 7h6M5 10h4" strokeLinecap="round"/>
    </svg>
  )
}
function IconExempt() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="8" cy="8" r="6"/>
      <path d="M5 8l2 2 4-4" strokeLinecap="round" strokeLinejoin="round"/>
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


const normalizeDate = (value: string) => {
  if (!value) return ""
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return parsed.toISOString()
}

export default function Taxes() {
  const { t } = useTranslation()
  const orgPath = useOrgPath()
  const [summary, setSummary] = useState<TaxesSummary | null>(null)
  const [rates, setRates] = useState<TaxRate[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const defaultFilters = useMemo(() => ({
    code: "", name: "", active: "", createdFrom: "", createdTo: "", pageSize: 20,
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
      const data = await api.taxes.summary()
      setSummary(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed_to_load")
    } finally { setLoading(false) }
  }, [])

  const applyListResponse = useCallback((resp: TaxRatesListResponse, reset: boolean) => {
    setRates((prev) => (reset ? resp.rates : [...prev, ...resp.rates]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadRates = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const token = reset ? undefined : nextTokenRef.current
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.taxes.list({
        page_token: token, page_size: f.pageSize,
        code: f.code, name: f.name, active: f.active,
        created_from: normalizeDate(f.createdFrom), created_to: normalizeDate(f.createdTo),
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("taxes.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadSummary(); void loadRates(true) }, [loadRates, loadSummary])

  const columns = useMemo(() => [
    { key: "name", label: t("taxes.table.columns.name"),
      render: (r: TaxRate) => <div style={{ fontWeight: 600 }}>{r.name}</div> },
    { key: "code", label: t("taxes.table.columns.code"), width: "110px",
      render: (r: TaxRate) => <span className="cell-mono">{r.code}</span> },
    { key: "percentage", label: t("taxes.table.columns.rate"), width: "90px",
      render: (r: TaxRate) => <strong>{r.percentage}%</strong> },
    { key: "inclusive", label: t("taxes.table.columns.type"), width: "110px",
      render: (r: TaxRate) => (
        <Badge variant="secondary" className={r.inclusive ? "badge-info" : "badge-muted"}>
          {r.inclusive ? t("taxes.table.type.inclusive") : t("taxes.table.type.exclusive")}
        </Badge>
      ),
    },
    { key: "active", label: t("taxes.table.columns.status"), width: "100px",
      render: (r: TaxRate) => (
        <span className={`badge ${r.active ? "badge-success" : "badge-muted"}`}>
          {r.active ? t("taxes.table.status.active") : t("taxes.table.status.inactive")}
        </span>
      ),
    },
    { key: "created_at", label: t("taxes.table.columns.created"), width: "130px",
      render: (r: TaxRate) => <span className="muted">{formatDate(r.created_at)}</span> },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconTax />}
        title={t("taxes.header.title")}
        description={t("taxes.header.description")}
        actions={(
          <Button asChild>
            <Link to={orgPath("/taxes/new")} style={{ display: "flex", alignItems: "center", gap: 6 }} data-testid="taxes-create-nav">
              <IconPlus /> {t("taxes.actions.new")}
            </Link>
          </Button>
        )}
      />

      {error ? <div className="inline-error" style={{ padding: "12px 16px", background: "var(--status-danger-bg)", borderRadius: 12 }}>{error}</div> : null}

      <div className="stat-grid">
        <StatCard
          label={t("taxes.kpis.active_profiles")}
          value={loading ? t("common.empty_dash") : formatNumber(summary?.profiles ?? 0)}
          icon={<IconProfiles />}
          accentColor="hsl(var(--accent-primary))"
        />
        <StatCard
          label={t("taxes.kpis.exempt_customers")}
          value={loading ? t("common.empty_dash") : formatNumber(summary?.exemptCustomers ?? 0)}
          icon={<IconExempt />}
          accentColor="hsl(var(--status-success))"
        />
      </div>

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadRates(true)} data-testid="taxes-filters-apply">
              {listLoading ? t("common.searching") : t("common.apply_filters")}
            </Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadRates(true, defaultFilters)
            }} data-testid="taxes-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <label className="filter-label">{t("taxes.filters.code")}</label>
            <Input className="filter-input" value={filters.code}
              onChange={(e) => setFilters((p) => ({ ...p, code: e.target.value }))} data-testid="taxes-filter-code" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("taxes.filters.name")}</label>
            <Input className="filter-input" value={filters.name}
              onChange={(e) => setFilters((p) => ({ ...p, name: e.target.value }))} data-testid="taxes-filter-name" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("taxes.filters.status")}</label>
            <Select
              value={toSelectValue(filters.active)}
              onValueChange={(value) => setFilters((p) => ({ ...p, active: fromSelectValue(value) }))}
            >
              <SelectTrigger className="filter-select" data-testid="taxes-filter-active">
                <SelectValue placeholder={t("common.all")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.all")}</SelectItem>
                <SelectItem value="true">{t("taxes.table.status.active")}</SelectItem>
                <SelectItem value="false">{t("taxes.table.status.inactive")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("taxes.filters.created_from")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" value={filters.createdFrom}
              onChange={(e) => setFilters((p) => ({ ...p, createdFrom: e.target.value }))}
              data-testid="taxes-filter-created-from" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("taxes.filters.created_to")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" min={filters.createdFrom || undefined} value={filters.createdTo}
              onChange={(e) => setFilters((p) => ({ ...p, createdTo: e.target.value }))}
              data-testid="taxes-filter-created-to" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="taxes-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={rates}
        loading={listLoading && rates.length === 0}
        emptyTitle={t("taxes.table.empty_title")}
        emptyDesc={t("taxes.table.empty_desc")}
        footer={hasMore ? (
          <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => loadRates(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
