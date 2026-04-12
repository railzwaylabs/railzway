import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { api } from "../lib/api"
import { formatDate } from "../lib/display"
import type { OrganizationListItem } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import { Button } from "../components/ui/button"
import { toast } from "../components/Toast"

function IconOrg() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="10" cy="6" r="3"/>
      <path d="M3 17c0-3.866 3.134-6 7-6s7 2.134 7 6" strokeLinecap="round"/>
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

export default function Organizations() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [orgs, setOrgs] = useState<OrganizationListItem[]>([])
  const [loading, setLoading] = useState(false)

  const loadOrganizations = useCallback(async () => {
    try {
      setLoading(true)
      const resp = await api.organizations.list()
      setOrgs(resp)
    } catch (err) {
      toast.error(t("organizations.toast_load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setLoading(false) }
  }, [t])

  useEffect(() => { void loadOrganizations() }, [loadOrganizations])

  const columns = [
    { key: "name", label: t("organizations.table.columns.name"), render: (r: OrganizationListItem) => (
      <div><div style={{ fontWeight: 600 }}>{r.name}</div><span className="cell-mono muted">{r.id.slice(0, 12)}…</span></div>
    ) },
    { key: "role", label: t("organizations.table.columns.role"), width: "100px", render: (r: OrganizationListItem) => (
      <span className="badge badge-info">{r.role}</span>
    ) },
    { key: "created_at", label: t("organizations.table.columns.created"), width: "130px", render: (r: OrganizationListItem) => (
      <span className="muted">{formatDate(r.created_at)}</span>
    ) },
    { key: "actions", label: "", width: "80px", render: (r: OrganizationListItem) => (
      <Button variant="outline" size="sm" onClick={() => navigate(`/organizations/${r.id}/edit`)}>
        {t("common.edit")}
      </Button>
    ) }
  ]

  return (
    <div className="page-content">
      <PageHeader 
        icon={<IconOrg />} 
        title={t("organizations.header.title")}
        description={t("organizations.header.description")}
        actions={
          <Button variant="default" onClick={() => navigate("/organizations/new")}>
            <IconPlus /> {t("organizations.actions.new")}
          </Button>
        }
      />

      <DataTable
        columns={columns as Parameters<typeof DataTable>[0]["columns"]}
        data={orgs}
        loading={loading}
        emptyTitle={t("organizations.table.empty_title")}
        emptyDesc={t("organizations.table.empty_desc")}
      />
    </div>
  )
}
