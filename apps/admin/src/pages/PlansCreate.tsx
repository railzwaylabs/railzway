import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"

import PlanFeatureEditor, { buildDraftPlanFeatures, type PlanFeatureDraft } from "../components/PlanFeatureEditor"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { currencyHint } from "../lib/hints"
import { DEFAULT_MONEY_INPUT, defaultMoneyInputForPriceType, isNonNegativeMoneyInput, moneyInputDecimalsForPriceType, moneyInputStepForPriceType, moneyInputToCents } from "../lib/money"
import { useCurrencies } from "../lib/reference"
import type { ProductFeature } from "../lib/types"
import { isCurrencyCode } from "../lib/validation"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import HelpHint from "../components/HelpHint"

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function PlansCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies()
  const [loading, setLoading] = useState(false)
  const [meterOptions, setMeterOptions] = useState<Array<{ value: string; label: string }>>([])
  const [productOptions, setProductOptions] = useState<Array<{ value: string; label: string }>>([])
  const [productFeatures, setProductFeatures] = useState<ProductFeature[]>([])
  const [planFeatureRows, setPlanFeatureRows] = useState<PlanFeatureDraft[]>([])

  // Load meters for usage-based plans
  useEffect(() => {
    let active = true
    const loadMeters = async () => {
      try {
        const resp = await api.meters.list({ page_size: 100, active: "true" })
        if (active) {
          setMeterOptions(resp.meters.map((meter) => ({
            value: meter.id, label: `${meter.name} · ${meter.code}`
          })))
        }
      } catch (err) { /* ignore */ }
    }
    void loadMeters()
    return () => { active = false }
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

  const [form, setForm] = useState({
    productId: "",
    // Plan Info
    planCode: "",
    planName: "",
    planDescription: "",
    
    // Pricing Info
    priceType: "flat", // flat | usage
    currency: "USD",
    billingInterval: "month",
    billingIntervalCount: 1,
    unitAmount: DEFAULT_MONEY_INPUT,
    
    // Usage Info
    meterId: "",
  })

  useEffect(() => {
    let active = true
    const loadProductFeatures = async () => {
      if (!form.productId) {
        if (active) {
          setProductFeatures([])
          setPlanFeatureRows([])
        }
        return
      }
      try {
        const features = await api.productFeatures.listByProduct(form.productId)
        if (!active) return
        setProductFeatures(features)
        setPlanFeatureRows((current) => {
          const existing = current
            .filter((row) => !row.inherited)
            .map((row) => ({
              id: row.feature_id,
              enabled: row.enabled,
              limit_numeric: row.limit_numeric.trim() ? Number(row.limit_numeric) : undefined,
              limit_unit: row.limit_unit.trim() || undefined,
              reset_period: row.reset_period,
            }))
          return buildDraftPlanFeatures(features, existing)
        })
      } catch (_err) {
        if (!active) return
        setProductFeatures([])
        setPlanFeatureRows([])
      }
    }
    void loadProductFeatures()
    return () => { active = false }
  }, [form.productId])

  const searchProducts = useCallback(async (query: string) => {
    const resp = await api.products.list({ page_size: 50, active: "true", name: query })
    let products = resp.products
    if (products.length === 0) {
      const fallback = await api.products.list({ page_size: 50, active: "true", code: query })
      products = fallback.products
    }
    const options = products.map((product) => ({
      value: product.id,
      label: `${product.name} · ${product.code}`,
    }))
    setProductOptions(options)
    return options
  }, [])

  const validation = useMemo(() => {
    const errors: string[] = []

    // Plan validation
    if (!form.planCode.trim()) errors.push(t("plans_create.validation.code_required"))
    if (!form.planName.trim()) errors.push(t("plans_create.validation.name_required"))

    // Price validation
    if (!form.currency.trim()) errors.push(t("plans_create.validation.currency_required"))
    else if (!isCurrencyCode(form.currency)) errors.push(t("plans_create.validation.currency_invalid"))
    if (form.billingIntervalCount < 1) errors.push(t("plans_create.validation.interval_count"))
    const amountDecimals = moneyInputDecimalsForPriceType(form.priceType)
    if (!isNonNegativeMoneyInput(form.unitAmount, amountDecimals)) errors.push(t("plans_create.validation.amount_min"))

    // Usage Validation
    if (form.priceType === "usage" && !form.meterId) errors.push(t("plans_create.validation.meter_required"))
    const invalidFeatureLimit = planFeatureRows.some((row) => {
      if (!row.active || row.inherited || !row.enabled || !row.limit_numeric.trim()) return false
      const parsed = Number(row.limit_numeric)
      return !Number.isFinite(parsed) || parsed < 0
    })
    if (invalidFeatureLimit) errors.push(t("plans_create.validation.plan_feature_limit"))

    return errors
  }, [form, planFeatureRows, t])

  const serializePlanFeatures = useCallback(
    (rows: PlanFeatureDraft[]) =>
      rows
        .filter((row) => row.active && !row.inherited)
        .map((row) => ({
          feature_id: row.feature_id,
          enabled: row.enabled,
          limit_numeric: row.enabled && row.limit_numeric.trim() ? Number(row.limit_numeric) : undefined,
          limit_unit: row.enabled && row.limit_unit.trim() ? row.limit_unit.trim() : undefined,
          reset_period: row.enabled ? row.reset_period : "none",
        })),
    []
  )

  const handleCreate = useCallback(async () => {
    try {
      setLoading(true)

      const plan = await api.plans.create({
        product_id: form.productId || undefined,
        code: form.planCode.trim(),
        name: form.planName.trim(),
        description: form.planDescription.trim() || undefined,
        active: true,
        prices: [
          {
            code: `${form.planCode.trim()}_base_price`,
            name: t("plans_create.defaults.base_price"),
            price_type: form.priceType.trim(),
            billing_interval: form.billingInterval.trim(),
            billing_interval_count: form.billingIntervalCount,
            meter_id: form.priceType === "usage" ? form.meterId : undefined,
            active: true,
            amounts: [
              {
                currency: form.currency.trim(),
                unit_amount_cents: moneyInputToCents(form.unitAmount, moneyInputDecimalsForPriceType(form.priceType))
              }
            ]
          }
        ]
      })

      const featurePayload = serializePlanFeatures(planFeatureRows)
      if (featurePayload.length > 0) {
        await api.plans.replaceFeatures(plan.id, { features: featurePayload })
      }

      toast.success(t("plans_create.toast.created_title"), t("plans_create.toast.created_desc"))
      navigate(orgPath("/plans"))
    } catch (err) {
      toast.error(t("plans_create.toast.create_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [form, navigate, orgPath, planFeatureRows, serializePlanFeatures, t])

  return (
    <div className="page-content">
      <PageHeader
        title={t("plans_create.header.title")}
        description={t("plans_create.header.description")}
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/plans"))}
        style={{ cursor: "pointer" }}
      />
      
      <div className="panel" style={{ maxWidth: 720 }}>
        
        {/* Step 1: Product Information */}
        <div className="action-section" style={{ border: "none", paddingBottom: 0 }}>
          <div className="action-section-title">{t("plans_create.sections.product")}</div>
          <div className="action-fields">
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <AutoCompleteInput
                id="plans-create-product"
                label={t("plans_create.fields.product_label")}
                value={form.productId}
                options={productOptions}
                placeholder={t("plans_create.fields.product_placeholder")}
                onSearch={searchProducts}
                onChange={(value) => setForm((p) => ({ ...p, productId: value }))}
              />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_create.fields.plan_name")}</Label>
              <Input className="action-input" value={form.planName} placeholder={t("plans_create.fields.plan_name_placeholder")} autoFocus
                onChange={(e) => setForm((p) => ({ ...p, planName: e.target.value }))} data-testid="plans-create-name" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_create.fields.plan_code")}</Label>
              <Input className="action-input" value={form.planCode} placeholder={t("plans_create.fields.plan_code_placeholder")}
                onChange={(e) => setForm((p) => ({ ...p, planCode: e.target.value }))} data-testid="plans-create-code" />
            </div>
            <div className="action-field" style={{ gridColumn: "span 2" }}>
              <Label className="action-label">{t("plans_create.fields.description")}</Label>
              <Input className="action-input" value={form.planDescription} placeholder={t("plans_create.fields.description_placeholder")}
                onChange={(e) => setForm((p) => ({ ...p, planDescription: e.target.value }))} data-testid="plans-create-description" />
            </div>
          </div>
        </div>

        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title" style={{ marginTop: 24 }}>{t("plans_edit.sections.plan_features")}</div>
          <div className="muted" style={{ marginBottom: 16, fontSize: 13 }}>
            {t("plans_create.plan_features.description")}
          </div>
          {!form.productId ? (
            <div className="muted" style={{ fontSize: 13 }}>{t("plans_create.empty.product_required_for_features")}</div>
          ) : productFeatures.length === 0 ? (
            <div className="muted" style={{ fontSize: 13 }}>{t("plans_create.empty.features")}</div>
          ) : (
            <PlanFeatureEditor rows={planFeatureRows} onChange={setPlanFeatureRows} t={t} disabled={loading} />
          )}
        </div>

        {/* Step 2: Pricing Configuration */}
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title" style={{ marginTop: 24 }}>{t("plans_create.sections.pricing")}</div>
          
          <div style={{ display: "flex", gap: "16px", marginBottom: "24px" }}>
             <Button type="button" 
                     variant={form.priceType === "flat" ? "default" : "secondary"}
                     onClick={() => setForm(p => ({...p, priceType: "flat", unitAmount: defaultMoneyInputForPriceType("flat")}))}
                     style={{ flex: 1 }}
                     data-testid="plans-create-type-flat">
               {t("plans_create.pricing.flat")}
             </Button>
             <Button type="button" 
                     variant={form.priceType === "usage" ? "default" : "secondary"}
                     onClick={() => setForm(p => ({...p, priceType: "usage", unitAmount: defaultMoneyInputForPriceType("usage")}))}
                     style={{ flex: 1 }}
                     data-testid="plans-create-type-usage">
               {t("plans_create.pricing.usage")}
             </Button>
          </div>

          <div className="action-fields">
            {form.priceType === "usage" && (
              <div className="action-field" style={{ gridColumn: "span 2" }}>
                <AutoCompleteInput
                  id="plan-create-meter"
                  label={t("plans_create.fields.meter_label")}
                  value={form.meterId}
                  options={meterOptions}
                  placeholder={t("plans_create.fields.meter_placeholder")}
                  onSearch={searchMeters}
                  onChange={(value) => setForm(p => ({ ...p, meterId: value }))}
                />
              </div>
            )}
            
            <div className="action-field">
              <Label className="action-label">{t("plans_create.fields.amount_label")}</Label>
              <Input className="action-input" type="text" value={form.unitAmount} min={0} step={moneyInputStepForPriceType(form.priceType)} inputMode="decimal"
                onChange={(e) => setForm((p) => ({ ...p, unitAmount: e.target.value }))} data-testid="plans-create-amount" />
            </div>
            <div className="action-field">
            <AutoCompleteInput
              id="plans-create-currency"
              label={<>{t("plans_create.fields.currency_label")} <HelpHint text={currencyHint} /></>}
              value={form.currency}
              options={currencyOptions}
              placeholder={currenciesLoading ? t("common.loading") : undefined}
              onChange={(value) => setForm((p) => ({ ...p, currency: value }))}
            />
            </div>
            
            <div className="action-field">
              <Label className="action-label">{t("plans_create.fields.interval_label")}</Label>
              <Select value={form.billingInterval} onValueChange={(value) => setForm((p) => ({ ...p, billingInterval: value }))}>
                <SelectTrigger className="action-select" data-testid="plans-create-interval">
                  <SelectValue placeholder={t("plans_create.fields.interval_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="month">{t("plans_create.fields.interval_options.month")}</SelectItem>
                  <SelectItem value="year">{t("plans_create.fields.interval_options.year")}</SelectItem>
                  <SelectItem value="week">{t("plans_create.fields.interval_options.week")}</SelectItem>
                  <SelectItem value="day">{t("plans_create.fields.interval_options.day")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_create.fields.interval_count_label")}</Label>
              <Input className="action-input" type="number" min={1} value={form.billingIntervalCount}
                onChange={(e) => setForm((p) => ({ ...p, billingIntervalCount: Number(e.target.value || 1) }))} data-testid="plans-create-interval-count" />
            </div>
          </div>
          
          {validation.length > 0 && <div className="inline-error" style={{ marginTop: 16 }}>{validation.join(" ")}</div>}
          
          <div className="action-buttons" style={{ marginTop: 24, justifyContent: "space-between" }}>
            <Button variant="outline" onClick={() => navigate(orgPath("/plans"))} data-testid="plans-create-cancel">{t("common.cancel")}</Button>
            <Button variant="default" disabled={loading || validation.length > 0} onClick={handleCreate} data-testid="plans-create-submit">
              {loading ? t("common.creating") : t("plans_create.actions.save")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
