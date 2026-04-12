import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { formatDate, shortID } from "../lib/display"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { formatNumber } from "../lib/format"
import { statusClass } from "../lib/status"
import type { Subscription, SubscriptionsListResponse, SubscriptionsSummary } from "../lib/types"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"

function IconSubs() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M3 5h14v11a2 2 0 01-2 2H5a2 2 0 01-2-2V5z"/>
      <path d="M3 5V4a2 2 0 012-2h10a2 2 0 012 2v1"/>
      <path d="M8 10l2 2 4-4" strokeLinecap="round" strokeLinejoin="round"/>
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


export default function Subscriptions() {
  const { t } = useTranslation()
  const orgPath = useOrgPath()
  const [summary, setSummary] = useState<SubscriptionsSummary | null>(null)
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const defaultFilters = useMemo(() => ({ customerId: "", status: "", pageSize: 20 }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const loadSummary = useCallback(async () => {
    try {
      setLoading(true)
      const data = await api.subscriptions.summary()
      setSummary(data)
    } catch { /* ignore */ } finally { setLoading(false) }
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

  const applyListResponse = useCallback((resp: SubscriptionsListResponse, reset: boolean) => {
    setSubscriptions((prev) => (reset ? resp.subscriptions : [...prev, ...resp.subscriptions]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadSubscriptions = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const token = reset ? undefined : nextTokenRef.current
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.subscriptions.list({
        page_token: token, page_size: f.pageSize, customer_id: f.customerId, status: f.status,
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("subscriptions.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadSummary(); void loadSubscriptions(true) }, [loadSubscriptions, loadSummary])

  const columns = useMemo(() => [
    { key: "id", label: t("subscriptions.table.columns.subscription_id"),
      render: (r: Subscription) => <span className="cell-mono">{shortID(r.id)}</span> },
    { key: "customer_id", label: t("subscriptions.table.columns.customer"),
      render: (r: Subscription) => <span className="cell-mono">{shortID(r.customer_id)}</span> },
    { key: "plan_id", label: t("subscriptions.table.columns.plan"),
      render: (r: Subscription) => <span className="cell-mono">{shortID(r.plan_id)}</span> },
    { key: "currency", label: t("subscriptions.table.columns.currency"), width: "100px",
      render: (r: Subscription) => <span className="cell-mono">{r.currency || t("common.empty_dash")}</span> },
    { key: "status", label: t("subscriptions.table.columns.status"), width: "120px",
      render: (r: Subscription) => (
        <span className={`badge ${statusClass(r.status)}`}>
          {t(`subscriptions.status.${r.status}`, { defaultValue: r.status })}
        </span>
      ) },
    { key: "current_period_end", label: t("subscriptions.table.columns.period_end"), width: "130px",
      render: (r: Subscription) => <span className="muted">{formatDate(r.current_period_end)}</span> },
    { key: "actions", label: "", width: "90px", className: "col-actions",
      render: (r: Subscription) => (
        <Button asChild variant="secondary" size="sm">
          <Link to={orgPath(`/subscriptions/${r.id}/edit`)} data-testid={`subscriptions-manage-${r.id}`}>{t("common.manage")}</Link>
        </Button>
      ) },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconSubs />}
        title={t("subscriptions.header.title")}
        description={t("subscriptions.header.description")}
        actions={
          <Button asChild>
            <Link to={orgPath("/subscriptions/new")} style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <IconPlus /> {t("subscriptions.actions.new")}
            </Link>
          </Button>
        }
      />

      <div className="stat-grid">
        <StatCard label={t("subscriptions.kpis.active")} value={loading ? t("common.empty_dash") : formatNumber(summary?.active ?? 0)} accentColor="hsl(var(--status-success))" />
        <StatCard label={t("subscriptions.kpis.trialing")} value={loading ? t("common.empty_dash") : formatNumber(summary?.trialing ?? 0)} accentColor="var(--accent-strong)" />
        <StatCard label={t("subscriptions.kpis.past_due")} value={loading ? t("common.empty_dash") : formatNumber(summary?.pastDue ?? 0)} accentColor="hsl(var(--status-error))" />
      </div>

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadSubscriptions(true)} data-testid="subscriptions-filters-apply">
              {listLoading ? t("common.searching") : t("common.apply_filters")}
            </Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadSubscriptions(true, defaultFilters)
            }} data-testid="subscriptions-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <AutoCompleteInput
              id="subscriptions-filter-customer"
              label={t("subscriptions.filters.customer")}
              value={filters.customerId}
              options={[]}
              placeholder={t("subscriptions.filters.customer_placeholder")}
              onSearch={searchCustomers}
              onChange={(value) => setFilters((p) => ({ ...p, customerId: value }))}
            />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("subscriptions.filters.status")}</label>
            <Select
              value={toSelectValue(filters.status)}
              onValueChange={(value) => setFilters((p) => ({ ...p, status: fromSelectValue(value) }))}
            >
              <SelectTrigger className="filter-select" data-testid="subscriptions-filter-status">
                <SelectValue placeholder={t("common.all")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.all")}</SelectItem>
                <SelectItem value="trial">{t("subscriptions.status.trial")}</SelectItem>
                <SelectItem value="active">{t("subscriptions.status.active")}</SelectItem>
                <SelectItem value="past_due">{t("subscriptions.status.past_due")}</SelectItem>
                <SelectItem value="canceled">{t("subscriptions.status.canceled")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="subscriptions-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={subscriptions}
        loading={listLoading && subscriptions.length === 0}
        emptyTitle={t("subscriptions.table.empty_title")}
        emptyDesc={t("subscriptions.table.empty_desc")}
        footer={hasMore ? (
          <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => loadSubscriptions(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
