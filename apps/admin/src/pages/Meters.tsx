import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import type { Meter, MetersListResponse } from "../lib/types"
import { statusClass } from "../lib/status"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"

function IconMeter() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="10" cy="10" r="7.5"/>
      <path d="M10 10L6.5 6.5" strokeLinecap="round"/>
      <circle cx="10" cy="10" r="1.5" fill="currentColor" stroke="none"/>
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

const formatDate = (value?: string) => (value ? new Date(value).toLocaleDateString() : "—")

export default function Meters() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [meters, setMeters] = useState<Meter[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [listLoading, setListLoading] = useState(false)
  const [actionLoading, setActionLoading] = useState(false)
  const defaultFilters = useMemo(() => ({ code: "", name: "", active: "", pageSize: 20 }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const applyListResponse = useCallback((resp: MetersListResponse, reset: boolean) => {
    setMeters((prev) => (reset ? resp.meters : [...prev, ...resp.meters]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadMeters = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const token = reset ? undefined : nextTokenRef.current
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.meters.list({
        page_token: token, page_size: f.pageSize, code: f.code, name: f.name, active: f.active,
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("meters.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse])

  useEffect(() => { void loadMeters(true) }, [loadMeters])

  const handleToggle = useCallback(async (meterId: string, active: boolean) => {
    try {
      setActionLoading(true)
      await api.meters.update(meterId, { active: !active })
      toast.success(t(!active ? "meters.toast_enabled" : "meters.toast_disabled"))
      void loadMeters(true)
    } catch (err) {
      toast.error(t("meters.toast_toggle_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [loadMeters])

  const columns = useMemo(() => [
    { key: "name", label: t("meters.table.columns.meter"),
      render: (r: Meter) => (
        <div><div style={{ fontWeight: 600 }}>{r.name}</div><span className="cell-mono">{r.code}</span></div>
      ) },
    { key: "aggregation", label: t("meters.table.columns.aggregation"), width: "120px",
      render: (r: Meter) => <span className="cell-mono">{r.aggregation || t("common.empty_dash")}</span> },
    { key: "unit", label: t("meters.table.columns.unit"), width: "100px",
      render: (r: Meter) => <span className="muted">{r.unit || t("common.empty_dash")}</span> },
    { key: "active", label: t("meters.table.columns.status"), width: "100px",
      render: (r: Meter) => (
        <span className={`badge ${statusClass(r.active ? "active" : "inactive")}`}>
          {r.active ? t("meters.table.status.active") : t("meters.table.status.inactive")}
        </span>
      ) },
    { key: "created_at", label: t("meters.table.columns.created"), width: "130px",
      render: (r: Meter) => <span className="muted">{formatDate(r.created_at)}</span> },
    { key: "actions", label: "", width: "140px", className: "col-actions",
      render: (r: Meter) => (
        <div style={{ display: "flex", gap: "8px" }}>
          <Button variant="outline" size="sm" onClick={() => navigate(orgPath(`/meters/${r.id}/edit`))} data-testid={`meters-edit-${r.id}`}>
            {t("common.edit")}
          </Button>
          <Button variant="secondary" size="sm" disabled={actionLoading}
            onClick={() => handleToggle(r.id, r.active)}>
            {r.active ? t("common.disable") : t("common.enable")}
          </Button>
        </div>
      ) },
  ], [actionLoading, handleToggle, navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader 
        icon={<IconMeter />} 
        title={t("meters.header.title")} 
        description={t("meters.header.description")} 
        actions={
          <Button variant="default" onClick={() => navigate(orgPath("/meters/new"))} style={{ display: "flex", gap: 6, alignItems: "center" }}>
            <IconPlus /> {t("meters.actions.new")}
          </Button>
        }
      />

      <FilterPanel
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadMeters(true)} data-testid="meters-filters-apply">
              {listLoading ? t("common.searching") : t("common.apply_filters")}
            </Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadMeters(true, defaultFilters)
            }} data-testid="meters-filters-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field">
            <label className="filter-label">{t("meters.filters.code")}</label>
            <Input className="filter-input" value={filters.code}
              onChange={(e) => setFilters((p) => ({ ...p, code: e.target.value }))} data-testid="meters-filter-code" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("meters.filters.name")}</label>
            <Input className="filter-input" value={filters.name}
              onChange={(e) => setFilters((p) => ({ ...p, name: e.target.value }))} data-testid="meters-filter-name" />
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("meters.filters.status")}</label>
            <Select
              value={toSelectValue(filters.active)}
              onValueChange={(value) => setFilters((p) => ({ ...p, active: fromSelectValue(value) }))}
            >
              <SelectTrigger className="filter-select" data-testid="meters-filter-active">
                <SelectValue placeholder={t("common.all")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>{t("common.all")}</SelectItem>
                <SelectItem value="true">{t("meters.table.status.active")}</SelectItem>
                <SelectItem value="false">{t("meters.table.status.inactive")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="filter-field">
            <label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="meters-filter-page-size" />
          </div>
        </div>
      </FilterPanel>

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={meters}
        loading={listLoading && meters.length === 0}
        emptyTitle={t("meters.table.empty_title")}
        emptyDesc={t("meters.table.empty_desc")}
        footer={hasMore ? (
          <Button variant="secondary" size="sm" disabled={listLoading} onClick={() => loadMeters(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
