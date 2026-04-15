import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { api } from "../lib/api"
import { formatNumber } from "../lib/format"
import { statusClass } from "../lib/status"
import { useOrgPath } from "../lib/org"
import type { UsageEvent, UsageEventsResponse, UsageSummary } from "../lib/types"
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

function IconUsage() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M3 14L7 9l4 3 4-8 4 4" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

function IconUpload() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M8 12V3m0 0L4.5 6.5M8 3l3.5 3.5" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function Usage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [summary, setSummary] = useState<UsageSummary | null>(null)
  const [events, setEvents] = useState<UsageEvent[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const defaultFilters = useMemo(() => ({
    meterId: "", customerId: "", status: "", recordedFrom: "", recordedTo: "", pageSize: 20,
  }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const loadSummary = useCallback(async () => {
    try { setLoading(true); const d = await api.usage.summary(); setSummary(d) }
    catch { /* ignore */ } finally { setLoading(false) }
  }, [])

  const searchMeters = useCallback(async (query: string) => {
    const resp = await api.meters.list({ page_size: 50, active: "true", name: query })
    let meters = resp.meters
    if (meters.length === 0) {
      const fallback = await api.meters.list({ page_size: 50, active: "true", code: query })
      meters = fallback.meters
    }
    return meters.map((meter) => ({
      value: meter.id,
      label: `${meter.name} · ${meter.code}`
    }))
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

  const applyListResponse = useCallback((resp: UsageEventsResponse, reset: boolean) => {
    setEvents((prev) => (reset ? resp.events : [...prev, ...resp.events]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadEvents = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.usage.events({
        page_token: reset ? undefined : nextTokenRef.current, page_size: f.pageSize,
        meter_id: f.meterId, customer_id: f.customerId, status: f.status,
        recorded_from: normalizeDate(f.recordedFrom), recorded_to: normalizeDate(f.recordedTo),
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("usage.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadSummary(); void loadEvents(true) }, [loadEvents, loadSummary])

  const columns = useMemo(() => [
    { key: "id", label: t("usage.table.columns.event_id"), render: (r: UsageEvent) => <span className="cell-mono">{shortID(r.id)}</span> },
    { key: "meter_code", label: t("usage.table.columns.meter"), render: (r: UsageEvent) => <span className="cell-mono">{r.meter_code}</span> },
    { key: "customer_id", label: t("usage.table.columns.customer"), render: (r: UsageEvent) => <span className="cell-mono">{shortID(r.customer_id)}</span> },
    { key: "value", label: t("usage.table.columns.value"), width: "90px", render: (r: UsageEvent) => <strong>{r.value}</strong> },
    { key: "status", label: t("usage.table.columns.status"), width: "110px",
      render: (r: UsageEvent) => <Badge className={`status-badge ${statusClass(r.status)}`}>{t(`usage.status.${r.status}`, { defaultValue: r.status })}</Badge> },
    { key: "recorded_at", label: t("usage.table.columns.recorded_at"), width: "135px",
      render: (r: UsageEvent) => <span className="muted">{formatDate(r.recorded_at)}</span> },
  ], [t])

  const latePct = loading ? t("common.empty_dash") : `${(summary?.latePct ?? 0).toFixed(1)}%`

  return (
    <div className="page-content">
      <PageHeader 
        icon={<IconUsage />} 
        title={t("usage.header.title")} 
        description={t("usage.header.description")} 
        actions={
          <Button variant="default" onClick={() => navigate(orgPath("/usage/new"))} style={{ display: "flex", gap: 6, alignItems: "center" }} data-testid="usage-ingest-button">
            <IconUpload /> {t("usage.actions.ingest")}
          </Button>
        }
      />

      <div className="stat-grid">
        <StatCard label={t("usage.kpis.events_per_hr")} value={loading ? t("common.empty_dash") : formatNumber(summary?.eventsPerHour ?? 0)} accentColor="hsl(var(--accent-primary))" />
        <StatCard label={t("usage.kpis.late_events")} value={latePct} accentColor="hsl(var(--status-error))" />
        <StatCard label={t("usage.kpis.active_meters")} value={loading ? t("common.empty_dash") : formatNumber(summary?.activeMeters ?? 0)} accentColor="hsl(var(--status-success))" />
      </div>

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadEvents(true)} data-testid="usage-filters-apply">
              {listLoading ? t("common.searching") : t("common.apply_filters")}
            </Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadEvents(true, defaultFilters)
            }} data-testid="usage-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <AutoCompleteInput
              id="usage-filter-meter"
              label={t("usage.filters.meter")}
              value={filters.meterId}
              options={[]}
              placeholder={t("usage.filters.meter_placeholder")}
              onSearch={searchMeters}
              onChange={(value) => setFilters((p) => ({ ...p, meterId: value }))}
            />
          </div>
          <div className="filter-field">
            <AutoCompleteInput
              id="usage-filter-customer"
              label={t("usage.filters.customer")}
              value={filters.customerId}
              options={[]}
              placeholder={t("usage.filters.customer_placeholder")}
              onSearch={searchCustomers}
              onChange={(value) => setFilters((p) => ({ ...p, customerId: value }))}
            />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("usage.filters.status")}</label>
            <Select
              value={toSelectValue(filters.status)}
              onValueChange={(value) => setFilters((p) => ({ ...p, status: fromSelectValue(value) }))}
            >
              <SelectTrigger className="filter-select" data-testid="usage-filter-status">
                <SelectValue placeholder={t("common.all")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.all")}</SelectItem>
                <SelectItem value="accepted">{t("usage.status.accepted")}</SelectItem>
                <SelectItem value="enriched">{t("usage.status.enriched")}</SelectItem>
                <SelectItem value="rated">{t("usage.status.rated")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("usage.filters.recorded_from")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="datetime-local" value={filters.recordedFrom}
              onChange={(e) => setFilters((p) => ({ ...p, recordedFrom: e.target.value }))} data-testid="usage-filter-recorded-from" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("usage.filters.recorded_to")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="datetime-local" min={filters.recordedFrom || undefined} value={filters.recordedTo}
              onChange={(e) => setFilters((p) => ({ ...p, recordedTo: e.target.value }))} data-testid="usage-filter-recorded-to" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="usage-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={events}
        loading={listLoading && events.length === 0}
        emptyTitle={t("usage.table.empty_title")}
        emptyDesc={t("usage.table.empty_desc")}
        footer={hasMore ? (
          <Button variant="secondary" size="sm" disabled={listLoading} onClick={() => loadEvents(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
