import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Badge } from "../components/ui/badge"
import { Button } from "../components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { api } from "../lib/api"
import { formatDate, normalizeDate } from "../lib/display"
import { centsToMoneyInput, DEFAULT_MONEY_INPUT, isNonNegativeMoneyInput, moneyInputToCents, MONEY_INPUT_STEP } from "../lib/money"
import type { BillingSegment, Coupon, PromotionCode } from "../lib/types"

function IconCoupon() {
  return (
    <svg width="20" height="20" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2.5 6.5l4-4h6.2a.8.8 0 01.8.8v6.2l-4 4a1.2 1.2 0 01-1.7 0L2.5 8.2a1.2 1.2 0 010-1.7z" strokeLinejoin="round"/>
      <circle cx="10.8" cy="5.2" r="1" />
      <path d="M6 10l4-4" strokeLinecap="round"/>
    </svg>
  )
}

function IconPlus() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M8 3v10M3 8h10" strokeLinecap="round" />
    </svg>
  )
}

function formatCouponValue(coupon: Coupon) {
  if (coupon.type === "PERCENT") return `${coupon.percentage ?? 0}%`
  const amount = centsToMoneyInput(coupon.amount_cents ?? 0)
  return `${coupon.currency ?? ""} ${amount}`.trim()
}

function formatDuration(coupon: Coupon) {
  if (coupon.duration === "REPEATING") return `${coupon.duration_months ?? 0} months`
  return coupon.duration
}

export default function Coupons() {
  const { t } = useTranslation()
  const [coupons, setCoupons] = useState<Coupon[]>([])
  const [promotionCodes, setPromotionCodes] = useState<PromotionCode[]>([])
  const [segments, setSegments] = useState<BillingSegment[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [couponOpen, setCouponOpen] = useState(false)
  const [promoOpen, setPromoOpen] = useState(false)
  const [redeemOpen, setRedeemOpen] = useState(false)
  const [segmentOpen, setSegmentOpen] = useState(false)
  const [couponForm, setCouponForm] = useState({
    name: "",
    type: "PERCENT",
    percentage: "20",
    amount: DEFAULT_MONEY_INPUT,
    currency: "USD",
    duration: "ONCE",
    durationMonths: "1",
    validFrom: "",
    validUntil: "",
    autoApply: "false",
    targetSegment: "__all__"
  })
  const [segmentForm, setSegmentForm] = useState({
    key: "",
    name: "",
    scope: "customer",
    description: ""
  })
  const [promoForm, setPromoForm] = useState({
    couponId: "",
    code: "",
    active: "true",
    maxRedemptions: ""
  })
  const [redeemForm, setRedeemForm] = useState({
    subscriptionId: "",
    code: ""
  })

  const couponByID = useMemo(() => {
    const out = new Map<string, Coupon>()
    coupons.forEach((coupon) => out.set(coupon.id, coupon))
    return out
  }, [coupons])

  const segmentByKey = useMemo(() => {
    const out = new Map<string, BillingSegment>()
    segments.forEach((segment) => out.set(segment.key, segment))
    return out
  }, [segments])

  const activeSegments = useMemo(() => segments.filter((segment) => segment.active), [segments])

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      const [couponResp, promoResp, segmentResp] = await Promise.all([
        api.coupons.list(),
        api.coupons.listPromotionCodes(),
        api.coupons.listSegments({ include_inactive: true })
      ])
      setCoupons(couponResp.coupons ?? [])
      setPromotionCodes(promoResp.promotion_codes ?? [])
      setSegments(segmentResp.segments ?? [])
      setPromoForm((prev) => ({
        ...prev,
        couponId: prev.couponId || couponResp.coupons?.[0]?.id || ""
      }))
    } catch (err) {
      toast.error(t("coupons.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const couponValidation = useMemo(() => {
    const errors: string[] = []
    if (!couponForm.name.trim()) errors.push(t("coupons.validation.name_required"))
    if (couponForm.type === "PERCENT") {
      const percentage = Number.parseFloat(couponForm.percentage)
      if (!Number.isFinite(percentage) || percentage <= 0 || percentage > 100) {
        errors.push(t("coupons.validation.percentage_range"))
      }
    }
    if (couponForm.type === "FIXED") {
      if (!isNonNegativeMoneyInput(couponForm.amount) || moneyInputToCents(couponForm.amount) <= 0) {
        errors.push(t("coupons.validation.amount_positive"))
      }
      if (!/^[A-Za-z]{3}$/.test(couponForm.currency.trim())) {
        errors.push(t("coupons.validation.currency"))
      }
    }
    if (couponForm.duration === "REPEATING") {
      const months = Number.parseInt(couponForm.durationMonths, 10)
      if (!Number.isFinite(months) || months <= 0) {
        errors.push(t("coupons.validation.duration_months"))
      }
    }
    if (couponForm.validFrom && couponForm.validUntil) {
      const from = new Date(couponForm.validFrom)
      const until = new Date(couponForm.validUntil)
      if (!Number.isNaN(from.getTime()) && !Number.isNaN(until.getTime()) && until <= from) {
        errors.push(t("coupons.validation.valid_until_after"))
      }
    }
    return errors
  }, [couponForm, t])

  const promoValidation = useMemo(() => {
    const errors: string[] = []
    if (!promoForm.couponId) errors.push(t("coupons.validation.coupon_required"))
    if (!promoForm.code.trim()) errors.push(t("coupons.validation.code_required"))
    if (promoForm.maxRedemptions.trim()) {
      const parsed = Number.parseInt(promoForm.maxRedemptions, 10)
      if (!Number.isFinite(parsed) || parsed <= 0) errors.push(t("coupons.validation.max_redemptions"))
    }
    return errors
  }, [promoForm, t])

  const redeemValidation = useMemo(() => {
    const errors: string[] = []
    if (!redeemForm.subscriptionId.trim()) errors.push(t("coupons.validation.subscription_required"))
    if (!redeemForm.code.trim()) errors.push(t("coupons.validation.code_required"))
    return errors
  }, [redeemForm, t])

  const segmentValidation = useMemo(() => {
    const errors: string[] = []
    if (!segmentForm.key.trim()) errors.push(t("coupons.validation.segment_key_required"))
    if (!segmentForm.name.trim()) errors.push(t("coupons.validation.segment_name_required"))
    if (!["any", "customer", "subscription"].includes(segmentForm.scope)) {
      errors.push(t("coupons.validation.segment_scope"))
    }
    return errors
  }, [segmentForm, t])

  const handleCreateCoupon = useCallback(async () => {
    try {
      setSaving(true)
      const payload: {
        name: string
        type: string
        amount_cents?: number
        percentage?: number
        duration: string
        duration_months?: number
        currency?: string
        valid_from?: string
        valid_until?: string
        auto_apply?: boolean
        target_segment?: string
      } = {
        name: couponForm.name.trim(),
        type: couponForm.type,
        duration: couponForm.duration,
        auto_apply: couponForm.autoApply === "true"
      }
      if (couponForm.type === "PERCENT") {
        payload.percentage = Number.parseFloat(couponForm.percentage)
      } else {
        payload.amount_cents = moneyInputToCents(couponForm.amount)
        payload.currency = couponForm.currency.trim().toUpperCase()
      }
      if (couponForm.duration === "REPEATING") {
        payload.duration_months = Number.parseInt(couponForm.durationMonths, 10)
      }
      if (couponForm.validFrom.trim()) {
        payload.valid_from = normalizeDate(couponForm.validFrom)
      }
      if (couponForm.validUntil.trim()) {
        payload.valid_until = normalizeDate(couponForm.validUntil)
      }
      if (couponForm.targetSegment !== "__all__") {
        payload.target_segment = couponForm.targetSegment.trim()
      }
      const resp = await api.coupons.create(payload)
      toast.success(t("coupons.toast.coupon_created"), resp.coupon.id)
      setCouponOpen(false)
      setCouponForm((prev) => ({ ...prev, name: "" }))
      await loadData()
    } catch (err) {
      toast.error(t("coupons.toast.coupon_create_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [couponForm, loadData, t])

  const handleCreateSegment = useCallback(async () => {
    try {
      setSaving(true)
      const resp = await api.coupons.createSegment({
        key: segmentForm.key.trim(),
        name: segmentForm.name.trim(),
        scope: segmentForm.scope,
        description: segmentForm.description.trim() || undefined,
        active: true
      })
      toast.success(t("coupons.toast.segment_created"), resp.segment.name)
      setSegmentOpen(false)
      setSegmentForm({ key: "", name: "", scope: "customer", description: "" })
      await loadData()
    } catch (err) {
      toast.error(t("coupons.toast.segment_create_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [loadData, segmentForm, t])

  const handleCreatePromotionCode = useCallback(async () => {
    try {
      setSaving(true)
      const maxRedemptions = promoForm.maxRedemptions.trim()
        ? Number.parseInt(promoForm.maxRedemptions, 10)
        : undefined
      const resp = await api.coupons.createPromotionCode({
        coupon_id: promoForm.couponId,
        code: promoForm.code.trim(),
        active: promoForm.active === "true",
        max_redemptions: maxRedemptions
      })
      toast.success(t("coupons.toast.promo_created"), resp.promotion_code.code)
      setPromoOpen(false)
      setPromoForm((prev) => ({ ...prev, code: "", maxRedemptions: "" }))
      await loadData()
    } catch (err) {
      toast.error(t("coupons.toast.promo_create_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [loadData, promoForm, t])

  const handleRedeemPromotionCode = useCallback(async () => {
    try {
      setSaving(true)
      const resp = await api.subscriptions.redeemPromotionCode(redeemForm.subscriptionId.trim(), {
        code: redeemForm.code.trim()
      })
      toast.success(t("coupons.toast.redeemed"), resp.coupon.name)
      setRedeemOpen(false)
      setRedeemForm({ subscriptionId: "", code: "" })
      await loadData()
    } catch (err) {
      toast.error(t("coupons.toast.redeem_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [loadData, redeemForm, t])

  const couponColumns = useMemo(() => [
    {
      key: "name",
      label: t("coupons.coupons_table.columns.name"),
      render: (row: Coupon) => (
        <div>
          <div style={{ fontWeight: 600 }}>{row.name}</div>
          <div className="muted" style={{ fontSize: 11 }}>{row.id}</div>
        </div>
      )
    },
    {
      key: "type",
      label: t("coupons.coupons_table.columns.type"),
      width: "110px",
      render: (row: Coupon) => <Badge className={row.type === "PERCENT" ? "badge-info" : "badge-muted"}>{row.type}</Badge>
    },
    {
      key: "value",
      label: t("coupons.coupons_table.columns.value"),
      width: "140px",
      render: (row: Coupon) => <strong>{formatCouponValue(row)}</strong>
    },
    {
      key: "duration",
      label: t("coupons.coupons_table.columns.duration"),
      width: "130px",
      render: (row: Coupon) => <span className="muted">{formatDuration(row)}</span>
    },
    {
      key: "automation",
      label: t("coupons.coupons_table.columns.automation"),
      width: "120px",
      render: (row: Coupon) => (
        <span className={`badge ${row.auto_apply ? "badge-success" : "badge-muted"}`}>
          {row.auto_apply ? t("coupons.automation.auto") : t("coupons.automation.manual")}
        </span>
      )
    },
    {
      key: "segment",
      label: t("coupons.coupons_table.columns.segment"),
      width: "140px",
      render: (row: Coupon) => {
        const segment = row.target_segment ? segmentByKey.get(row.target_segment) : undefined
        return <span className="muted">{segment?.name || row.target_segment || t("coupons.segments.all")}</span>
      }
    },
    {
      key: "validity",
      label: t("coupons.coupons_table.columns.validity"),
      width: "190px",
      render: (row: Coupon) => (
        <span className="muted">
          {row.valid_from ? formatDate(row.valid_from) : "—"} / {row.valid_until ? formatDate(row.valid_until) : "—"}
        </span>
      )
    },
    {
      key: "created_at",
      label: t("common.created"),
      width: "130px",
      render: (row: Coupon) => <span className="muted">{formatDate(row.created_at)}</span>
    },
  ], [segmentByKey, t])

  const promoColumns = useMemo(() => [
    {
      key: "code",
      label: t("coupons.promos_table.columns.code"),
      render: (row: PromotionCode) => (
        <div>
          <code className="cell-mono">{row.code}</code>
          <div className="muted" style={{ fontSize: 11 }}>{row.id}</div>
        </div>
      )
    },
    {
      key: "coupon",
      label: t("coupons.promos_table.columns.coupon"),
      render: (row: PromotionCode) => couponByID.get(row.coupon_id)?.name ?? row.coupon_id
    },
    {
      key: "active",
      label: t("coupons.promos_table.columns.status"),
      width: "100px",
      render: (row: PromotionCode) => (
        <span className={`badge ${row.active ? "badge-success" : "badge-muted"}`}>
          {row.active ? t("coupons.status.active") : t("coupons.status.inactive")}
        </span>
      )
    },
    {
      key: "redemptions",
      label: t("coupons.promos_table.columns.redemptions"),
      width: "140px",
      render: (row: PromotionCode) => (
        <span>{row.redemption_count}{row.max_redemptions ? ` / ${row.max_redemptions}` : ""}</span>
      )
    },
    {
      key: "created_at",
      label: t("common.created"),
      width: "130px",
      render: (row: PromotionCode) => <span className="muted">{formatDate(row.created_at)}</span>
    },
  ], [couponByID, t])

  const segmentColumns = useMemo(() => [
    {
      key: "name",
      label: t("coupons.segments_table.columns.name"),
      render: (row: BillingSegment) => (
        <div>
          <div style={{ fontWeight: 600 }}>{row.name}</div>
          <div className="muted" style={{ fontSize: 11 }}>{row.key}</div>
        </div>
      )
    },
    {
      key: "scope",
      label: t("coupons.segments_table.columns.scope"),
      width: "130px",
      render: (row: BillingSegment) => <Badge className="badge-muted">{t(`coupons.segment_scopes.${row.scope}`, row.scope)}</Badge>
    },
    {
      key: "status",
      label: t("coupons.segments_table.columns.status"),
      width: "100px",
      render: (row: BillingSegment) => (
        <span className={`badge ${row.active ? "badge-success" : "badge-muted"}`}>
          {row.active ? t("coupons.status.active") : t("coupons.status.inactive")}
        </span>
      )
    },
    {
      key: "description",
      label: t("coupons.segments_table.columns.description"),
      render: (row: BillingSegment) => <span className="muted">{row.description || "—"}</span>
    },
  ], [t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconCoupon />}
        title={t("coupons.header.title")}
        description={t("coupons.header.description")}
        actions={
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <Dialog open={couponOpen} onOpenChange={setCouponOpen}>
              <DialogTrigger asChild>
                <Button style={{ gap: 6 }} data-testid="coupons-create-open"><IconPlus /> {t("coupons.actions.new_coupon")}</Button>
              </DialogTrigger>
              <DialogContent style={{ maxWidth: 720 }}>
                <DialogHeader>
                  <DialogTitle>{t("coupons.create_coupon.title")}</DialogTitle>
                  <DialogDescription>{t("coupons.create_coupon.description")}</DialogDescription>
                </DialogHeader>
                <div className="action-fields">
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.name")}</Label>
                    <Input className="action-input" value={couponForm.name} onChange={(e) => setCouponForm((p) => ({ ...p, name: e.target.value }))} data-testid="coupons-create-name" />
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.type")}</Label>
                    <Select value={couponForm.type} onValueChange={(value) => setCouponForm((p) => ({ ...p, type: value }))}>
                      <SelectTrigger className="action-select" data-testid="coupons-create-type"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="PERCENT">{t("coupons.types.percent")}</SelectItem>
                        <SelectItem value="FIXED">{t("coupons.types.fixed")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  {couponForm.type === "PERCENT" ? (
                    <div className="action-field">
                      <Label className="action-label">{t("coupons.fields.percentage")}</Label>
                      <Input className="action-input" type="number" min="0.01" max="100" step="0.01" value={couponForm.percentage} onChange={(e) => setCouponForm((p) => ({ ...p, percentage: e.target.value }))} data-testid="coupons-create-percentage" />
                    </div>
                  ) : (
                    <>
                      <div className="action-field">
                        <Label className="action-label">{t("coupons.fields.amount")}</Label>
                        <Input className="action-input" type="number" min="0" step={MONEY_INPUT_STEP} value={couponForm.amount} onChange={(e) => setCouponForm((p) => ({ ...p, amount: e.target.value }))} data-testid="coupons-create-amount" />
                      </div>
                      <div className="action-field">
                        <Label className="action-label">{t("coupons.fields.currency")}</Label>
                        <Input className="action-input" maxLength={3} value={couponForm.currency} onChange={(e) => setCouponForm((p) => ({ ...p, currency: e.target.value.toUpperCase() }))} data-testid="coupons-create-currency" />
                      </div>
                    </>
                  )}
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.duration")}</Label>
                    <Select value={couponForm.duration} onValueChange={(value) => setCouponForm((p) => ({ ...p, duration: value }))}>
                      <SelectTrigger className="action-select" data-testid="coupons-create-duration"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="ONCE">{t("coupons.durations.once")}</SelectItem>
                        <SelectItem value="REPEATING">{t("coupons.durations.repeating")}</SelectItem>
                        <SelectItem value="FOREVER">{t("coupons.durations.forever")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  {couponForm.duration === "REPEATING" ? (
                    <div className="action-field">
                      <Label className="action-label">{t("coupons.fields.duration_months")}</Label>
                      <Input className="action-input" type="number" min="1" step="1" value={couponForm.durationMonths} onChange={(e) => setCouponForm((p) => ({ ...p, durationMonths: e.target.value }))} data-testid="coupons-create-duration-months" />
                    </div>
                  ) : null}
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.auto_apply")}</Label>
                    <Select value={couponForm.autoApply} onValueChange={(value) => setCouponForm((p) => ({ ...p, autoApply: value }))}>
                      <SelectTrigger className="action-select" data-testid="coupons-create-auto-apply"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="false">{t("coupons.automation.manual")}</SelectItem>
                        <SelectItem value="true">{t("coupons.automation.auto")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.target_segment")}</Label>
                    <Select value={couponForm.targetSegment} onValueChange={(value) => setCouponForm((p) => ({ ...p, targetSegment: value }))}>
                      <SelectTrigger className="action-select" data-testid="coupons-create-target-segment"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__all__">{t("coupons.segments.all")}</SelectItem>
                        {activeSegments.map((segment) => (
                          <SelectItem key={segment.key} value={segment.key}>{segment.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.valid_from")}</Label>
                    <Input className="action-input" type="date" value={couponForm.validFrom} onChange={(e) => setCouponForm((p) => ({ ...p, validFrom: e.target.value }))} data-testid="coupons-create-valid-from" />
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.valid_until")}</Label>
                    <Input className="action-input" type="date" min={couponForm.validFrom || undefined} value={couponForm.validUntil} onChange={(e) => setCouponForm((p) => ({ ...p, validUntil: e.target.value }))} data-testid="coupons-create-valid-until" />
                  </div>
                </div>
                {couponValidation.length ? <div className="inline-error">{couponValidation.join(" ")}</div> : null}
                <DialogFooter>
                  <Button variant="secondary" onClick={() => setCouponOpen(false)}>{t("common.cancel")}</Button>
                  <Button disabled={saving || couponValidation.length > 0} onClick={handleCreateCoupon}>{saving ? t("common.creating") : t("coupons.actions.create_coupon")}</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>

            <Dialog open={promoOpen} onOpenChange={setPromoOpen}>
              <DialogTrigger asChild>
                <Button variant="secondary" style={{ gap: 6 }} data-testid="coupons-promo-open"><IconPlus /> {t("coupons.actions.new_promo")}</Button>
              </DialogTrigger>
              <DialogContent style={{ maxWidth: 640 }}>
                <DialogHeader>
                  <DialogTitle>{t("coupons.create_promo.title")}</DialogTitle>
                  <DialogDescription>{t("coupons.create_promo.description")}</DialogDescription>
                </DialogHeader>
                <div className="action-fields">
                  <div className="action-field" style={{ gridColumn: "1 / -1" }}>
                    <Label className="action-label">{t("coupons.fields.coupon")}</Label>
                    <Select value={promoForm.couponId} onValueChange={(value) => setPromoForm((p) => ({ ...p, couponId: value }))}>
                      <SelectTrigger className="action-select" data-testid="coupons-promo-coupon"><SelectValue placeholder={t("common.select_placeholder")} /></SelectTrigger>
                      <SelectContent>
                        {coupons.map((coupon) => (
                          <SelectItem key={coupon.id} value={coupon.id}>{coupon.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.code")}</Label>
                    <Input className="action-input" value={promoForm.code} onChange={(e) => setPromoForm((p) => ({ ...p, code: e.target.value.toUpperCase() }))} data-testid="coupons-promo-code" />
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.status")}</Label>
                    <Select value={promoForm.active} onValueChange={(value) => setPromoForm((p) => ({ ...p, active: value }))}>
                      <SelectTrigger className="action-select" data-testid="coupons-promo-active"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="true">{t("coupons.status.active")}</SelectItem>
                        <SelectItem value="false">{t("coupons.status.inactive")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.max_redemptions")}</Label>
                    <Input className="action-input" type="number" min="1" step="1" value={promoForm.maxRedemptions} onChange={(e) => setPromoForm((p) => ({ ...p, maxRedemptions: e.target.value }))} data-testid="coupons-promo-max-redemptions" />
                  </div>
                </div>
                {promoValidation.length ? <div className="inline-error">{promoValidation.join(" ")}</div> : null}
                <DialogFooter>
                  <Button variant="secondary" onClick={() => setPromoOpen(false)}>{t("common.cancel")}</Button>
                  <Button disabled={saving || promoValidation.length > 0} onClick={handleCreatePromotionCode}>{saving ? t("common.creating") : t("coupons.actions.create_promo")}</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>

            <Dialog open={segmentOpen} onOpenChange={setSegmentOpen}>
              <DialogTrigger asChild>
                <Button variant="secondary" style={{ gap: 6 }} data-testid="coupons-segment-open"><IconPlus /> {t("coupons.actions.new_segment")}</Button>
              </DialogTrigger>
              <DialogContent style={{ maxWidth: 560 }}>
                <DialogHeader>
                  <DialogTitle>{t("coupons.create_segment.title")}</DialogTitle>
                  <DialogDescription>{t("coupons.create_segment.description")}</DialogDescription>
                </DialogHeader>
                <div className="action-fields">
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.segment_key")}</Label>
                    <Input className="action-input" value={segmentForm.key} onChange={(e) => setSegmentForm((p) => ({ ...p, key: e.target.value }))} data-testid="coupons-segment-key" />
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.name")}</Label>
                    <Input className="action-input" value={segmentForm.name} onChange={(e) => setSegmentForm((p) => ({ ...p, name: e.target.value }))} data-testid="coupons-segment-name" />
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.scope")}</Label>
                    <Select value={segmentForm.scope} onValueChange={(value) => setSegmentForm((p) => ({ ...p, scope: value }))}>
                      <SelectTrigger className="action-select" data-testid="coupons-segment-scope"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="customer">{t("coupons.segment_scopes.customer")}</SelectItem>
                        <SelectItem value="subscription">{t("coupons.segment_scopes.subscription")}</SelectItem>
                        <SelectItem value="any">{t("coupons.segment_scopes.any")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="action-field">
                    <Label className="action-label">{t("coupons.fields.description")}</Label>
                    <Input className="action-input" value={segmentForm.description} onChange={(e) => setSegmentForm((p) => ({ ...p, description: e.target.value }))} data-testid="coupons-segment-description" />
                  </div>
                </div>
                {segmentValidation.length ? <div className="inline-error">{segmentValidation.join(" ")}</div> : null}
                <DialogFooter>
                  <Button variant="secondary" onClick={() => setSegmentOpen(false)}>{t("common.cancel")}</Button>
                  <Button disabled={saving || segmentValidation.length > 0} onClick={handleCreateSegment}>{saving ? t("common.creating") : t("coupons.actions.create_segment")}</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>

            <Dialog open={redeemOpen} onOpenChange={setRedeemOpen}>
              <DialogTrigger asChild>
                <Button variant="secondary" data-testid="coupons-redeem-open">{t("coupons.actions.redeem")}</Button>
              </DialogTrigger>
              <DialogContent style={{ maxWidth: 560 }}>
                <DialogHeader>
                  <DialogTitle>{t("coupons.redeem.title")}</DialogTitle>
                  <DialogDescription>{t("coupons.redeem.description")}</DialogDescription>
                </DialogHeader>
                <div className="action-fields">
                  <div className="action-field" style={{ gridColumn: "1 / -1" }}>
                    <Label className="action-label">{t("coupons.fields.subscription_id")}</Label>
                    <Input className="action-input" value={redeemForm.subscriptionId} onChange={(e) => setRedeemForm((p) => ({ ...p, subscriptionId: e.target.value }))} data-testid="coupons-redeem-subscription" />
                  </div>
                  <div className="action-field" style={{ gridColumn: "1 / -1" }}>
                    <Label className="action-label">{t("coupons.fields.code")}</Label>
                    <Input className="action-input" value={redeemForm.code} onChange={(e) => setRedeemForm((p) => ({ ...p, code: e.target.value.toUpperCase() }))} data-testid="coupons-redeem-code" />
                  </div>
                </div>
                {redeemValidation.length ? <div className="inline-error">{redeemValidation.join(" ")}</div> : null}
                <DialogFooter>
                  <Button variant="secondary" onClick={() => setRedeemOpen(false)}>{t("common.cancel")}</Button>
                  <Button disabled={saving || redeemValidation.length > 0} onClick={handleRedeemPromotionCode}>{saving ? t("common.saving") : t("coupons.actions.redeem")}</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        }
      />

      <div className="stat-grid">
        <div className="stat-card">
          <div className="stat-label">{t("coupons.kpis.coupons")}</div>
          <div className="stat-value">{loading ? t("common.loading_dash") : coupons.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">{t("coupons.kpis.promotion_codes")}</div>
          <div className="stat-value">{loading ? t("common.loading_dash") : promotionCodes.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">{t("coupons.kpis.redemptions")}</div>
          <div className="stat-value">{loading ? t("common.loading_dash") : promotionCodes.reduce((sum, item) => sum + item.redemption_count, 0)}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">{t("coupons.kpis.segments")}</div>
          <div className="stat-value">{loading ? t("common.loading_dash") : segments.length}</div>
        </div>
      </div>

      <div className="action-panel">
        <div className="action-section">
          <div className="action-section-title">{t("coupons.sections.coupons")}</div>
          <DataTable
            columns={couponColumns as Parameters<typeof DataTable<Coupon>>[0]["columns"]}
            data={coupons}
            loading={loading}
            emptyTitle={t("coupons.coupons_table.empty_title")}
            emptyDesc={t("coupons.coupons_table.empty_desc")}
          />
        </div>
        <div className="action-section">
          <div className="action-section-title">{t("coupons.sections.promotion_codes")}</div>
          <DataTable
            columns={promoColumns as Parameters<typeof DataTable<PromotionCode>>[0]["columns"]}
            data={promotionCodes}
            loading={loading}
            emptyTitle={t("coupons.promos_table.empty_title")}
            emptyDesc={t("coupons.promos_table.empty_desc")}
          />
        </div>
        <div className="action-section">
          <div className="action-section-title">{t("coupons.sections.segments")}</div>
          <DataTable
            columns={segmentColumns as Parameters<typeof DataTable<BillingSegment>>[0]["columns"]}
            data={segments}
            loading={loading}
            emptyTitle={t("coupons.segments_table.empty_title")}
            emptyDesc={t("coupons.segments_table.empty_desc")}
          />
        </div>
      </div>
    </div>
  )
}
