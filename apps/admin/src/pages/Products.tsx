import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { formatDate } from "../lib/display"
import { Button } from "../components/ui/button"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import type { Product } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"

function IconProduct() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
      <line x1="8" y1="21" x2="16" y2="21" />
      <line x1="12" y1="17" x2="12" y2="21" />
    </svg>
  )
}

export default function Products() {
  const { t } = useTranslation()
  const orgPath = useOrgPath()
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(true)

  const loadProducts = useCallback(async () => {
    try {
      setLoading(true)
      const data = await api.products.list()
      setProducts(data.products || [])
    } catch (err) {
      // Handle error
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void loadProducts() }, [loadProducts])

  const columns = useMemo(() => [
    {
      key: "name", label: t("common.name"),
      render: (row: Product) => (
        <div>
          <div style={{ fontWeight: 600 }}>{row.name}</div>
          <div className="muted" style={{ fontSize: "11px" }}>{row.code}</div>
        </div>
      ),
    },
    {
      key: "status", label: t("plans.table.columns.status"), width: "100px",
      render: (row: Product) => (
        <span className={`badge ${row.active ? 'badge-success' : 'badge-neutral'}`}>
          {row.active ? t("plans.table.status.active") : t("plans.table.status.inactive")}
        </span>
      )
    },
    {
      key: "created_at", label: t("common.created"), width: "130px",
      render: (row: Product) => <span className="muted">{formatDate(row.created_at)}</span>
    },
    {
      key: "actions", label: "", width: "80px", className: "col-actions",
      render: (row: Product) => (
        <Button asChild variant="secondary" size="sm" data-testid={`products-edit-${row.id}`}>
          <Link to={orgPath(`/products/${row.id}/edit`)}>{t("common.edit")}</Link>
        </Button>
      ),
    },
  ], [orgPath, t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconProduct />}
        title={t("products.header.title")}
        description={t("products.header.description")}
        actions={
          <Button asChild data-testid="products-new-button">
            <Link to={orgPath("/products/new")}>+ {t("products.actions.new")}</Link>
          </Button>
        }
      />

      <DataTable
        columns={columns as any}
        data={products}
        loading={loading}
        emptyTitle={t("products.empty_title")}
        emptyDesc={t("products.empty_desc")}
      />
    </div>
  )
}
