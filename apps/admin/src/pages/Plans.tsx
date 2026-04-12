import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { Badge } from "../components/ui/badge"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { formatCurrency, formatNumber } from "../lib/format"
import { statusClass } from "../lib/status"
import { formatDate, shortID } from "../lib/display"
import type { Plan, PlansListResponse, PlansSummary } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import FilterPanel from "../components/FilterPanel"
import { toast } from "../components/Toast"

function IconPlans() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M4 6h12M4 10h12M4 14h8" strokeLinecap="round" strokeLinejoin="round"/>
      <path d="M3 3h14c.552 0 1 .448 1 1v12c0 .552-.448 1-1 1H3c-.552 0-1-.448-1-1V4c0-.552.448-1 1-1z" strokeLinecap="round" strokeLinejoin="round"/>
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

function IconLayers() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M1 5l7 3 7-3-7-3-7 3z" strokeLinecap="round" strokeLinejoin="round"/>
      <path d="M1 9l7 3 7-3" strokeLinecap="round" strokeLinejoin="round"/>
      <path d="M1 13l7 3 7-3" strokeLinecap="round" strokeLinejoin="round"/>
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

export default function Plans() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [summary, setSummary] = useState<PlansSummary | null>(null)
  const [plans, setPlans] = useState<Plan[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const defaultFilters = useMemo(() => ({ code: "", name: "", active: "", pageSize: 20 }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const loadSummary = useCallback(async () => {
    try {
      setLoading(true); setError(null)
      const data = await api.plans.summary()
      setSummary(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed_to_load")
    } finally { setLoading(false) }
  }, [])

  const applyListResponse = useCallback((resp: PlansListResponse, reset: boolean) => {
    setPlans((prev) => (reset ? resp.plans : [...prev, ...resp.plans]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadPlans = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const token = reset ? undefined : nextTokenRef.current
      const activeFilters = overrideFilters ?? filtersRef.current
      const resp = await api.plans.list({
        page_token: token,
        page_size: activeFilters.pageSize,
        code: activeFilters.code,
        name: activeFilters.name,
        active: activeFilters.active
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("plans.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadSummary(); void loadPlans(true) }, [loadPlans, loadSummary])

  const columns = useMemo(() => [
    { key: "name", label: t("plans.table.columns.plan"), render: (r: Plan) => (
      <div>
        <div style={{ fontWeight: 600 }}>{r.name}</div>
        <div className="muted">{r.code} • {shortID(r.id)}</div>
      </div>
    ) },
    { key: "description", label: t("plans.table.columns.description"), width: "240px", render: (r: Plan) => r.description || <span className="muted">{t("common.empty_dash")}</span> },
    { key: "pricing", label: t("plans.table.columns.pricing"), width: "280px", render: (r: Plan) => {
      if (!r.prices || r.prices.length === 0) {
        return <span className="muted">{t("plans.table.no_prices")}</span>
      }
      const visible = r.prices.slice(0, 2)
      return (
        <div style={{ display: "grid", gap: 8 }}>
          {visible.map((price) => {
            const interval = price.billing_interval_count > 1
              ? `${price.billing_interval_count} ${price.billing_interval}s`
              : price.billing_interval
            const amounts = price.amounts ?? []
            const tiers = price.tiers ?? []
            const amountLabel = amounts.length > 0
              ? `${formatCurrency(amounts[0].unit_amount_cents, amounts[0].currency)}${amounts.length > 1 ? ` +${amounts.length - 1}` : ""}`
              : t("plans.table.no_amount")
            const tierLabel = tiers.length > 0 ? t("plans.table.tiers", { count: tiers.length }) : ""

            return (
              <div key={price.id} style={{ display: "grid", gap: 4 }}>
                <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
                  <span style={{ fontWeight: 600 }}>{price.name || price.code}</span>
                  <Badge className={`status-badge ${statusClass(price.active ? "active" : "inactive")}`}>
                    {price.price_type}
                  </Badge>
                </div>
                <div className="muted" style={{ fontSize: 12 }}>
                  {amountLabel} • {interval}{tierLabel ? ` • ${tierLabel}` : ""}
                </div>
              </div>
            )
          })}
          {r.prices.length > visible.length ? (
            <span className="muted" style={{ fontSize: 12 }}>{t("plans.table.more_prices", { count: r.prices.length - visible.length })}</span>
          ) : null}
        </div>
      )
    }},
    { key: "status", label: t("plans.table.columns.status"), width: "100px", render: (r: Plan) => (
      <Badge className={`status-badge ${statusClass(r.active ? "active" : "inactive")}`}>
        {r.active ? t("plans.table.status.active") : t("plans.table.status.inactive")}
      </Badge>
    ) },
    { key: "created_at", label: t("plans.table.columns.created"), width: "130px", render: (r: Plan) => <span className="muted">{formatDate(r.created_at)}</span> },
    { key: "actions", label: "", width: "80px", render: (r: Plan) => (
      <Button variant="secondary" size="sm" onClick={() => navigate(orgPath(`/plans/${r.id}/edit`))} data-testid={`plans-edit-${r.id}`}>{t("common.edit")}</Button>
    ) },
  ], [navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader 
        icon={<IconPlans />} 
        title={t("plans.header.title")}
        description={t("plans.header.description")}
        actions={
          <Button variant="default" onClick={() => navigate(orgPath("/plans/new"))} style={{ display: "flex", gap: 6, alignItems: "center" }}>
            <IconPlus /> {t("plans.actions.new")}
          </Button>
        }
      />

      {error ? <div className="inline-error">{error}</div> : null}

      <div className="stat-grid">
        <StatCard label={t("plans.kpis.active")} value={loading ? t("common.empty_dash") : formatNumber(summary?.active ?? 0)} icon={<IconCheck />} accentColor="hsl(var(--status-success))" />
        <StatCard label={t("plans.kpis.draft")} value={loading ? t("common.empty_dash") : formatNumber(summary?.draft ?? 0)} icon={<IconEdit />} accentColor="hsl(var(--status-warning))" />
        <StatCard label={t("plans.kpis.tiered")} value={loading ? t("common.empty_dash") : formatNumber(summary?.tiered ?? 0)} icon={<IconLayers />} accentColor="var(--accent-strong)" />
      </div>

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadPlans(true)} data-testid="plans-filters-apply">{t("common.apply_filters")}</Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => { setFilters(defaultFilters); setNextToken(undefined); void loadPlans(true, defaultFilters) }} data-testid="plans-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <label className="filter-label">{t("plans.filters.code")}</label>
            <Input className="filter-input" value={filters.code} onChange={(e) => setFilters(p => ({ ...p, code: e.target.value }))} data-testid="plans-filter-code" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("plans.filters.name")}</label>
             <Input className="filter-input" value={filters.name} onChange={(e) => setFilters(p => ({ ...p, name: e.target.value }))} data-testid="plans-filter-name" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("plans.filters.active_status")}</label>
            <Select
              value={toSelectValue(filters.active)}
              onValueChange={(value) => setFilters(p => ({ ...p, active: fromSelectValue(value) }))}
            >
              <SelectTrigger className="filter-select" data-testid="plans-filter-active">
                <SelectValue placeholder={t("common.all")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.all")}</SelectItem>
                <SelectItem value="true">{t("plans.table.status.active")}</SelectItem>
                <SelectItem value="false">{t("plans.table.status.inactive")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize} onChange={(e) => setFilters(p => ({ ...p, pageSize: Number(e.target.value||"20") }))} data-testid="plans-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={plans}
        loading={listLoading && plans.length === 0}
        emptyTitle={t("plans.table.empty_title")}
        emptyDesc={t("plans.table.empty_desc")}
        footer={hasMore ? (
          <Button variant="secondary" size="sm" disabled={listLoading} onClick={() => loadPlans(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
