import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { useParams, useNavigate, Link } from "react-router-dom"
import PageHeader from "../components/PageHeader"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import AutoCompleteInput from "../components/AutoCompleteInput"
import type { Product, Feature, PlanResponse } from "../lib/types"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs"
import DataTable from "../components/DataTable"
import { formatDate } from "../lib/display"

export default function ProductsEdit() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  
  const [product, setProduct] = useState<Product | null>(null)
  const [form, setForm] = useState({
    name: "",
    description: "",
    active: true
  })
  
  // Features state
  const [allFeatures, setAllFeatures] = useState<Feature[]>([])
  const [selectedFeatureIds, setSelectedFeatureIds] = useState<Set<string>>(new Set())

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadData = useCallback(async () => {
    if (!id) return
    try {
      setLoading(true)
      const [productData, featuresData] = await Promise.all([
        api.products.get(id, { expand: "features,plans" }),
        api.features.list({ page_size: 100 })
      ])
      
      setProduct(productData)
      setForm({
        name: productData.name,
        description: productData.description || "",
        active: productData.active
      })
      
      setAllFeatures(featuresData.features || [])
      
      const pFeatures = productData.features || []
      const selected = new Set(pFeatures.filter(f => f.active).map(f => f.id))
      setSelectedFeatureIds(selected)
      
    } catch (err) {
      setError(t("products_edit.toast.load_failed") || "Failed to load product data")
    } finally {
      setLoading(false)
    }
  }, [id, t])

  useEffect(() => { void loadData() }, [loadData])

  const handleSubmit = async () => {
    if (!id) return
    try {
      setSaving(true)
      setError(null)
      // Save product details + features in one simple update API if Backend supports it, 
      // otherwise two calls. Backend recently updated to support aggregate Update for features.
      await api.products.update(id, {
        name: form.name,
        description: form.description || undefined,
        active: form.active,
        feature_ids: Array.from(selectedFeatureIds)
      })
      
      navigate(orgPath("/products"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("products_edit.toast.update_failed"))
    } finally {
      setSaving(false)
    }
  }

  const toggleFeature = (featureId: string) => {
    setSelectedFeatureIds(prev => {
      const next = new Set(prev)
      if (next.has(featureId)) {
        next.delete(featureId)
      } else {
        next.add(featureId)
      }
      return next
    })
  }

  const planColumns = [
    {
      key: "name", label: t("plans.table.columns.plan"),
      render: (row: PlanResponse) => (
        <Link to={orgPath(`/plans/${row.id}/edit`)} className="link-standard font-medium">
          {row.name}
        </Link>
      )
    },
    { key: "code", label: t("plans.filters.code"), render: (row: PlanResponse) => <span className="cell-mono text-xs">{row.code}</span> },
    { key: "prices", label: t("products.table.columns.prices"), render: (row: PlanResponse) => (
      <span className="muted text-xs">{t("products.table.price_count", { count: (row.prices || []).length })}</span>
    )},
    { key: "status", label: t("plans.table.columns.status"), render: (row: PlanResponse) => (
      <span className={`badge ${row.active ? 'badge-success' : 'badge-neutral'}`}>
        {row.active ? t("plans.table.status.active") : t("plans.table.status.inactive")}
      </span>
    )},
    { key: "created_at", label: t("plans.table.columns.created"), render: (row: PlanResponse) => <span className="muted text-xs">{formatDate(row.created_at)}</span> },
  ]

  const disabled = saving || !form.name

  if (loading) return <div className="page-content p-8 text-center muted">{t("common.loading")}...</div>
  if (!product) return <div className="page-content">{t("not_found.title")}</div>

  return (
    <div className="page-content">
      <PageHeader
        title={t("products_edit.header.title_with_name", { name: product.code })}
        description={t("products.header.description")}
      />

      <Tabs defaultValue="details" className="panel" style={{ maxWidth: 960 }}>
        <TabsList style={{ borderBottom: "1px solid var(--border-color)", padding: "0 16px" }}>
          <TabsTrigger value="details">{t("products_edit.tabs.details")}</TabsTrigger>
          <TabsTrigger value="features">{t("products_edit.tabs.features")}</TabsTrigger>
          <TabsTrigger value="plans">{t("products_edit.tabs.plans")}</TabsTrigger>
        </TabsList>
        
        <TabsContent value="details" style={{ padding: "24px" }}>
          <div className="action-section" style={{ border: "none" }}>
            <div className="action-fields mb-8">
              <div className="action-field">
                <Label className="action-label mb-2">{t("plans_create.fields.plan_name")} *</Label>
                <Input
                  className="action-input"
                  value={form.name}
                  onChange={(e) => setForm(p => ({ ...p, name: e.target.value }))}
                />
              </div>
              <div className="action-field">
                <Label className="action-label mb-2">{t("plans_create.fields.description")}</Label>
                <Input
                  className="action-input"
                  value={form.description}
                  onChange={(e) => setForm(p => ({ ...p, description: e.target.value }))}
                />
              </div>
              <div className="action-field">
                <AutoCompleteInput
                  id="product-active"
                  label={<>{t("plans.table.columns.status")}</>}
                  value={form.active ? "true" : "false"}
                  options={[{value: "true", label: t("plans.table.status.active")}, {value: "false", label: t("plans.table.status.inactive")}]}
                  onChange={(v) => setForm(p => ({ ...p, active: v === "true" }))}
                />
              </div>
            </div>
            {error && <div className="inline-error mb-4">{error}</div>}
            <div className="action-buttons pt-6 border-t">
              <Button onClick={handleSubmit} disabled={disabled} size="lg">
                {saving ? t("common.saving") : t("products_edit.actions.submit")}
              </Button>
              <Button variant="secondary" onClick={() => navigate(orgPath("/products"))} disabled={saving} size="lg">
                {t("common.cancel")}
              </Button>
            </div>
          </div>
        </TabsContent>
        
        <TabsContent value="features" style={{ padding: "24px" }}>
          <div className="action-section" style={{ border: "none" }}>
            <div className="action-section-title" style={{ marginBottom: "16px" }}>{t("features.header.title")}</div>
            <p className="muted mb-8 text-sm">
              {t("features.header.description")}
            </p>
            
            {allFeatures.length === 0 ? (
              <div className="muted p-8 text-center italic">{t("features.empty_title")}</div>
            ) : (
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px" }}>
                {allFeatures.map(feature => (
                  <label key={feature.id} className={`flex items-start gap-3 p-4 border rounded-lg cursor-pointer transition-colors ${selectedFeatureIds.has(feature.id) ? "bg-primary/5 border-primary" : "bg-white border-border hover:border-primary/30"}`}>
                    <input
                      type="checkbox"
                      checked={selectedFeatureIds.has(feature.id)}
                      onChange={() => toggleFeature(feature.id)}
                      className="mt-1"
                    />
                    <div className="flex-1 min-w-0">
                      <div className="font-semibold text-sm truncate">{feature.name}</div>
                      <div className="muted text-xs font-mono">{feature.code}</div>
                      <div className="muted text-[10px] uppercase mt-1 tracking-wider">{feature.feature_type}</div>
                    </div>
                  </label>
                ))}
              </div>
            )}
            
            <div className="action-buttons pt-8 mt-8 border-t">
              <Button onClick={handleSubmit} disabled={disabled} size="lg">
                {saving ? t("common.saving") : t("products_edit.actions.update_features")}
              </Button>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="plans" style={{ padding: "24px" }}>
           <div className="action-section" style={{ border: "none" }}>
              <div className="flex justify-between items-center mb-6">
                <div>
                  <h3 className="text-lg font-semibold">{t("products_edit.plans.title")}</h3>
                  <p className="text-sm muted">{t("products_edit.plans.description")}</p>
                </div>
                <Button asChild size="sm" variant="outline">
                  <Link to={orgPath("/plans/new")}>+ {t("plans.actions.new")}</Link>
                </Button>
              </div>
              
              <DataTable
                columns={planColumns as any}
                data={product.plans || []}
                loading={false}
                emptyTitle={t("plans.table.empty_title")}
                emptyDesc={t("plans.table.empty_desc")}
              />

              <div className="mt-8 p-4 bg-amber-50 rounded-lg border border-amber-200">
                <p className="text-xs text-amber-800">
                  <strong>{t("products_edit.plans.boundary_title")}</strong> {t("products_edit.plans.boundary_desc")}
                </p>
              </div>
           </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
