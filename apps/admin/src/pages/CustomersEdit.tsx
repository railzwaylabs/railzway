import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";
import HelpHint from "../components/HelpHint";
import PageHeader from "../components/PageHeader";
import AutoCompleteInput from "../components/AutoCompleteInput";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { api } from "../lib/api";
import { useOrgPath } from "../lib/org";
import { currencyHint } from "../lib/hints";
import { useCurrencies } from "../lib/reference";
import { isCurrencyCode, isEmail } from "../lib/validation";
import type { TestClock } from "../lib/types";

export default function CustomersEdit() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const orgPath = useOrgPath();
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies();
  const params = useParams();
  const customerId = params.id ?? "";
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [form, setForm] = useState({
    id: "",
    name: "",
    email: "",
    externalId: "",
    currency: "",
    testClockID: "none"
  });
  const [original, setOriginal] = useState({
    name: "",
    email: "",
    externalId: "",
    currency: "",
    testClockID: "none"
  });
  const [testClocks, setTestClocks] = useState<TestClock[]>([]);

  useEffect(() => {
    if (!customerId) {
      setError(t("customers_edit.validation.id_required"));
      setLoading(false);
      return;
    }
    let active = true;
    const load = async () => {
      try {
        setLoading(true);
        const [resp, clocksResp] = await Promise.all([
          api.customers.get(customerId),
          api.testClock.list().catch(() => ({ test_clocks: [] as TestClock[] }))
        ]);
        if (!active) {
          return;
        }
        setTestClocks(clocksResp.test_clocks);
        const testClockID = resp.test_clock_id ?? "none";
        setForm({
          id: resp.id,
          name: resp.name ?? "",
          email: resp.email ?? "",
          externalId: resp.external_id ?? "",
          currency: resp.currency ?? "",
          testClockID
        });
        setOriginal({
          name: resp.name ?? "",
          email: resp.email ?? "",
          externalId: resp.external_id ?? "",
          currency: resp.currency ?? "",
          testClockID
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : t("customers_edit.toast.load_failed"));
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };
    void load();
    return () => {
      active = false;
    };
  }, [customerId, t]);

  const validation = useMemo(() => {
    const errors: string[] = [];
    if (!form.id.trim()) {
      errors.push(t("customers_edit.validation.id_required"));
    }
    const hasUpdates = Boolean(
      form.name.trim() !== original.name ||
        form.email.trim() !== original.email ||
        form.externalId.trim() !== original.externalId ||
        form.currency.trim() !== original.currency ||
        form.testClockID !== original.testClockID
    );
    if (!hasUpdates) {
      errors.push(t("customers_edit.validation.no_changes"));
    }
    if (form.email.trim() && !isEmail(form.email)) {
      errors.push(t("customers_edit.validation.email_invalid"));
    }
    if (form.currency.trim() && !isCurrencyCode(form.currency)) {
      errors.push(t("customers_edit.validation.currency_invalid"));
    }
    return errors;
  }, [form, original, t]);

  const disabled = saving || validation.length > 0;

  const handleSubmit = useCallback(async () => {
    try {
      setSaving(true);
      setError(null);
      setMessage(null);
      const payload: { name?: string; email?: string; external_id?: string; currency?: string; test_clock?: string } = {};
      if (form.name.trim() !== original.name) {
        payload.name = form.name.trim();
      }
      if (form.email.trim() !== original.email) {
        payload.email = form.email.trim();
      }
      if (form.externalId.trim() !== original.externalId) {
        payload.external_id = form.externalId.trim() || undefined;
      }
      if (form.currency.trim() !== original.currency) {
        payload.currency = form.currency.trim() || undefined;
      }
      if (form.testClockID !== original.testClockID) {
        payload.test_clock = form.testClockID === "none" ? "" : form.testClockID;
      }
      const resp = await api.customers.update(form.id.trim(), payload);
      setMessage(t("customers_edit.toast.updated", { id: resp.id }));
      setTimeout(() => navigate(orgPath("/customers")), 800);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("customers_edit.toast.update_failed"));
    } finally {
      setSaving(false);
    }
  }, [form, navigate, original, orgPath, t]);

  if (loading) {
    return <div className="page-content"><div className="loader" /></div>;
  }

  return (
    <div className="page-content">
      <PageHeader
        title={t("customers_edit.header.title")}
        description={t("customers_edit.header.description")}
        actions={
          <Button variant="secondary" type="button" onClick={() => navigate(orgPath("/customers"))} disabled={saving}>
            {t("customers_edit.actions.back")}
          </Button>
        }
      />

      <div className="panel" style={{ maxWidth: 720 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">{t("customers_edit.section.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("customers_edit.fields.id")}</Label>
              <Input className="action-input" value={form.id} onChange={(event) => setForm((prev) => ({ ...prev, id: event.target.value }))} data-testid="customers-edit-id" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("customers_edit.fields.name")}</Label>
              <Input className="action-input" value={form.name} onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))} data-testid="customers-edit-name" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("customers_edit.fields.email")}</Label>
              <Input className="action-input" value={form.email} onChange={(event) => setForm((prev) => ({ ...prev, email: event.target.value }))} data-testid="customers-edit-email" />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("customers_edit.fields.external_id")}</Label>
              <Input
                className="action-input"
                value={form.externalId}
                onChange={(event) => setForm((prev) => ({ ...prev, externalId: event.target.value }))}
                data-testid="customers-edit-external-id"
              />
            </div>
            <div className="action-field">
              <AutoCompleteInput
                id="customers-edit-currency"
                label={<>{t("customers_edit.fields.currency")} <HelpHint text={currencyHint} /></>}
                value={form.currency}
                options={currencyOptions}
                placeholder={currenciesLoading ? t("common.loading") : t("customers_edit.fields.currency_placeholder")}
                onChange={(value) => setForm((prev) => ({ ...prev, currency: value }))}
              />
            </div>
            <div className="action-field">
              <Label className="action-label">Test clock</Label>
              <Select value={form.testClockID} onValueChange={(value) => setForm((prev) => ({ ...prev, testClockID: value }))}>
                <SelectTrigger className="action-select" data-testid="customers-edit-test-clock">
                  <SelectValue placeholder="No test clock" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">No test clock</SelectItem>
                  {testClocks.map((clock) => (
                    <SelectItem key={clock.id} value={clock.id}>{clock.name || clock.id}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          {validation.length > 0 ? <div className="inline-error">{validation.join(" ")}</div> : null}
          {message ? <div className="muted">{message}</div> : null}
          {error ? <div className="inline-error">{t("customers_edit.errors.prefix")}: {error}</div> : null}
          <div className="action-buttons">
            <Button type="button" onClick={handleSubmit} disabled={disabled} data-testid="customers-edit-submit">
              {saving ? t("common.saving") : t("common.save_changes")}
            </Button>
            <Button variant="secondary" type="button" onClick={() => navigate(orgPath("/customers"))} disabled={saving} data-testid="customers-edit-cancel">
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
