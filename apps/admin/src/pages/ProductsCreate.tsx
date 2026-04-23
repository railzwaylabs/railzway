import { useEffect, useState, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { useForm, useFieldArray, useWatch, FormProvider, useFormContext } from "react-hook-form"
import { Plus, Trash2, Package, Layers, Wrench } from "lucide-react"

import PageHeader from "../components/PageHeader"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { useCurrencies } from "../lib/reference"
import AutoCompleteInput from "../components/AutoCompleteInput"
import type { CreateProductRequest, Feature, Meter, CreateProductPlanInput, CreateProductPlanPriceInput } from "../lib/types"
import { DEFAULT_MONEY_INPUT, defaultMoneyInputForPriceType, isNonNegativeMoneyInput, moneyInputDecimalsForPriceType, moneyInputStepForPriceType, moneyInputToCents, optionalMoneyInputToCents } from "../lib/money"

type ProductAmountFormInput = {
  currency: string
  unit_amount_cents: string
  minimum_amount_cents?: string
  maximum_amount_cents?: string
  effective_from?: string
  effective_to?: string
}

type ProductTierFormInput = {
  tier_mode: string
  start_quantity: number
  end_quantity?: number | string
  unit_amount_cents?: string
  flat_amount_cents?: string
  unit: string
}

type ProductPriceFormInput = Omit<CreateProductPlanPriceInput, "amounts" | "tiers"> & {
  amounts?: ProductAmountFormInput[]
  tiers?: ProductTierFormInput[]
}

type ProductPlanFormInput = Omit<CreateProductPlanInput, "prices"> & {
  prices?: ProductPriceFormInput[]
}

type FormValues = Omit<CreateProductRequest, "plans"> & {
  selected_features: string[];
  plans?: ProductPlanFormInput[];
}

function toInt(value: unknown, fallback = 0) {
  if (value === "" || value == null) return fallback
  const parsed = Number.parseInt(String(value), 10)
  return Number.isFinite(parsed) ? parsed : fallback
}

function toFloat(value: unknown, fallback = 0) {
  if (value === "" || value == null) return fallback
  const parsed = Number.parseFloat(String(value))
  return Number.isFinite(parsed) ? parsed : fallback
}

function toOptionalFloat(value: unknown) {
  if (value === "" || value == null) return undefined
  return toFloat(value)
}

function normalizeCreateProductPayload(data: FormValues): CreateProductRequest {
  return {
    code: data.code.trim(),
    name: data.name.trim(),
    description: data.description?.trim() || undefined,
    active: data.active,
    idempotency_key: data.idempotency_key,
    feature_ids: data.selected_features,
    plans: (data.plans ?? []).map((plan) => ({
      code: plan.code.trim(),
      name: plan.name.trim(),
      description: plan.description?.trim() || undefined,
      active: plan.active,
      prices: (plan.prices ?? []).map((price) => ({
        code: price.code.trim(),
        name: price.name?.trim() || undefined,
        description: price.description?.trim() || undefined,
        active: price.active,
        price_type: price.price_type,
        billing_interval: price.billing_interval,
        billing_interval_count: toInt(price.billing_interval_count, 1),
        meter_id: price.meter_id?.trim() || undefined,
        amounts: (price.amounts ?? []).map((amount) => ({
          currency: amount.currency.trim(),
          unit_amount_cents: moneyInputToCents(amount.unit_amount_cents, moneyInputDecimalsForPriceType(price.price_type)),
          minimum_amount_cents: optionalMoneyInputToCents(amount.minimum_amount_cents),
          maximum_amount_cents: optionalMoneyInputToCents(amount.maximum_amount_cents),
          effective_from: amount.effective_from || undefined,
          effective_to: amount.effective_to || undefined,
        })),
        tiers: (price.tiers ?? []).map((tier) => ({
          tier_mode: tier.tier_mode,
          start_quantity: toFloat(tier.start_quantity),
          end_quantity: toOptionalFloat(tier.end_quantity),
          unit_amount_cents: optionalMoneyInputToCents(tier.unit_amount_cents, moneyInputDecimalsForPriceType("usage")),
          flat_amount_cents: optionalMoneyInputToCents(tier.flat_amount_cents),
          unit: tier.unit?.trim() || "unit",
        })),
      })),
    })),
  }
}

export default function ProductsCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const { options: currencyOptions } = useCurrencies()

  const [bootstrap, setBootstrap] = useState<{ features: Feature[]; meters: Meter[] }>({
    features: [],
    meters: []
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mirrorPrimaryPlan, setMirrorPrimaryPlan] = useState(true)

  const methods = useForm<FormValues>({
    defaultValues: {
      code: "",
      name: "",
      description: "",
      active: true,
      idempotency_key: crypto.randomUUID(),
      selected_features: [],
      plans: [
        {
          code: "",
          name: "",
          active: true,
          prices: [
            {
              code: "",
              name: "",
              price_type: "flat",
              billing_interval: "month",
              billing_interval_count: 1,
              active: true,
              amounts: [{ currency: "USD", unit_amount_cents: DEFAULT_MONEY_INPUT }]
            }
          ]
        }
      ]
    }
  })

  const { register, control, handleSubmit, setValue, watch, formState: { errors } } = methods
  const productName = watch("name")
  const productCode = watch("code")

  const { fields: planFields, append: appendPlan, remove: removePlan } = useFieldArray({
    control,
    name: "plans"
  })

  useEffect(() => {
    const loadBootstrap = async () => {
      try {
        const [featuresResp, metersResp] = await Promise.all([
          api.features.list({ page_size: 200 }),
          api.meters.list({ page_size: 200 })
        ])
        setBootstrap({
          features: featuresResp.features || [],
          meters: metersResp.meters || []
        })
      } catch (err) {
        console.error("Failed to load bootstrap data", err)
      } finally {
        setLoading(false)
      }
    }
    void loadBootstrap()
  }, [])

  const meterOptions = useMemo(() => 
    bootstrap.meters.map(m => ({ value: m.id, label: `${m.name} (${m.code})` })),
    [bootstrap.meters]
  )

  useEffect(() => {
    if (planFields.length > 1 && mirrorPrimaryPlan) {
      setMirrorPrimaryPlan(false)
    }
  }, [mirrorPrimaryPlan, planFields.length])

  useEffect(() => {
    if (!mirrorPrimaryPlan || planFields.length !== 1) return
    setValue("plans.0.name", productName?.trim() ?? "", { shouldDirty: false, shouldTouch: false, shouldValidate: false })
    setValue("plans.0.code", productCode?.trim() ?? "", { shouldDirty: false, shouldTouch: false, shouldValidate: false })
  }, [mirrorPrimaryPlan, planFields.length, productName, productCode, setValue])

  const onSubmit = async (data: FormValues) => {
    try {
      setSaving(true)
      setError(null)

      const payload = normalizeCreateProductPayload(data)
      
      await api.products.create(payload)
      navigate(orgPath("/products"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("products_create.toast.create_failed"))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <div className="p-8 text-center muted">{t("common.loading")}...</div>
  }

  const selectedFeatures = watch("selected_features")
  const toggleFeature = (id: string) => {
    const next = selectedFeatures.includes(id)
      ? selectedFeatures.filter((f: string) => f !== id)
      : [...selectedFeatures, id]
    setValue("selected_features", next)
  }

  return (
    <FormProvider {...methods}>
      <div className="page-content">
        <PageHeader
          title={t("products_create.header.title")}
          description={t("products_create.header.description")}
        />

        <form onSubmit={handleSubmit(onSubmit)}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 340px", gap: "24px", alignItems: "start" }}>
            <div style={{ display: "flex", flexDirection: "column", gap: "32px" }}>
              
              {/* Section 1: Product Basic Info */}
              <div className="panel p-6">
                <div className="flex items-center gap-2 mb-6 border-b pb-4">
                  <Package className="w-5 h-5 text-primary" />
                  <h3 className="font-semibold text-lg">{t("products_create.sections.product")}</h3>
                </div>
                <p className="mb-4 text-sm muted">{t("products_create.hints.product_identity")}</p>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>{t("products_create.fields.product_code")} *</Label>
                    <Input {...register("code", { required: true })} placeholder={t("products_create.placeholders.product_code")} data-testid="products-create-code" />
                    {errors.code && <span className="text-destructive text-xs">{t("products_create.validation.required")}</span>}
                  </div>
                  <div className="space-y-2">
                    <Label>{t("products_create.fields.product_name")} *</Label>
                    <Input {...register("name", { required: true })} placeholder={t("products_create.placeholders.product_name")} data-testid="products-create-name" />
                    {errors.name && <span className="text-destructive text-xs">{t("products_create.validation.required")}</span>}
                  </div>
                  <div className="col-span-2 space-y-2">
                    <Label>{t("products_create.fields.description")}</Label>
                    <Input {...register("description")} placeholder={t("plans_create.fields.description_placeholder")} data-testid="products-create-description" />
                  </div>
                </div>
              </div>

              {/* Section 2: Plans Aggregate Setup */}
              <div className="space-y-6">
                <div className="flex items-center justify-between px-2">
                  <div className="flex items-center gap-2">
                    <Layers className="w-5 h-5 text-primary" />
                    <h3 className="font-semibold text-lg">{t("products_create.sections.plans")}</h3>
                  </div>
                  <Button 
                    type="button" 
                    variant="outline" 
                    size="sm"
                    onClick={() => {
                      setMirrorPrimaryPlan(false)
                      appendPlan({ code: "", name: "", active: true, prices: [] })
                    }}
                    data-testid="products-create-add-plan"
                  >
                    <Plus className="w-4 h-4 mr-1" />
                    {t("plans.actions.create")}
                  </Button>
                </div>
                <div className="px-2 text-sm muted">{t("products_create.hints.product_plan_boundary")}</div>

                {planFields.map((plan: ProductPlanFormInput & { id: string }, index: number) => (
                  <PlanFormItem 
                    key={plan.id}
                    index={index}
                    remove={() => removePlan(index)}
                    currencyOptions={currencyOptions}
                    meterOptions={meterOptions}
                    t={t}
                    isPrimary={index === 0}
                    canMirror={planFields.length === 1}
                    mirrorPlanIdentity={mirrorPrimaryPlan}
                    onToggleMirrorPlanIdentity={setMirrorPrimaryPlan}
                  />
                ))}

                {planFields.length === 0 && (
                  <div className="p-8 border-2 border-dashed rounded-lg text-center muted bg-white">
                    {t("products_create.hints.no_plans")}
                  </div>
                )}
              </div>

              {error && <div className="p-4 bg-destructive/10 text-destructive rounded-md text-sm font-medium">{error}</div>}

              <div className="flex items-center gap-3 pt-4 sticky bottom-4 bg-background/80 backdrop-blur-sm p-4 rounded-lg shadow-sm border border-border/50">
                <Button type="submit" size="lg" disabled={saving} data-testid="products-create-submit">
                  {saving ? t("common.creating") : t("products_create.actions.submit")}
                </Button>
                <Button type="button" variant="secondary" size="lg" onClick={() => navigate(orgPath("/products"))} disabled={saving} data-testid="products-create-cancel">
                  {t("common.cancel")}
                </Button>
              </div>
            </div>

            {/* Sidebar: Feature Selection */}
            <div className="space-y-6">
              <div className="panel p-5">
                <div className="flex items-center gap-2 mb-4 border-b pb-3">
                   <Wrench className="w-4 h-4 text-primary" />
                  <h3 className="font-medium">{t("features.header.title")}</h3>
                </div>
                <p className="text-xs muted mb-4">{t("features.header.description")}</p>
                
                {bootstrap.features.length === 0 ? (
                  <div className="p-4 bg-subtle rounded border border-dashed text-center text-xs muted">
                    {t("features.empty_title")}
                  </div>
                ) : (
                  <div className="space-y-2 max-h-[400px] overflow-y-auto pr-2">
                    {bootstrap.features.map(f => (
                      <div 
                        key={f.id}
                        onClick={() => toggleFeature(f.id)}
                        className={`flex items-center gap-3 p-3 rounded-md border cursor-pointer transition-colors ${
                          selectedFeatures.includes(f.id) 
                            ? "bg-primary/5 border-primary" 
                            : "bg-white border-border hover:border-primary/50"
                        }`}
                      >
                        <input 
                          type="checkbox" 
                          checked={selectedFeatures.includes(f.id)} 
                          readOnly
                          className="rounded border-gray-300 pointer-events-none"
                          data-testid={`feature-checkbox-${f.code}`}
                        />
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium truncate">{f.name}</div>
                          <div className="text-[10px] muted uppercase tracking-tight">{f.code}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="panel p-5 bg-primary/5 border-primary/20">
                <h4 className="text-sm font-semibold mb-2">{t("products_create.fields.idempotency")}</h4>
                <p className="text-xs muted mb-3">{t("products_create.hints.idempotency")}</p>
                <Input {...register("idempotency_key")} className="text-[10px] font-mono" readOnly />
              </div>
            </div>
          </div>
        </form>
      </div>
    </FormProvider>
  )
}

function PlanFormItem({
  index,
  remove,
  currencyOptions,
  meterOptions,
  t,
  isPrimary,
  canMirror,
  mirrorPlanIdentity,
  onToggleMirrorPlanIdentity,
}: any) {
  const { control, register } = useFormContext<FormValues>()
  const { fields: priceFields, append: appendPrice, remove: removePrice } = useFieldArray({
    control,
    name: `plans.${index}.prices`
  })

  return (
    <div className="panel p-6 border-l-4 border-l-primary/40 relative overflow-hidden bg-white shadow-sm">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold text-sm">
            {index + 1}
          </div>
          <h4 className="font-semibold">{t("products_create.sections.plan_item", { number: index + 1 })}</h4>
        </div>
        <Button 
          type="button" 
          variant="ghost" 
          size="sm" 
          onClick={remove}
          className="text-destructive hover:bg-destructive/10"
        >
          <Trash2 className="w-4 h-4 mr-1" />
          {t("common.remove")}
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-8">
        <div className="col-span-2 text-xs muted">{t("products_create.hints.plan_identity")}</div>
        {isPrimary && canMirror ? (
          <label className="col-span-2 flex items-center gap-2 rounded-md border border-border bg-subtle/30 px-3 py-2 text-xs">
            <input
              type="checkbox"
              checked={mirrorPlanIdentity}
              onChange={(event) => onToggleMirrorPlanIdentity(event.target.checked)}
            />
            <span>{t("products_create.hints.mirror_primary_plan")}</span>
          </label>
        ) : null}
        <div className="space-y-2">
          <Label>{t("products_create.fields.plan_name")} *</Label>
          <Input
            {...register(`plans.${index}.name`, { required: true })}
            placeholder={t("products_create.placeholders.plan_name")}
            data-testid={`plans-name-${index}`}
            disabled={isPrimary && canMirror && mirrorPlanIdentity}
          />
        </div>
        <div className="space-y-2">
          <Label>{t("products_create.fields.plan_code")} *</Label>
          <Input
            {...register(`plans.${index}.code`, { required: true })}
            placeholder={t("products_create.placeholders.plan_code")}
            data-testid={`plans-code-${index}`}
            disabled={isPrimary && canMirror && mirrorPlanIdentity}
          />
        </div>
      </div>

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h5 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">{t("plans_create.sections.pricing")}</h5>
          <Button 
            type="button" 
            variant="ghost" 
            size="sm"
            onClick={() => appendPrice({ 
              code: "", 
              price_type: "flat", 
              billing_interval: "month", 
              billing_interval_count: 1,
              active: true,
              amounts: [{ currency: "USD", unit_amount_cents: DEFAULT_MONEY_INPUT }]
            })}
            className="h-8 py-0 px-2 text-primary"
          >
            <Plus className="w-3 h-3 mr-1" />
            {t("products_create.actions.add_price")}
          </Button>
        </div>

        <div className="space-y-4 pl-4 border-l-2 border-border/50">
          {priceFields.map((price: ProductPriceFormInput & { id: string }, pIndex: number) => (
            <PriceFormItem 
              key={price.id}
              planIndex={index}
              priceIndex={pIndex}
              remove={() => removePrice(pIndex)}
              currencyOptions={currencyOptions}
              meterOptions={meterOptions}
              t={t}
            />
          ))}

          {priceFields.length === 0 && (
            <div className="p-4 bg-muted/30 rounded text-center text-xs italic muted">
              {t("products_create.hints.price_recommendation")}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function PriceFormItem({ planIndex, priceIndex, remove, currencyOptions, meterOptions, t }: any) {
  const { control, register, setValue, watch } = useFormContext<FormValues>()
  const pricePath = `plans.${planIndex}.prices.${priceIndex}` as const
  const priceType = useWatch({
    control,
    name: `${pricePath}.price_type`
  })

  const { fields: tierFields, append: appendTier, remove: removeTier } = useFieldArray({
    control,
    name: `${pricePath}.tiers`
  })

  const { fields: amountFields, append: appendAmount, remove: removeAmount } = useFieldArray({
    control,
    name: `${pricePath}.amounts`
  })

  useEffect(() => {
    if (priceType !== 'tiered' && amountFields.length === 0) {
      appendAmount({ currency: "USD", unit_amount_cents: defaultMoneyInputForPriceType(priceType) })
    }
  }, [priceType, amountFields.length, appendAmount])

  return (
    <div className="bg-subtle/50 border rounded-lg p-4 relative group hover:border-primary/30 transition-all">
      <Button 
        type="button"
        variant="ghost"
        size="sm"
        onClick={remove}
        className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity h-6 w-6 p-0 text-destructive"
      >
        <Trash2 className="w-3.5 h-3.5" />
      </Button>

      <div className="grid grid-cols-12 gap-4">
        <div className="col-span-3 space-y-2">
          <Label className="text-xs">{t("plans_edit.price_fields.type")}</Label>
          <Select 
            defaultValue="flat"
            value={priceType}
            onValueChange={(v) => {
              setValue(`${pricePath}.price_type`, v)
              if (v === 'tiered') {
                setValue(`${pricePath}.amounts`, [])
                if (tierFields.length === 0) appendTier({ tier_mode: "volume", start_quantity: 0, unit: "unit" })
              } else {
                setValue(`${pricePath}.tiers`, [])
                if (amountFields.length === 0) appendAmount({ currency: "USD", unit_amount_cents: defaultMoneyInputForPriceType(v) })
                else {
                  amountFields.forEach((_, aIndex) => {
                    setValue(`${pricePath}.amounts.${aIndex}.unit_amount_cents`, defaultMoneyInputForPriceType(v))
                  })
                }
              }
            }}
          >
            <SelectTrigger className="h-9 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="flat">{t("plans_edit.price_fields.type_options.flat")}</SelectItem>
              <SelectItem value="usage">{t("plans_edit.price_fields.type_options.usage")}</SelectItem>
              <SelectItem value="tiered">{t("plans_edit.price_fields.type_options.tiered")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="col-span-3 space-y-2">
          <Label className="text-xs">{t("plans_edit.price_fields.interval")}</Label>
          <div className="flex gap-1">
            <Input {...register(`${pricePath}.billing_interval_count`)} type="number" className="h-9 w-12 text-xs px-1 text-center" />
            <Select defaultValue="month" onValueChange={(v) => setValue(`${pricePath}.billing_interval`, v)}>
              <SelectTrigger className="h-9 text-xs flex-1">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="day">{t("plans_edit.price_fields.interval_options.day")}</SelectItem>
                <SelectItem value="week">{t("plans_edit.price_fields.interval_options.week")}</SelectItem>
                <SelectItem value="month">{t("plans_edit.price_fields.interval_options.month")}</SelectItem>
                <SelectItem value="year">{t("plans_edit.price_fields.interval_options.year")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="col-span-6 space-y-2">
           <Label className="text-xs">{t("products_create.fields.price_code")}</Label>
           <Input
             {...register(`${pricePath}.code`, { required: true })}
             className="h-9 text-xs"
             placeholder={t("products_create.placeholders.price_code")}
             data-testid={`products-plan-${planIndex}-price-${priceIndex}-code`}
           />
        </div>

        {(priceType === 'usage' || priceType === 'tiered') && (
          <div className="col-span-12 space-y-2 pt-1">
             <Label className="text-xs flex items-center gap-1">
               {t("plans_edit.price_fields.meter")} <span className="text-primary font-bold">*</span>
               <span className="text-[10px] muted normal-case font-normal">({t("features_create.hints.meter_required", { type: priceType })})</span>
             </Label>
             <AutoCompleteInput
               id={`meter-${planIndex}-${priceIndex}`}
               label={null}
               value={watch(`${pricePath}.meter_id`) || ""}
               options={meterOptions}
               onChange={(v: string) => setValue(`${pricePath}.meter_id`, v)}
             />
          </div>
        )}

         {priceType === 'tiered' && (
          <div className="col-span-12 space-y-3 pt-2">
             <div className="flex items-center justify-between">
                <Label className="text-xs">{t("products_create.fields.pricing_tiers")}</Label>
                 <Button 
                  type="button" 
                  variant="outline" 
                   size="sm" 
                  className="h-6 w-6 p-0"
                  onClick={() => appendTier({ tier_mode: "volume", start_quantity: 0, unit: "unit" })}
                >
                  <Plus className="w-3 h-3" />
                </Button>
             </div>
              <div className="space-y-2">
                {tierFields.map((tier: { id: string; tier_mode: string; start_quantity: number }, tIndex: number) => (
                  <div key={tier.id} className="flex gap-2 items-end">
                    <div className="flex-1 space-y-1">
                      <Label className="text-[10px] muted">{t("products_create.fields.tier_start")}</Label>
                      <Input {...register(`${pricePath}.tiers.${tIndex}.start_quantity`)} type="number" className="h-8 text-xs" data-testid={`products-plan-${planIndex}-price-${priceIndex}-tier-${tIndex}-start`} />
                    </div>
                    <div className="flex-1 space-y-1">
                      <Label className="text-[10px] muted">{t("products_create.fields.tier_end")}</Label>
                      <Input {...register(`${pricePath}.tiers.${tIndex}.end_quantity`)} type="number" className="h-8 text-xs" data-testid={`products-plan-${planIndex}-price-${priceIndex}-tier-${tIndex}-end`} />
                    </div>
                    <div className="flex-1 space-y-1">
                      <Label className="text-[10px] muted">{t("products_create.fields.tier_base_rate")}</Label>
                      <Input {...register(`${pricePath}.tiers.${tIndex}.unit_amount_cents`, { validate: (value) => value == null || value === "" || isNonNegativeMoneyInput(value, moneyInputDecimalsForPriceType("usage")) })} type="text" min={0} step={moneyInputStepForPriceType("usage")} inputMode="decimal" className="h-8 text-xs" data-testid={`products-plan-${planIndex}-price-${priceIndex}-tier-${tIndex}-unit-amount`} />
                    </div>
                     <Button 
                      type="button" 
                      variant="ghost" 
                      size="sm" 
                      onClick={() => removeTier(tIndex)}
                      className="h-8 w-8 p-0 text-destructive"
                    >
                      <Trash2 className="w-3 h-3" />
                    </Button>
                  </div>
                ))}
             </div>
          </div>
        )}

        {priceType !== 'tiered' && (
          <div className="col-span-12 space-y-2 pt-2">
              <Label className="text-xs">{t("products_create.fields.amount")}</Label>
             {amountFields.map((amount: { id: string; currency: string; unit_amount_cents: string }, aIndex: number) => (
               <div key={amount.id} className="flex gap-2">
                  <div className="w-24">
                     <AutoCompleteInput
                      id={`currency-${planIndex}-${priceIndex}-${aIndex}`}
                      label={null}
                      value={watch(`${pricePath}.amounts.${aIndex}.currency`) || ""}
                      options={currencyOptions}
                      onChange={(v: string) => setValue(`${pricePath}.amounts.${aIndex}.currency`, v)}
                    />
                  </div>
                  <div className="flex-1">
                    <Input 
                      {...register(`${pricePath}.amounts.${aIndex}.unit_amount_cents`, { validate: (value) => isNonNegativeMoneyInput(value, moneyInputDecimalsForPriceType(priceType)) })}
                      type="text"
                      min={0}
                      step={moneyInputStepForPriceType(priceType)}
                      inputMode="decimal"
                      className="h-9 text-xs" 
                      placeholder={t("products_create.placeholders.amount_cents")}
                      data-testid={`products-plan-${planIndex}-price-${priceIndex}-amount-${aIndex}`}
                    />
                  </div>
                   {amountFields.length > 1 && (
                    <Button type="button" variant="ghost" size="sm" onClick={() => removeAmount(aIndex)} className="h-9 w-9 p-0">
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  )}
               </div>
             ))}
             <Button 
                type="button" 
                variant="link" 
                size="sm" 
                onClick={() => appendAmount({ currency: "USD", unit_amount_cents: defaultMoneyInputForPriceType(priceType) })}
                className="h-6 text-[10px] p-0"
              >
               {t("products_create.actions.add_multi_currency")}
             </Button>
          </div>
        )}
      </div>
    </div>
  )
}
