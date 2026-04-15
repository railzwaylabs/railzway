import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { formatDate } from "../lib/display"
import { Button } from "../components/ui/button"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import type { Feature } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"

function IconFeature() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 2L2 7l10 5 10-5-10-5z" />
      <path d="M2 17l10 5 10-5" />
      <path d="M2 12l10 5 10-5" />
    </svg>
  )
}

export default function Features() {
  const { t } = useTranslation()
  const orgPath = useOrgPath()
  const [features, setFeatures] = useState<Feature[]>([])
  const [loading, setLoading] = useState(true)

  const loadFeatures = useCallback(async () => {
    try {
      setLoading(true)
      const data = await api.features.list()
      setFeatures(data.features || [])
    } catch (err) {
      // Handle error
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void loadFeatures() }, [loadFeatures])

  const columns = useMemo(() => [
    {
      key: "name", label: t("common.name"),
      render: (row: Feature) => (
        <div>
          <div style={{ fontWeight: 600 }}>{row.name}</div>
          <div className="muted" style={{ fontSize: "11px" }}>{row.code}</div>
        </div>
      ),
    },
    {
      key: "type", label: t("plans_edit.price_fields.type"), width: "120px",
      render: (row: Feature) => <span className="cell-mono">{row.feature_type}</span>
    },
    {
      key: "status", label: t("plans.table.columns.status"), width: "100px",
      render: (row: Feature) => (
        <span className={`badge ${row.active ? 'badge-success' : 'badge-neutral'}`}>
          {row.active ? t("plans.table.status.active") : t("plans.table.status.inactive")}
        </span>
      )
    },
    {
      key: "created_at", label: t("common.created"), width: "130px",
      render: (row: Feature) => <span className="muted">{formatDate(row.created_at)}</span>
    },
    {
      key: "actions", label: "", width: "80px", className: "col-actions",
      render: (row: Feature) => (
        <Button asChild variant="secondary" size="sm" data-testid={`features-edit-${row.id}`}>
          <Link to={orgPath(`/features/${row.id}/edit`)}>{t("common.edit")}</Link>
        </Button>
      ),
    },
  ], [orgPath, t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconFeature />}
        title={t("features.header.title")}
        description={t("features.header.description")}
        actions={
          <Button asChild data-testid="features-new-button">
            <Link to={orgPath("/features/new")}>+ {t("features.actions.new")}</Link>
          </Button>
        }
      />

      <DataTable
        columns={columns as any}
        data={features}
        loading={loading}
        emptyTitle={t("features.empty_title")}
        emptyDesc={t("features.empty_desc")}
      />
    </div>
  )
}
