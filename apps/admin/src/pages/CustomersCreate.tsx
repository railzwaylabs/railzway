import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import HelpHint from "../components/HelpHint";
import PageHeader from "../components/PageHeader";
import AutoCompleteInput from "../components/AutoCompleteInput";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { api } from "../lib/api";
import { useOrgPath } from "../lib/org";
import { currencyHint } from "../lib/hints";
import { useCurrencies } from "../lib/reference";
import { isCurrencyCode, isEmail } from "../lib/validation";

export default function CustomersCreate() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const orgPath = useOrgPath();
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies();
  const [form, setForm] = useState({
    name: "",
    email: "",
    externalId: "",
    currency: ""
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const validation = useMemo(() => {
    const errors: string[] = [];
    if (!form.name.trim()) {
      errors.push(t("customers_create.validation.name_required"));
    }
    if (!form.email.trim()) {
      errors.push(t("customers_create.validation.email_required"));
    } else if (!isEmail(form.email)) {
      errors.push(t("customers_create.validation.email_invalid"));
    }
    if (form.currency.trim() && !isCurrencyCode(form.currency)) {
      errors.push(t("customers_create.validation.currency_invalid"));
    }
    return errors;
  }, [form, t]);

  const disabled = saving || validation.length > 0;

  const handleSubmit = useCallback(async () => {
    try {
      setSaving(true);
      setError(null);
      setMessage(null);
      const resp = await api.customers.create({
        name: form.name.trim(),
        email: form.email.trim(),
        external_id: form.externalId.trim() || undefined,
        currency: form.currency.trim() || undefined
      });
      setMessage(t("customers_create.toast.created", { id: resp.id }));
      setTimeout(() => navigate(orgPath("/customers")), 800);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("customers_create.toast.create_failed"));
    } finally {
      setSaving(false);
    }
  }, [form, navigate, orgPath, t]);

  return (
    <div className="page-content">
      <PageHeader
        title={t("customers_create.header.title")}
        description={t("customers_create.header.description")}
      />

      <div className="panel" style={{ maxWidth: 720 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">{t("customers_create.section.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("customers_create.fields.name")}</Label>
              <Input
                className="action-input"
                value={form.name}
                onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))}
                data-testid="customers-create-name"
              />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("customers_create.fields.email")}</Label>
              <Input
                className="action-input"
                value={form.email}
                onChange={(event) => setForm((prev) => ({ ...prev, email: event.target.value }))}
                data-testid="customers-create-email"
              />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("customers_create.fields.external_id")}</Label>
              <Input
                className="action-input"
                value={form.externalId}
                onChange={(event) => setForm((prev) => ({ ...prev, externalId: event.target.value }))}
                data-testid="customers-create-external-id"
              />
            </div>
            <div className="action-field">
              <AutoCompleteInput
                id="customers-create-currency"
                label={<>{t("customers_create.fields.currency")} <HelpHint text={currencyHint} /></>}
                value={form.currency}
                options={currencyOptions}
                placeholder={currenciesLoading ? t("common.loading") : t("customers_create.fields.currency_placeholder")}
                onChange={(value) => setForm((prev) => ({ ...prev, currency: value }))}
              />
            </div>
          </div>
          {validation.length > 0 ? <div className="inline-error">{validation.join(" ")}</div> : null}
          {message ? <div className="muted">{message}</div> : null}
          {error ? <div className="inline-error">{t("customers_create.errors.prefix")}: {error}</div> : null}
          <div className="action-buttons">
            <Button type="button" onClick={handleSubmit} disabled={disabled} data-testid="customers-create-submit">
              {saving ? t("common.creating") : t("customers_create.actions.create")}
            </Button>
            <Button variant="secondary" type="button" onClick={() => navigate(orgPath("/customers"))} disabled={saving} data-testid="customers-create-cancel">
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
