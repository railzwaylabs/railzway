import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import HelpHint from "../components/HelpHint";
import AutoCompleteInput from "../components/AutoCompleteInput";
import PageHeader from "../components/PageHeader";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { api } from "../lib/api";
import { useOrgPath } from "../lib/org";
import { currencyHint } from "../lib/hints";
import { useCurrencies } from "../lib/reference";
import { isCurrencyCode } from "../lib/validation";
import { normalizeDate, rfc3339Hint } from "../lib/display";
import type { Plan } from "../lib/types";

export default function SubscriptionsCreate() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const orgPath = useOrgPath();
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies();
  const [form, setForm] = useState({
    customerId: "",
    planId: "",
    planPriceId: "",
    quantity: 1,
    currency: "USD",
    startAt: "",
    currentPeriodStart: "",
    currentPeriodEnd: "",
    trialEnd: "",
    cancelAt: "",
    status: "active"
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [optionsError, setOptionsError] = useState<string | null>(null);
  const [customerOptions, setCustomerOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [planOptions, setPlanOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [planCatalog, setPlanCatalog] = useState<Plan[]>([]);

  useEffect(() => {
    let active = true;
    const loadOptions = async () => {
      try {
        setOptionsError(null);
        const [customersResp, plansResp] = await Promise.all([
          api.customers.list({ page_size: 50 }),
          api.plans.list({ page_size: 50 })
        ]);
        if (!active) {
          return;
        }
        setPlanCatalog(plansResp.plans);
        setCustomerOptions(
          customersResp.customers.map((customer) => ({
            value: customer.id,
            label: `${customer.name} · ${customer.email}`
          }))
        );
        setPlanOptions(
          plansResp.plans.map((plan) => ({
            value: plan.id,
            label: `${plan.name} · ${plan.code}`
          }))
        );
      } catch (err) {
        if (active) {
          setOptionsError(err instanceof Error ? err.message : t("subscriptions_create.toast.load_options_failed"));
        }
      }
    };
    void loadOptions();
    return () => {
      active = false;
    };
  }, [t]);

  const searchCustomers = useCallback(async (query: string) => {
    const trimmed = query.trim();
    if (!trimmed) return [];
    const params = trimmed.includes("@")
      ? { page_size: 50, email: trimmed }
      : { page_size: 50, name: trimmed };
    const resp = await api.customers.list(params);
    return resp.customers.map((customer) => ({
      value: customer.id,
      label: `${customer.name} · ${customer.email}`
    }));
  }, []);

  const searchPlans = useCallback(async (query: string) => {
    const resp = await api.plans.list({ page_size: 50, name: query });
    let plans = resp.plans;
    if (plans.length === 0) {
      const fallback = await api.plans.list({ page_size: 50, code: query });
      plans = fallback.plans;
    }
    return plans.map((plan) => ({
      value: plan.id,
      label: `${plan.name} · ${plan.code}`
    }));
  }, []);

  const validation = useMemo(() => {
    const errors: string[] = [];
    if (!form.customerId.trim()) {
      errors.push(t("subscriptions_create.validation.customer_required"));
    }
    if (!form.planId.trim()) {
      errors.push(t("subscriptions_create.validation.plan_required"));
    }
    if (!form.planPriceId.trim()) {
      errors.push(t("subscriptions_create.validation.price_required"));
    }
    if (!Number.isFinite(form.quantity) || form.quantity <= 0) {
      errors.push(t("subscriptions_create.validation.quantity_min"));
    }
    if (!form.currency.trim()) {
      errors.push(t("subscriptions_create.validation.currency_required"));
    } else if (!isCurrencyCode(form.currency)) {
      errors.push(t("subscriptions_create.validation.currency_invalid"));
    }
    if (!form.currentPeriodStart.trim()) {
      errors.push(t("subscriptions_create.validation.period_start_required"));
    }
    if (!form.currentPeriodEnd.trim()) {
      errors.push(t("subscriptions_create.validation.period_end_required"));
    }
    if (form.currentPeriodStart && form.currentPeriodEnd) {
      const start = new Date(form.currentPeriodStart);
      const end = new Date(form.currentPeriodEnd);
      if (!isNaN(start.getTime()) && !isNaN(end.getTime()) && end < start) {
        errors.push(t("subscriptions_create.validation.period_end_after"));
      }
    }
    return errors;
  }, [form, t]);

  const disabled = saving || validation.length > 0;

  const handleSubmit = useCallback(async () => {
    try {
      setSaving(true);
      setError(null);
      setMessage(null);
      const resp = await api.subscriptions.create({
        customer_id: form.customerId.trim(),
        plan_id: form.planId.trim(),
        currency: form.currency.trim(),
        start_at: form.startAt.trim() ? normalizeDate(form.startAt) : undefined,
        current_period_start: normalizeDate(form.currentPeriodStart),
        current_period_end: normalizeDate(form.currentPeriodEnd),
        trial_end: form.trialEnd.trim() ? normalizeDate(form.trialEnd) : undefined,
        cancel_at: form.cancelAt.trim() ? normalizeDate(form.cancelAt) : undefined,
        status: form.status.trim() || undefined,
        items: [
          {
            plan_price_id: form.planPriceId.trim(),
            quantity: form.quantity
          }
        ]
      });
      setMessage(t("subscriptions_create.toast.created", { id: resp.id }));
      setTimeout(() => navigate(orgPath("/subscriptions")), 800);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("subscriptions_create.toast.create_failed"));
    } finally {
      setSaving(false);
    }
  }, [form, navigate, orgPath, t]);

  const planPriceOptions = useMemo(() => {
    const plan = planCatalog.find((item) => item.id === form.planId);
    if (!plan || !plan.prices) {
      return [];
    }
    return plan.prices.map((price) => {
      const interval = price.billing_interval_count > 1
        ? `${price.billing_interval_count} ${price.billing_interval}s`
        : price.billing_interval;
      return {
        value: price.id,
        label: `${price.name || price.code} · ${price.price_type} · ${interval}`
      };
    });
  }, [form.planId, planCatalog]);

  useEffect(() => {
    if (!form.planPriceId) {
      return;
    }
    const exists = planPriceOptions.some((option) => option.value === form.planPriceId);
    if (!exists) {
      setForm((prev) => ({ ...prev, planPriceId: "" }));
    }
  }, [form.planPriceId, planPriceOptions]);

  return (
    <div className="page-content">
      <PageHeader
        title={t("subscriptions_create.header.title")}
        description={t("subscriptions_create.header.description")}
      />

      <div className="panel" style={{ maxWidth: 920 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">{t("subscriptions_create.sections.customer_pricing")}</div>
          <div className="action-fields">
            <div className="action-field">
              <AutoCompleteInput
                id="subscription-customer-id"
                label={t("subscriptions_create.fields.customer")}
                value={form.customerId}
                options={customerOptions}
                placeholder={t("subscriptions_create.fields.customer_placeholder")}
                onSearch={searchCustomers}
                onChange={(value) => setForm((prev) => ({ ...prev, customerId: value }))}
              />
            </div>
            <div className="action-field">
              <AutoCompleteInput
                id="subscription-plan-id"
                label={t("subscriptions_create.fields.plan")}
                value={form.planId}
                options={planOptions}
                placeholder={t("subscriptions_create.fields.plan_placeholder")}
                onSearch={searchPlans}
                onChange={(value) => setForm((prev) => ({ ...prev, planId: value }))}
              />
            </div>
            <div className="action-field">
              <AutoCompleteInput
                id="subscription-plan-price-id"
                label={t("subscriptions_create.fields.plan_price")}
                value={form.planPriceId}
                options={planPriceOptions}
                placeholder={form.planId ? t("subscriptions_create.fields.plan_price_placeholder") : t("subscriptions_create.fields.plan_price_placeholder_empty")}
                onChange={(value) => setForm((prev) => ({ ...prev, planPriceId: value }))}
              />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("subscriptions_create.fields.quantity")}</Label>
              <Input
                type="number"
                value={form.quantity}
                min={1}
                onChange={(event) => setForm((prev) => ({ ...prev, quantity: Number(event.target.value || 0) }))}
                data-testid="subscriptions-create-quantity"
              />
            </div>
            <div className="action-field">
              <AutoCompleteInput
                id="subscriptions-create-currency"
                label={<>{t("subscriptions_create.fields.currency")} <HelpHint text={currencyHint} /></>}
                value={form.currency}
                options={currencyOptions}
                placeholder={currenciesLoading ? t("common.loading") : undefined}
                onChange={(value) => setForm((prev) => ({ ...prev, currency: value }))}
              />
            </div>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("subscriptions_create.sections.schedule")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_create.fields.start_at")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={form.startAt} onChange={(event) => setForm((prev) => ({ ...prev, startAt: event.target.value }))} data-testid="subscriptions-create-start-at" />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_create.fields.period_start")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={form.currentPeriodStart} onChange={(event) => setForm((prev) => ({ ...prev, currentPeriodStart: event.target.value }))} data-testid="subscriptions-create-period-start" />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_create.fields.period_end")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" min={form.currentPeriodStart || undefined} value={form.currentPeriodEnd} onChange={(event) => setForm((prev) => ({ ...prev, currentPeriodEnd: event.target.value }))} data-testid="subscriptions-create-period-end" />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_create.fields.trial_end")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={form.trialEnd} onChange={(event) => setForm((prev) => ({ ...prev, trialEnd: event.target.value }))} data-testid="subscriptions-create-trial-end" />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_create.fields.cancel_at")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={form.cancelAt} onChange={(event) => setForm((prev) => ({ ...prev, cancelAt: event.target.value }))} data-testid="subscriptions-create-cancel-at" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("subscriptions_create.fields.status")}</Label>
              <Select value={form.status} onValueChange={(value) => setForm((prev) => ({ ...prev, status: value }))}>
                <SelectTrigger className="action-select" data-testid="subscriptions-create-status">
                  <SelectValue placeholder={t("subscriptions_create.fields.status_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">{t("subscriptions_create.fields.status_options.active")}</SelectItem>
                  <SelectItem value="trialing">{t("subscriptions_create.fields.status_options.trialing")}</SelectItem>
                  <SelectItem value="past_due">{t("subscriptions_create.fields.status_options.past_due")}</SelectItem>
                  <SelectItem value="paused">{t("subscriptions_create.fields.status_options.paused")}</SelectItem>
                  <SelectItem value="canceled">{t("subscriptions_create.fields.status_options.canceled")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>

        {validation.length > 0 ? <div className="inline-error">{validation.join(" ")}</div> : null}
        {optionsError ? <div className="inline-error">{t("subscriptions_create.errors.options_prefix")}: {optionsError}</div> : null}
        {message ? <div className="muted">{message}</div> : null}
        {error ? <div className="inline-error">{t("subscriptions_create.errors.prefix")}: {error}</div> : null}

        <div className="action-buttons">
          <Button type="button" onClick={handleSubmit} disabled={disabled} data-testid="subscriptions-create-submit">
            {saving ? t("common.creating") : t("subscriptions_create.actions.create")}
          </Button>
          <Button variant="secondary" type="button" onClick={() => navigate(orgPath("/subscriptions"))} disabled={saving} data-testid="subscriptions-create-cancel">
            {t("common.cancel")}
          </Button>
        </div>
      </div>
    </div>
  );
}
