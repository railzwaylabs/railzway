import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate, useParams } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"

import { Badge } from "../components/ui/badge"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import { formatCurrency } from "../lib/format"
import { currencyHint } from "../lib/hints"
import { useCurrencies } from "../lib/reference"
import { isCurrencyCode } from "../lib/validation"
import { formatDate, normalizeDate, rfc3339Hint } from "../lib/display"
import { statusClass } from "../lib/status"
import type { Plan, PlanPrice } from "../lib/types"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function PlansEdit() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies()

  const [plan, setPlan] = useState<Plan | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [meterOptions, setMeterOptions] = useState<Array<{ value: string; label: string }>>([])

  const [updatePlanForm, setUpdatePlanForm] = useState({
    name: "",
    description: "",
    active: ""
  })
  const [createPriceForm, setCreatePriceForm] = useState({
    code: "", name: "", description: "", priceType: "",
    billingInterval: "month", billingIntervalCount: 1,
    aggregateUsage: "", billingUnit: "", meterId: "", meterCode: "", active: true
  })
  const [createAmountForm, setCreateAmountForm] = useState({
    priceId: "", currency: "USD", unitAmountCents: 0,
    minimumAmountCents: "", maximumAmountCents: "",
    effectiveFrom: "", effectiveTo: ""
  })
  const [createTierForm, setCreateTierForm] = useState({
    priceId: "", tierMode: "volume", startQuantity: 0,
    endQuantity: "", unitAmountCents: "", flatAmountCents: "", unit: ""
  })
  const [activeAmountPriceId, setActiveAmountPriceId] = useState("")
  const [activeTierPriceId, setActiveTierPriceId] = useState("")

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

  const loadData = useCallback(async () => {
    if (!id) return
    try {
      setLoading(true)
      // fetch options
      const metersResp = await api.meters.list({ page_size: 50, active: "true" }).catch(() => null)
      if (metersResp) {
        setMeterOptions(metersResp.meters.map((meter) => ({
          value: meter.id, label: `${meter.name} · ${meter.code}`
        })))
      }
      
      // We list and find by ID because there's no api.plans.get
      let found: Plan | undefined
      let pageToken: string | undefined
      while (!found) {
        const resp = await api.plans.list({ page_token: pageToken, page_size: 50 })
        found = resp.plans.find(p => p.id === id)
        pageToken = resp.next_page_token
        if (!resp.has_more && !pageToken) break
      }

      if (found) {
        setPlan(found)
        setUpdatePlanForm({ name: found.name, description: found.description || "", active: String(found.active) })
      }
    } catch (err) {
      toast.error(t("plans_edit.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [id, t])

  useEffect(() => { void loadData() }, [loadData])

  const updateValidation = useMemo(() => {
    const e: string[] = []
    if (!updatePlanForm.name.trim() && !updatePlanForm.description.trim() && updatePlanForm.active === "") {
      e.push(t("plans_edit.validation.no_changes"))
    }
    return e
  }, [updatePlanForm, t])

  const createPriceValidation = useMemo(() => {
    const e: string[] = []
    if (!createPriceForm.code.trim()) e.push(t("plans_edit.validation.price_code_required"))
    if (!createPriceForm.priceType.trim()) e.push(t("plans_edit.validation.price_type_required"))
    if (!createPriceForm.billingInterval.trim()) e.push(t("plans_edit.validation.interval_required"))
    if (!Number.isFinite(createPriceForm.billingIntervalCount) || createPriceForm.billingIntervalCount < 1) e.push(t("plans_edit.validation.interval_count"))
    if (createPriceForm.priceType.trim() && createPriceForm.priceType !== "flat" && !createPriceForm.meterId.trim() && !createPriceForm.meterCode.trim()) {
      e.push(t("plans_edit.validation.meter_required"))
    }
    return e
  }, [createPriceForm, t])

  const createAmountValidation = useMemo(() => {
    const e: string[] = []
    if (!createAmountForm.priceId.trim()) e.push(t("plans_edit.validation.amount_price_required"))
    if (!createAmountForm.currency.trim()) e.push(t("plans_edit.validation.currency_required"))
    else if (!isCurrencyCode(createAmountForm.currency)) e.push(t("plans_edit.validation.currency_invalid"))
    if (!Number.isFinite(createAmountForm.unitAmountCents) || createAmountForm.unitAmountCents < 0) e.push(t("plans_edit.validation.unit_amount_min"))
    if (createAmountForm.effectiveFrom && createAmountForm.effectiveTo) {
      const start = new Date(createAmountForm.effectiveFrom)
      const end = new Date(createAmountForm.effectiveTo)
      if (!isNaN(start.getTime()) && !isNaN(end.getTime()) && end < start) {
        e.push(t("plans_edit.validation.effective_range"))
      }
    }
    const min = createAmountForm.minimumAmountCents ? Number.parseInt(createAmountForm.minimumAmountCents) : null
    const max = createAmountForm.maximumAmountCents ? Number.parseInt(createAmountForm.maximumAmountCents) : null
    if (min !== null && max !== null && max < min) e.push(t("plans_edit.validation.min_max"))
    return e
  }, [createAmountForm, t])

  const createTierValidation = useMemo(() => {
    const e: string[] = []
    if (!createTierForm.priceId.trim()) e.push(t("plans_edit.validation.tier_price_required"))
    if (!createTierForm.tierMode.trim()) e.push(t("plans_edit.validation.tier_mode_required"))
    if (!Number.isFinite(createTierForm.startQuantity) || createTierForm.startQuantity < 0) e.push(t("plans_edit.validation.tier_start_min"))
    if (createTierForm.endQuantity) {
      const end = Number.parseFloat(createTierForm.endQuantity)
      if (!isNaN(end) && end < createTierForm.startQuantity) e.push(t("plans_edit.validation.tier_end_min"))
    }
    if (!createTierForm.unitAmountCents.trim() && !createTierForm.flatAmountCents.trim()) e.push(t("plans_edit.validation.tier_amount_required"))
    return e
  }, [createTierForm, t])

  const handleUpdate = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      const payload: { name?: string; description?: string; active?: boolean } = {}
      if (updatePlanForm.name.trim()) payload.name = updatePlanForm.name.trim()
      if (updatePlanForm.description.trim()) payload.description = updatePlanForm.description.trim()
      if (updatePlanForm.active !== "") payload.active = updatePlanForm.active === "true"
        
      const resp = await api.plans.update(id, payload)
      toast.success(t("plans_edit.toast.updated"), resp.id)
      void loadData()
    } catch (err) {
      toast.error(t("plans_edit.toast.update_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, updatePlanForm, loadData, t])

  const handleCreatePrice = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      const resp = await api.plans.createPrice(id, {
        code: createPriceForm.code.trim(),
        name: createPriceForm.name.trim() || undefined,
        description: createPriceForm.description.trim() || undefined,
        price_type: createPriceForm.priceType.trim(),
        billing_interval: createPriceForm.billingInterval.trim(),
        billing_interval_count: createPriceForm.billingIntervalCount,
        aggregate_usage: createPriceForm.aggregateUsage.trim() || undefined,
        billing_unit: createPriceForm.billingUnit.trim() || undefined,
        meter_id: createPriceForm.meterId.trim() || undefined,
        meter_code: createPriceForm.meterCode.trim() || undefined,
        active: createPriceForm.active
      })
      toast.success(t("plans_edit.toast.price_created"), resp.id)
      setCreatePriceForm(p => ({ ...p, code: "", name: "", description: "", priceType: "", aggregateUsage: "", billingUnit: "", meterId: "", meterCode: "" }))
      void loadData()
    } catch(err) {
      toast.error(t("plans_edit.toast.price_create_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, createPriceForm, loadData, t])

  const handleCreateAmount = useCallback(async () => {
    try {
      setActionLoading(true)
      const resp = await api.plans.createAmount(createAmountForm.priceId, {
        currency: createAmountForm.currency.trim(),
        unit_amount_cents: createAmountForm.unitAmountCents,
        minimum_amount_cents: createAmountForm.minimumAmountCents ? Number.parseInt(createAmountForm.minimumAmountCents, 10) : undefined,
        maximum_amount_cents: createAmountForm.maximumAmountCents ? Number.parseInt(createAmountForm.maximumAmountCents, 10) : undefined,
        effective_from: createAmountForm.effectiveFrom ? normalizeDate(createAmountForm.effectiveFrom) : undefined,
        effective_to: createAmountForm.effectiveTo ? normalizeDate(createAmountForm.effectiveTo) : undefined
      })
      toast.success(t("plans_edit.toast.amount_created"), resp.id)
      setCreateAmountForm(p => ({ ...p, priceId: "", currency: "USD", unitAmountCents: 0, minimumAmountCents: "", maximumAmountCents: "", effectiveFrom: "", effectiveTo: "" }))
      setActiveAmountPriceId("")
      void loadData()
    } catch(err) {
      toast.error(t("plans_edit.toast.amount_create_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [createAmountForm, loadData, t])

  const handleCreateTier = useCallback(async () => {
    try {
      setActionLoading(true)
      const resp = await api.plans.createTier(createTierForm.priceId, {
        tier_mode: createTierForm.tierMode.trim(),
        start_quantity: createTierForm.startQuantity,
        end_quantity: createTierForm.endQuantity ? Number.parseFloat(createTierForm.endQuantity) : undefined,
        unit_amount_cents: createTierForm.unitAmountCents ? Number.parseInt(createTierForm.unitAmountCents, 10) : undefined,
        flat_amount_cents: createTierForm.flatAmountCents ? Number.parseInt(createTierForm.flatAmountCents, 10) : undefined,
        unit: createTierForm.unit.trim()
      })
      toast.success(t("plans_edit.toast.tier_created"), resp.id)
      setCreateTierForm(p => ({ ...p, priceId: "", startQuantity: 0, endQuantity: "", unitAmountCents: "", flatAmountCents: "", unit: "" }))
      setActiveTierPriceId("")
      void loadData()
    } catch(err) {
      toast.error(t("plans_edit.toast.tier_create_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [createTierForm, loadData, t])

  const openAmountForm = useCallback((price: PlanPrice) => {
    const currency = price.amounts?.[0]?.currency || "USD"
    setActiveAmountPriceId(price.id)
    setActiveTierPriceId("")
    setCreateAmountForm({
      priceId: price.id,
      currency,
      unitAmountCents: 0,
      minimumAmountCents: "",
      maximumAmountCents: "",
      effectiveFrom: "",
      effectiveTo: "",
    })
  }, [])

  const openTierForm = useCallback((price: PlanPrice) => {
    setActiveTierPriceId(price.id)
    setActiveAmountPriceId("")
    setCreateTierForm({
      priceId: price.id,
      tierMode: "volume",
      startQuantity: 0,
      endQuantity: "",
      unitAmountCents: "",
      flatAmountCents: "",
      unit: price.billing_unit || "",
    })
  }, [])

  if (loading) return <div className="page-content"><div className="loader" /></div>

  return (
    <div className="page-content">
      <PageHeader 
        title={plan ? t("plans_edit.header.title_with_name", { name: plan.name }) : t("plans_edit.header.title")} 
        description={plan ? plan.code : ""} 
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/plans"))}
        style={{ cursor: "pointer" }}
      />
      <div className="action-panel">
        <div className="action-section">
          <div className="action-section-title">{t("plans_edit.sections.current_pricing")}</div>
          {plan?.prices && plan.prices.length > 0 ? (
            <div style={{ display: "grid", gap: 16 }}>
              {plan.prices.map((price) => {
                const interval = price.billing_interval_count > 1
                  ? `${price.billing_interval_count} ${price.billing_interval}s`
                  : price.billing_interval
                const currency = price.amounts?.[0]?.currency || "USD"
                return (
                  <div key={price.id} style={{ border: "1px solid var(--border)", borderRadius: 12, padding: 16, background: "var(--panel)" }}>
                    <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 12, marginBottom: 12 }}>
                      <div>
                        <div style={{ fontWeight: 600 }}>{price.name || price.code}</div>
                        <div className="muted" style={{ fontSize: 12 }}>
                          {price.code} • {price.price_type} • {interval}
                          {price.meter_code ? ` • meter ${price.meter_code}` : ""}
                        </div>
                      </div>
                        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                          <Button variant="outline" size="sm" onClick={() => openAmountForm(price)}>
                          {t("plans_edit.actions.add_amount")}
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => openTierForm(price)}>
                          {t("plans_edit.actions.add_tier")}
                        </Button>
                        <Badge className={`status-badge ${statusClass(price.active ? "active" : "inactive")}`}>
                          {price.active ? t("plans_edit.status.active") : t("plans_edit.status.inactive")}
                        </Badge>
                      </div>
                    </div>
                    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
                      <div>
                        <div className="muted" style={{ fontSize: 12, marginBottom: 6 }}>{t("plans_edit.sections.amounts")}</div>
                        {price.amounts && price.amounts.length > 0 ? (
                          <div style={{ display: "grid", gap: 6 }}>
                            {price.amounts.map((amount) => (
                              <div key={amount.id} style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                                <span style={{ fontWeight: 600 }}>{formatCurrency(amount.unit_amount_cents, amount.currency)}</span>
                                <span className="muted" style={{ fontSize: 12 }}>
                                  {formatDate(amount.effective_from)} → {formatDate(amount.effective_to)}
                                </span>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <span className="muted">{t("plans_edit.empty.amounts")}</span>
                        )}
                      </div>
                      <div>
                        <div className="muted" style={{ fontSize: 12, marginBottom: 6 }}>{t("plans_edit.sections.tiers")}</div>
                        {price.tiers && price.tiers.length > 0 ? (
                          <div style={{ display: "grid", gap: 6 }}>
                            {price.tiers.map((tier) => (
                              <div key={tier.id} style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                                <span>{tier.start_quantity}–{tier.end_quantity ?? "∞"} {tier.unit}</span>
                                <span className="muted" style={{ fontSize: 12 }}>
                                  {tier.unit_amount_cents != null ? formatCurrency(tier.unit_amount_cents, currency) : t("common.empty_dash")}
                                  {tier.flat_amount_cents != null ? ` + ${formatCurrency(tier.flat_amount_cents, currency)}` : ""}
                                </span>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <span className="muted">{t("plans_edit.empty.tiers")}</span>
                        )}
                      </div>
                    </div>
                    {activeAmountPriceId === price.id ? (
                      <div style={{ marginTop: 16, paddingTop: 16, borderTop: "1px dashed var(--line)" }}>
                        <div className="action-section-title" style={{ marginBottom: 12 }}>{t("plans_edit.amount_form.title")}</div>
                        <div className="action-fields">
                          <div className="action-field">
                            <AutoCompleteInput
                              id="plans-edit-amount-currency"
                              label={<>{t("plans_edit.amount_form.currency")} <HelpHint text={currencyHint} /></>}
                              value={createAmountForm.currency}
                              options={currencyOptions}
                              placeholder={currenciesLoading ? t("common.loading") : undefined}
                              onChange={(value) => setCreateAmountForm(p => ({ ...p, currency: value }))}
                            />
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.amount_form.unit_amount")}</Label>
                            <Input className="action-input" type="number" value={createAmountForm.unitAmountCents} onChange={e => setCreateAmountForm(p => ({ ...p, unitAmountCents: Number(e.target.value || 0) }))} />
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.amount_form.minimum")}</Label>
                            <Input className="action-input" type="number" value={createAmountForm.minimumAmountCents} onChange={e => setCreateAmountForm(p => ({ ...p, minimumAmountCents: e.target.value }))} />
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.amount_form.maximum")}</Label>
                            <Input className="action-input" type="number" value={createAmountForm.maximumAmountCents} onChange={e => setCreateAmountForm(p => ({ ...p, maximumAmountCents: e.target.value }))} />
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.amount_form.effective_from")} <HelpHint text={rfc3339Hint} /></Label>
                            <Input
                              className="action-input"
                              type="datetime-local"
                              value={createAmountForm.effectiveFrom}
                              onChange={e => setCreateAmountForm(p => ({ ...p, effectiveFrom: e.target.value }))}
                            />
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.amount_form.effective_to")} <HelpHint text={rfc3339Hint} /></Label>
                            <Input
                              className="action-input"
                              type="datetime-local"
                              min={createAmountForm.effectiveFrom || undefined}
                              value={createAmountForm.effectiveTo}
                              onChange={e => setCreateAmountForm(p => ({ ...p, effectiveTo: e.target.value }))}
                            />
                          </div>
                        </div>
                        {createAmountValidation.length > 0 && <div className="inline-error">{createAmountValidation.join(" ")}</div>}
                        <div className="action-buttons">
                          <Button variant="default" disabled={actionLoading || createAmountValidation.length > 0} onClick={handleCreateAmount}>
                            {actionLoading ? t("common.creating") : t("plans_edit.amount_form.create")}
                          </Button>
                          <Button variant="outline" onClick={() => setActiveAmountPriceId("")}>{t("common.cancel")}</Button>
                        </div>
                      </div>
                    ) : null}
                    {activeTierPriceId === price.id ? (
                      <div style={{ marginTop: 16, paddingTop: 16, borderTop: "1px dashed var(--line)" }}>
                        <div className="action-section-title" style={{ marginBottom: 12 }}>{t("plans_edit.tier_form.title")}</div>
                        <div className="action-fields">
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.tier_form.mode")}</Label>
                            <Select value={createTierForm.tierMode} onValueChange={(value) => setCreateTierForm(p => ({ ...p, tierMode: value }))}>
                              <SelectTrigger className="action-select">
                                <SelectValue placeholder={t("plans_edit.tier_form.mode_placeholder")} />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="volume">{t("plans_edit.tier_form.mode_options.volume")}</SelectItem>
                                <SelectItem value="graduated">{t("plans_edit.tier_form.mode_options.graduated")}</SelectItem>
                              </SelectContent>
                            </Select>
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.tier_form.quantities")}</Label>
                            <div style={{ display: "flex", gap: 10 }}>
                              <Input type="number" value={createTierForm.startQuantity} onChange={e => setCreateTierForm(p => ({ ...p, startQuantity: Number(e.target.value || 0) }))} />
                              <Input type="number" min={createTierForm.startQuantity} value={createTierForm.endQuantity} placeholder={t("plans_edit.tier_form.end_placeholder")} onChange={e => setCreateTierForm(p => ({ ...p, endQuantity: e.target.value }))} />
                            </div>
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.tier_form.amount")}</Label>
                            <div style={{ display: "flex", gap: 10 }}>
                              <Input type="number" value={createTierForm.unitAmountCents} placeholder={t("plans_edit.tier_form.unit_placeholder")} onChange={e => setCreateTierForm(p => ({ ...p, unitAmountCents: e.target.value }))} />
                              <Input type="number" value={createTierForm.flatAmountCents} placeholder={t("plans_edit.tier_form.flat_placeholder")} onChange={e => setCreateTierForm(p => ({ ...p, flatAmountCents: e.target.value }))} />
                            </div>
                          </div>
                          <div className="action-field">
                            <Label className="action-label">{t("plans_edit.tier_form.unit")}</Label>
                            <Input className="action-input" value={createTierForm.unit} onChange={e => setCreateTierForm(p => ({ ...p, unit: e.target.value }))} />
                          </div>
                        </div>
                        {createTierValidation.length > 0 && <div className="inline-error">{createTierValidation.join(" ")}</div>}
                        <div className="action-buttons">
                          <Button variant="default" disabled={actionLoading || createTierValidation.length > 0} onClick={handleCreateTier}>
                            {actionLoading ? t("common.creating") : t("plans_edit.tier_form.create")}
                          </Button>
                          <Button variant="outline" onClick={() => setActiveTierPriceId("")}>{t("common.cancel")}</Button>
                        </div>
                      </div>
                    ) : null}
                  </div>
                )
              })}
            </div>
          ) : (
            <div className="muted">{t("plans_edit.empty.prices")}</div>
          )}
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("plans_edit.sections.update_details")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.update_fields.name")}</Label>
              <Input className="action-input" value={updatePlanForm.name} onChange={(e) => setUpdatePlanForm(p => ({ ...p, name: e.target.value }))} data-testid="plans-edit-name" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.update_fields.description")}</Label>
              <Input className="action-input" value={updatePlanForm.description} onChange={(e) => setUpdatePlanForm(p => ({ ...p, description: e.target.value }))} data-testid="plans-edit-description" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.update_fields.status")}</Label>
              <Select
                value={toSelectValue(updatePlanForm.active)}
                onValueChange={(value) => setUpdatePlanForm(p => ({ ...p, active: fromSelectValue(value) }))}
              >
                <SelectTrigger className="action-select" data-testid="plans-edit-active">
                  <SelectValue placeholder={t("common.no_change")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL_VALUE}>{t("common.no_change")}</SelectItem>
                  <SelectItem value="true">{t("plans_edit.status.active")}</SelectItem>
                  <SelectItem value="false">{t("plans_edit.status.inactive")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          {updateValidation.length > 0 && <div className="inline-error">{updateValidation.join(" ")}</div>}
          <div className="action-buttons">
            <Button variant="default" disabled={actionLoading || updateValidation.length > 0} onClick={handleUpdate} data-testid="plans-edit-submit">
              {actionLoading ? t("common.updating") : t("plans_edit.actions.update")}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("plans_edit.sections.attach_price")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.price_fields.code")}</Label>
              <Input className="action-input" value={createPriceForm.code} onChange={(e) => setCreatePriceForm(p => ({ ...p, code: e.target.value }))} />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.price_fields.name")}</Label>
              <Input className="action-input" value={createPriceForm.name} onChange={(e) => setCreatePriceForm(p => ({ ...p, name: e.target.value }))} />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.price_fields.type")}</Label>
              <Select
                value={toSelectValue(createPriceForm.priceType)}
                onValueChange={(value) => setCreatePriceForm(p => ({ ...p, priceType: fromSelectValue(value) }))}
              >
                <SelectTrigger className="action-select">
                  <SelectValue placeholder={t("plans_edit.price_fields.type_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL_VALUE}>{t("plans_edit.price_fields.type_placeholder")}</SelectItem>
                  <SelectItem value="flat">{t("plans_edit.price_fields.type_options.flat")}</SelectItem>
                  <SelectItem value="usage">{t("plans_edit.price_fields.type_options.usage")}</SelectItem>
                  <SelectItem value="tiered">{t("plans_edit.price_fields.type_options.tiered")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.price_fields.interval")}</Label>
              <Select value={createPriceForm.billingInterval} onValueChange={(value) => setCreatePriceForm(p => ({ ...p, billingInterval: value }))}>
                <SelectTrigger className="action-select">
                  <SelectValue placeholder={t("plans_edit.price_fields.interval_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="day">{t("plans_edit.price_fields.interval_options.day")}</SelectItem>
                  <SelectItem value="week">{t("plans_edit.price_fields.interval_options.week")}</SelectItem>
                  <SelectItem value="month">{t("plans_edit.price_fields.interval_options.month")}</SelectItem>
                  <SelectItem value="year">{t("plans_edit.price_fields.interval_options.year")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field">
              <Label className="action-label">{t("plans_edit.price_fields.interval_count")}</Label>
              <Input className="action-input" type="number" min={1} value={createPriceForm.billingIntervalCount} onChange={(e) => setCreatePriceForm(p => ({ ...p, billingIntervalCount: Number(e.target.value||1) }))} />
            </div>
            <div className="action-field">
              <AutoCompleteInput
                id="plan-edit-meter"
                label={t("plans_edit.price_fields.meter")}
                value={createPriceForm.meterId}
                options={meterOptions}
                placeholder={t("plans_edit.price_fields.meter_placeholder")}
                onSearch={searchMeters}
                onChange={(value) => setCreatePriceForm(p => ({ ...p, meterId: value }))}
              />
            </div>
          </div>
          {createPriceValidation.length > 0 && <div className="inline-error">{createPriceValidation.join(" ")}</div>}
          <div className="action-buttons">
            <Button variant="default" disabled={actionLoading || createPriceValidation.length > 0} onClick={handleCreatePrice}>
              {actionLoading ? t("common.creating") : t("plans_edit.actions.create_price")}
            </Button>
          </div>
        </div>

        
      </div>
    </div>
  )
}
