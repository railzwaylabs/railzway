import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { formatDate, normalizeDate, rfc3339Hint, shortID } from "../lib/display"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { api } from "../lib/api"
import { formatCurrency, formatNumber } from "../lib/format"
import type { RatingResult, RatingSummary, UsageAggregate } from "../lib/types"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import { toast } from "../components/Toast"

function IconRating() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M3 15L7 9l4 3 3-6 4 3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function Rating() {
  const { t } = useTranslation()
  const [summary, setSummary] = useState<RatingSummary | null>(null)
  const [results, setResults] = useState<RatingResult[]>([])
  const [aggregates, setAggregates] = useState<UsageAggregate[]>([])
  const [activeView, setActiveView] = useState<"results" | "aggregates">("results")
  const [resultsNextToken, setResultsNextToken] = useState<string | undefined>(undefined)
  const [aggregatesNextToken, setAggregatesNextToken] = useState<string | undefined>(undefined)
  const [resultsHasMore, setResultsHasMore] = useState(false)
  const [aggregatesHasMore, setAggregatesHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [resultsLoading, setResultsLoading] = useState(false)
  const [aggregatesLoading, setAggregatesLoading] = useState(false)

  const defaultResultsFilters = useMemo(() => ({
    customerId: "", subscriptionId: "", planPriceId: "", meterId: "",
    usageEventId: "", windowStartFrom: "", windowStartTo: "", pageSize: 20,
  }), [])
  const defaultAggregatesFilters = useMemo(() => ({
    customerId: "", subscriptionId: "", planPriceId: "", meterId: "",
    periodStartFrom: "", periodStartTo: "", pageSize: 20,
  }), [])
  const [resultsFilters, setResultsFilters] = useState(defaultResultsFilters)
  const [aggregatesFilters, setAggregatesFilters] = useState(defaultAggregatesFilters)
  const resultsFiltersRef = useRef(defaultResultsFilters)
  const aggregatesFiltersRef = useRef(defaultAggregatesFilters)
  const resultsNextTokenRef = useRef<string | undefined>(undefined)
  const aggregatesNextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    resultsFiltersRef.current = resultsFilters
  }, [resultsFilters])

  useEffect(() => {
    aggregatesFiltersRef.current = aggregatesFilters
  }, [aggregatesFilters])

  const loadSummary = useCallback(async () => {
    try { setLoading(true); const d = await api.rating.summary(); setSummary(d) }
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

  const loadResults = useCallback(async (reset: boolean, overrideFilters?: typeof resultsFilters) => {
    try {
      setResultsLoading(true)
      if (reset) {
        resultsNextTokenRef.current = undefined
      }
      const f = overrideFilters ?? resultsFiltersRef.current
      const resp = await api.rating.results({
        page_token: reset ? undefined : resultsNextTokenRef.current, page_size: f.pageSize,
        customer_id: f.customerId, subscription_id: f.subscriptionId,
        plan_price_id: f.planPriceId, meter_id: f.meterId,
        usage_event_id: f.usageEventId,
        window_start_from: normalizeDate(f.windowStartFrom),
        window_start_to: normalizeDate(f.windowStartTo),
      })
      setResults((prev) => reset ? (resp.results ?? []) : [...prev, ...(resp.results ?? [])])
      setResultsNextToken(resp.next_page_token)
      resultsNextTokenRef.current = resp.next_page_token
      setResultsHasMore(Boolean(resp.has_more ?? resp.next_page_token))
    } catch (err) {
      toast.error(t("rating.toast_results_failed"), err instanceof Error ? err.message : undefined)
    } finally { setResultsLoading(false) }
  }, [])

  const loadAggregates = useCallback(async (reset: boolean, overrideFilters?: typeof aggregatesFilters) => {
    try {
      setAggregatesLoading(true)
      if (reset) {
        aggregatesNextTokenRef.current = undefined
      }
      const f = overrideFilters ?? aggregatesFiltersRef.current
      const resp = await api.rating.aggregates({
        page_token: reset ? undefined : aggregatesNextTokenRef.current, page_size: f.pageSize,
        customer_id: f.customerId, subscription_id: f.subscriptionId,
        plan_price_id: f.planPriceId, meter_id: f.meterId,
        period_start_from: normalizeDate(f.periodStartFrom),
        period_start_to: normalizeDate(f.periodStartTo),
      })
      setAggregates((prev) => reset ? (resp.aggregates ?? []) : [...prev, ...(resp.aggregates ?? [])])
      setAggregatesNextToken(resp.next_page_token)
      aggregatesNextTokenRef.current = resp.next_page_token
      setAggregatesHasMore(Boolean(resp.has_more ?? resp.next_page_token))
    } catch (err) {
      toast.error(t("rating.toast_aggregates_failed"), err instanceof Error ? err.message : undefined)
    } finally { setAggregatesLoading(false) }
  }, [])

  useEffect(() => {
    void loadSummary()
    void loadResults(true)
    void loadAggregates(true)
  }, [loadSummary, loadResults, loadAggregates])

  const latencyLabel = loading ? t("common.empty_dash") : `${(summary?.avgLatencySec ?? 0).toFixed(2)}s`

  const resultColumns = useMemo(() => [
    { key: "usage_event_id", label: t("rating.results.columns.usage_event"),
      render: (r: RatingResult) => <span className="cell-mono">{shortID(r.usage_event_id)}</span> },
    { key: "customer_id", label: t("rating.results.columns.customer"),
      render: (r: RatingResult) => <span className="cell-mono">{shortID(r.customer_id)}</span> },
    { key: "meter_id", label: t("rating.results.columns.meter"),
      render: (r: RatingResult) => <span className="cell-mono">{shortID(r.meter_id)}</span> },
    { key: "quantity", label: t("rating.results.columns.quantity"), width: "80px",
      render: (r: RatingResult) => <span>{formatNumber(r.quantity)}</span> },
    { key: "amount_cents", label: t("rating.results.columns.amount"), width: "120px",
      render: (r: RatingResult) => <strong>{formatCurrency(r.amount_cents, r.currency)}</strong> },
    { key: "window_start", label: t("rating.results.columns.window_start"), width: "135px",
      render: (r: RatingResult) => <span className="muted">{formatDate(r.window_start)}</span> },
  ], [t])

  const aggregateColumns = useMemo(() => [
    { key: "customer_id", label: t("rating.aggregates.columns.customer"),
      render: (r: UsageAggregate) => <span className="cell-mono">{shortID(r.customer_id)}</span> },
    { key: "meter_id", label: t("rating.aggregates.columns.meter"),
      render: (r: UsageAggregate) => <span className="cell-mono">{shortID(r.meter_id)}</span> },
    { key: "period", label: t("rating.aggregates.columns.period"),
      render: (r: UsageAggregate) => (
        <span className="muted">{formatDate(r.period_start)} → {formatDate(r.period_end)}</span>
      ) },
    { key: "quantity", label: t("rating.aggregates.columns.quantity"), width: "90px",
      render: (r: UsageAggregate) => <span>{formatNumber(r.quantity)}</span> },
    { key: "amount_cents", label: t("rating.aggregates.columns.amount"), width: "120px",
      render: (r: UsageAggregate) => <strong>{formatCurrency(r.amount_cents, r.currency)}</strong> },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader icon={<IconRating />} title={t("rating.header.title")} description={t("rating.header.description")} />

      <div className="stat-grid">
        <StatCard label={t("rating.kpis.rated_events")} value={loading ? t("common.empty_dash") : formatNumber(summary?.ratedEvents ?? 0)} accentColor="hsl(var(--accent-primary))" />
        <StatCard label={t("rating.kpis.avg_latency")} value={latencyLabel} accentColor="var(--accent-strong)" />
        <StatCard label={t("rating.kpis.replays_today")} value={loading ? t("common.empty_dash") : formatNumber(summary?.replaysToday ?? 0)} accentColor="hsl(var(--status-warning))" />
      </div>

      <div className="rating-tabs">
        <button
          type="button"
          className={`rating-tab${activeView === "results" ? " is-active" : ""}`}
          onClick={() => setActiveView("results")}
          data-testid="rating-tab-results"
        >
          {t("rating.tabs.results")}
        </button>
        <button
          type="button"
          className={`rating-tab${activeView === "aggregates" ? " is-active" : ""}`}
          onClick={() => setActiveView("aggregates")}
          data-testid="rating-tab-aggregates"
        >
          {t("rating.tabs.aggregates")}
        </button>
      </div>

      {activeView === "results" ? (
        <>
          <FilterPanel
            title={t("rating.results.title")}
            actions={
              <>
                <Button size="sm" variant="default" disabled={resultsLoading} onClick={() => loadResults(true)} data-testid="rating-results-apply">
                  {resultsLoading ? t("common.searching") : t("common.apply_filters")}
                </Button>
                <Button size="sm" variant="secondary" disabled={resultsLoading} onClick={() => {
                  setResultsFilters(defaultResultsFilters); setResultsNextToken(undefined)
                  void loadResults(true, defaultResultsFilters)
                }} data-testid="rating-results-reset">{t("common.reset")}</Button>
              </>
            }
          >
            <div className="filter-grid">
              <div className="filter-field">
                <AutoCompleteInput
                  id="rating-results-customer"
                  label={t("rating.filters.customer")}
                  value={resultsFilters.customerId}
                  options={[]}
                  placeholder={t("rating.filters.customer_placeholder")}
                  onSearch={searchCustomers}
                  onChange={(value) => setResultsFilters((p) => ({ ...p, customerId: value }))}
                />
              </div>
              <div className="filter-field">
                <AutoCompleteInput
                  id="rating-results-subscription"
                  label={t("rating.filters.subscription")}
                  value={resultsFilters.subscriptionId}
                  options={[]}
                  placeholder={t("rating.filters.subscription_placeholder")}
                  onSearch={searchSubscriptions}
                  onChange={(value) => setResultsFilters((p) => ({ ...p, subscriptionId: value }))}
                />
              </div>
              <div className="filter-field"><label className="filter-label">{t("rating.filters.plan_price_id")}</label>
                <Input className="filter-input" value={resultsFilters.planPriceId} onChange={(e) => setResultsFilters((p) => ({ ...p, planPriceId: e.target.value }))} data-testid="rating-results-plan-price" /></div>
              <div className="filter-field">
                <AutoCompleteInput
                  id="rating-results-meter"
                  label={t("rating.filters.meter")}
                  value={resultsFilters.meterId}
                  options={[]}
                  placeholder={t("rating.filters.meter_placeholder")}
                  onSearch={searchMeters}
                  onChange={(value) => setResultsFilters((p) => ({ ...p, meterId: value }))}
                />
              </div>
              <div className="filter-field"><label className="filter-label">{t("rating.filters.usage_event_id")}</label>
                <Input className="filter-input" value={resultsFilters.usageEventId} onChange={(e) => setResultsFilters((p) => ({ ...p, usageEventId: e.target.value }))} data-testid="rating-results-usage-event" /></div>
              <div className="filter-field"><label className="filter-label">{t("rating.filters.window_start_from")} <HelpHint text={rfc3339Hint} /></label>
                <Input className="filter-input" type="datetime-local" value={resultsFilters.windowStartFrom} onChange={(e) => setResultsFilters((p) => ({ ...p, windowStartFrom: e.target.value }))} data-testid="rating-results-window-from" /></div>
              <div className="filter-field"><label className="filter-label">{t("rating.filters.window_start_to")} <HelpHint text={rfc3339Hint} /></label>
                <Input className="filter-input" type="datetime-local" min={resultsFilters.windowStartFrom || undefined} value={resultsFilters.windowStartTo} onChange={(e) => setResultsFilters((p) => ({ ...p, windowStartTo: e.target.value }))} data-testid="rating-results-window-to" /></div>
              <div className="filter-field"><label className="filter-label">{t("common.page_size")}</label>
                <Input className="filter-input" type="number" min={1} max={100} value={resultsFilters.pageSize}
                  onChange={(e) => setResultsFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="rating-results-page-size" /></div>
            </div>
          </FilterPanel>

          <DataTable
            columns={resultColumns as Parameters<typeof DataTable>[0]["columns"]}
            data={results}
            loading={resultsLoading && results.length === 0}
            emptyTitle={t("rating.results.empty_title")}
            emptyDesc={t("rating.results.empty_desc")}
            footer={resultsHasMore ? (
              <Button size="sm" variant="secondary" disabled={resultsLoading} onClick={() => loadResults(false)}>
                {resultsLoading ? t("common.loading") : t("common.load_more")}
              </Button>
            ) : undefined}
          />
        </>
      ) : (
        <>
          <FilterPanel
            title={t("rating.aggregates.title")}
            actions={
              <>
                <Button size="sm" variant="default" disabled={aggregatesLoading} onClick={() => loadAggregates(true)} data-testid="rating-aggregates-apply">
                  {aggregatesLoading ? t("common.searching") : t("common.apply_filters")}
                </Button>
                <Button size="sm" variant="secondary" disabled={aggregatesLoading} onClick={() => {
                  setAggregatesFilters(defaultAggregatesFilters); setAggregatesNextToken(undefined)
                  void loadAggregates(true, defaultAggregatesFilters)
                }} data-testid="rating-aggregates-reset">{t("common.reset")}</Button>
              </>
            }
          >
            <div className="filter-grid">
              <div className="filter-field">
                <AutoCompleteInput
                  id="rating-aggregates-customer"
                  label={t("rating.filters.customer")}
                  value={aggregatesFilters.customerId}
                  options={[]}
                  placeholder={t("rating.filters.customer_placeholder")}
                  onSearch={searchCustomers}
                  onChange={(value) => setAggregatesFilters((p) => ({ ...p, customerId: value }))}
                />
              </div>
              <div className="filter-field">
                <AutoCompleteInput
                  id="rating-aggregates-subscription"
                  label={t("rating.filters.subscription")}
                  value={aggregatesFilters.subscriptionId}
                  options={[]}
                  placeholder={t("rating.filters.subscription_placeholder")}
                  onSearch={searchSubscriptions}
                  onChange={(value) => setAggregatesFilters((p) => ({ ...p, subscriptionId: value }))}
                />
              </div>
              <div className="filter-field"><label className="filter-label">{t("rating.filters.plan_price_id")}</label>
                <Input className="filter-input" value={aggregatesFilters.planPriceId} onChange={(e) => setAggregatesFilters((p) => ({ ...p, planPriceId: e.target.value }))} data-testid="rating-aggregates-plan-price" /></div>
              <div className="filter-field">
                <AutoCompleteInput
                  id="rating-aggregates-meter"
                  label={t("rating.filters.meter")}
                  value={aggregatesFilters.meterId}
                  options={[]}
                  placeholder={t("rating.filters.meter_placeholder")}
                  onSearch={searchMeters}
                  onChange={(value) => setAggregatesFilters((p) => ({ ...p, meterId: value }))}
                />
              </div>
              <div className="filter-field"><label className="filter-label">{t("rating.filters.period_start_from")} <HelpHint text={rfc3339Hint} /></label>
                <Input className="filter-input" type="datetime-local" value={aggregatesFilters.periodStartFrom} onChange={(e) => setAggregatesFilters((p) => ({ ...p, periodStartFrom: e.target.value }))} data-testid="rating-aggregates-period-from" /></div>
              <div className="filter-field"><label className="filter-label">{t("rating.filters.period_start_to")} <HelpHint text={rfc3339Hint} /></label>
                <Input className="filter-input" type="datetime-local" min={aggregatesFilters.periodStartFrom || undefined} value={aggregatesFilters.periodStartTo} onChange={(e) => setAggregatesFilters((p) => ({ ...p, periodStartTo: e.target.value }))} data-testid="rating-aggregates-period-to" /></div>
              <div className="filter-field"><label className="filter-label">{t("common.page_size")}</label>
                <Input className="filter-input" type="number" min={1} max={100} value={aggregatesFilters.pageSize}
                  onChange={(e) => setAggregatesFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="rating-aggregates-page-size" /></div>
            </div>
          </FilterPanel>

          <DataTable
            columns={aggregateColumns as Parameters<typeof DataTable>[0]["columns"]}
            data={aggregates}
            loading={aggregatesLoading && aggregates.length === 0}
            emptyTitle={t("rating.aggregates.empty_title")}
            emptyDesc={t("rating.aggregates.empty_desc")}
            footer={aggregatesHasMore ? (
              <Button size="sm" variant="secondary" disabled={aggregatesLoading} onClick={() => loadAggregates(false)}>
                {aggregatesLoading ? t("common.loading") : t("common.load_more")}
              </Button>
            ) : undefined}
          />
        </>
      )}
    </div>
  )
}
