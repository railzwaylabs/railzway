import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import HelpHint from "../components/HelpHint"
import { formatDate, normalizeDate, rfc3339Hint, shortID } from "../lib/display"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { api } from "../lib/api"
import type { AuditLog, AuditLogsListResponse, AuditLogsSummary, OrganizationMemberInfo } from "../lib/types"
import DataTable from "../components/DataTable"
import FilterPanel from "../components/FilterPanel"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { useOrgIdParam } from "../lib/org"

function IconAudit() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="10" cy="10" r="7.5"/>
      <path d="M10 6.5v4l2.5 1.5" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function AuditLogs() {
  const { t } = useTranslation()
  const [summary, setSummary] = useState<AuditLogsSummary | null>(null)
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const [members, setMembers] = useState<OrganizationMemberInfo[]>([])
  const orgId = useOrgIdParam()
  const memberMap = useMemo(() => {
    const map = new Map<string, OrganizationMemberInfo>()
    for (const member of members) {
      map.set(member.user_id, member)
    }
    return map
  }, [members])
  const defaultFilters = useMemo(() => ({
    action: "", actorType: "", resourceType: "", resourceId: "", requestId: "",
    createdFrom: "", createdTo: "", pageSize: 20,
  }), [])
  const [filters, setFilters] = useState(defaultFilters)
  const filtersRef = useRef(defaultFilters)
  const nextTokenRef = useRef<string | undefined>(undefined)

  useEffect(() => {
    filtersRef.current = filters
  }, [filters])

  const loadSummary = useCallback(async () => {
    try { setLoading(true); const d = await api.auditLogs.summary(); setSummary(d) }
    catch { /* ignore */ } finally { setLoading(false) }
  }, [])

  const applyListResponse = useCallback((resp: AuditLogsListResponse, reset: boolean) => {
    setLogs((prev) => (reset ? resp.logs : [...prev, ...resp.logs]))
    setNextToken(resp.next_page_token)
    nextTokenRef.current = resp.next_page_token
    setHasMore(Boolean(resp.has_more ?? resp.next_page_token))
  }, [])

  const loadLogs = useCallback(async (reset: boolean, overrideFilters?: typeof filters) => {
    try {
      setListLoading(true)
      if (reset) {
        nextTokenRef.current = undefined
      }
      const f = overrideFilters ?? filtersRef.current
      const resp = await api.auditLogs.list({
        page_token: reset ? undefined : nextTokenRef.current, page_size: f.pageSize,
        action: f.action, actor_type: f.actorType, resource_type: f.resourceType,
        resource_id: f.resourceId, request_id: f.requestId,
        created_from: normalizeDate(f.createdFrom), created_to: normalizeDate(f.createdTo),
      })
      applyListResponse(resp, reset)
    } catch (err) {
      toast.error(t("audit_logs.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setListLoading(false) }
  }, [applyListResponse, t])

  useEffect(() => { void loadSummary(); void loadLogs(true) }, [loadLogs, loadSummary])

  useEffect(() => {
    if (!orgId) return
    api.organizations.listMembers(orgId)
      .then(setMembers)
      .catch(() => setMembers([]))
  }, [orgId])

  const highlights = summary?.entries ?? []

  const highlightColumns = [
    { key: "title", label: t("audit_logs.highlights.columns.entry"),
      render: (r: { title: string; note: string }) => (
        <div><div style={{ fontWeight: 600 }}>{r.title}</div><div className="muted">{r.note}</div></div>
      ) },
    { key: "tag", label: t("audit_logs.highlights.columns.tag"), width: "120px",
      render: (r: { tag: string }) => <span className="badge badge-info">{r.tag}</span> },
  ]

  const logColumns = [
    { key: "action", label: t("audit_logs.table.columns.action"), render: (r: AuditLog) => <strong>{r.action}</strong> },
    { key: "resource_type", label: t("audit_logs.table.columns.resource"), render: (r: AuditLog) => (
      <div>
        <div>{r.resource_type}</div>
        <span className="cell-mono">{shortID(r.resource_id)}</span>
      </div>
    ) },
    { key: "actor_type", label: t("audit_logs.table.columns.actor"), width: "200px", render: (r: AuditLog) => {
      const member = r.actor_id ? memberMap.get(r.actor_id) : undefined
      if (r.actor_type === "user" && member) {
        return (
          <div>
            <a
              className="link"
              href={`/organizations/${orgId}/edit?member_id=${member.user_id}#members`}
            >
              {member.display_name || member.email}
            </a>
            <div className="muted" style={{ fontSize: "0.75rem" }}>{member.email}</div>
          </div>
        )
      }
      return (
        <div>
          <div className="muted">{r.actor_type}</div>
          <span className="cell-mono">{shortID(r.actor_id)}</span>
        </div>
      )
    } },
    { key: "reason", label: t("audit_logs.table.columns.reason"), render: (r: AuditLog) => <span className="muted">{r.reason || t("common.empty_dash")}</span> },
    { key: "created_at", label: t("audit_logs.table.columns.when"), width: "145px",
      render: (r: AuditLog) => <span className="muted">{formatDate(r.created_at)}</span> },
  ]

  return (
    <div className="page-content">
      <PageHeader icon={<IconAudit />} title={t("audit_logs.header.title")} description={t("audit_logs.header.description")} />

      {highlights.length > 0 ? (
        <DataTable
          columns={highlightColumns as Parameters<typeof DataTable>[0]["columns"]}
          data={highlights as Array<{ id?: string; title: string; note: string; tag: string }>}
          loading={loading}
          emptyTitle={t("audit_logs.highlights.empty_title")}
          emptyDesc={t("audit_logs.highlights.empty_desc")}
          keyExtractor={(r) => `${r.title}-${r.note}`}
        />
      ) : null}

      <FilterPanel
        title={t("audit_logs.filters.title")}
        actions={
          <>
            <Button size="sm" variant="default" disabled={listLoading} onClick={() => loadLogs(true)} data-testid="audit-logs-apply">
              {listLoading ? t("common.searching") : t("common.apply_filters")}
            </Button>
            <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => {
              setFilters(defaultFilters); setNextToken(undefined); void loadLogs(true, defaultFilters)
            }} data-testid="audit-logs-reset">{t("common.reset")}</Button>
          </>
        }
      >
        <div className="filter-grid">
          <div className="filter-field"><label className="filter-label">{t("audit_logs.filters.action")}</label>
            <Input className="filter-input" value={filters.action} onChange={(e) => setFilters((p) => ({ ...p, action: e.target.value }))} data-testid="audit-logs-filter-action" /></div>
          <div className="filter-field"><label className="filter-label">{t("audit_logs.filters.actor_type")}</label>
            <Input className="filter-input" value={filters.actorType} placeholder={t("audit_logs.filters.actor_type_placeholder")} onChange={(e) => setFilters((p) => ({ ...p, actorType: e.target.value }))} data-testid="audit-logs-filter-actor-type" /></div>
          <div className="filter-field"><label className="filter-label">{t("audit_logs.filters.resource_type")}</label>
            <Input className="filter-input" value={filters.resourceType} placeholder={t("audit_logs.filters.resource_type_placeholder")} onChange={(e) => setFilters((p) => ({ ...p, resourceType: e.target.value }))} data-testid="audit-logs-filter-resource-type" /></div>
          <div className="filter-field"><label className="filter-label">{t("audit_logs.filters.resource_id")}</label>
            <Input className="filter-input" value={filters.resourceId} onChange={(e) => setFilters((p) => ({ ...p, resourceId: e.target.value }))} data-testid="audit-logs-filter-resource-id" /></div>
          <div className="filter-field"><label className="filter-label">{t("audit_logs.filters.request_id")}</label>
            <Input className="filter-input" value={filters.requestId} onChange={(e) => setFilters((p) => ({ ...p, requestId: e.target.value }))} data-testid="audit-logs-filter-request-id" /></div>
          <div className="filter-field"><label className="filter-label">{t("audit_logs.filters.created_from")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" value={filters.createdFrom} onChange={(e) => setFilters((p) => ({ ...p, createdFrom: e.target.value }))} data-testid="audit-logs-filter-created-from" /></div>
          <div className="filter-field"><label className="filter-label">{t("audit_logs.filters.created_to")} <HelpHint text={rfc3339Hint} /></label>
            <Input className="filter-input" type="date" min={filters.createdFrom || undefined} value={filters.createdTo} onChange={(e) => setFilters((p) => ({ ...p, createdTo: e.target.value }))} data-testid="audit-logs-filter-created-to" /></div>
          <div className="filter-field"><label className="filter-label">{t("common.page_size")}</label>
            <Input className="filter-input" type="number" min={1} max={100} value={filters.pageSize}
              onChange={(e) => setFilters((p) => ({ ...p, pageSize: Number.parseInt(e.target.value || "20", 10) }))} data-testid="audit-logs-filter-page-size" /></div>
        </div>
      </FilterPanel>

      <DataTable
        columns={logColumns as Parameters<typeof DataTable>[0]["columns"]}
        data={logs}
        loading={listLoading && logs.length === 0}
        emptyTitle={t("audit_logs.table.empty_title")}
        emptyDesc={t("audit_logs.table.empty_desc")}
        footer={hasMore ? (
          <Button size="sm" variant="secondary" disabled={listLoading} onClick={() => loadLogs(false)}>
            {listLoading ? t("common.loading") : t("common.load_more")}
          </Button>
        ) : undefined}
      />
    </div>
  )
}
