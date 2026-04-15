import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";
import HelpHint from "../components/HelpHint";
import PageHeader from "../components/PageHeader";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { ALL_VALUE, fromSelectValue, toSelectValue } from "../lib/select";
import { api } from "../lib/api";
import { useOrgPath } from "../lib/org";
import { formatDateTime, normalizeDate, rfc3339Hint } from "../lib/display";
import { statusClass } from "../lib/status";

export default function SubscriptionsEdit() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const orgPath = useOrgPath();
  const params = useParams();
  const subscriptionId = params.id ?? "";
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [subscription, setSubscription] = useState<{
    id: string;
    status: string;
    currentPeriodEnd: string;
  }>({ id: "", status: "", currentPeriodEnd: "" });
  const [updateForm, setUpdateForm] = useState({
    status: "",
    cancelAt: "",
    canceledAt: "",
    endedAt: ""
  });
  const [itemForm, setItemForm] = useState({
    planPriceId: "",
    quantity: 1,
    startAt: "",
    endAt: ""
  });
  const [confirmAction, setConfirmAction] = useState<"cancel_now" | "cancel_end" | null>(null);

  useEffect(() => {
    if (!subscriptionId) {
      setError(t("subscriptions_edit.validation.id_required"));
      setLoading(false);
      return;
    }
    let active = true;
    const load = async () => {
      try {
        setLoading(true);
        const resp = await api.subscriptions.get(subscriptionId);
        if (!active) {
          return;
        }
        setSubscription({
          id: resp.id,
          status: resp.status,
          currentPeriodEnd: resp.current_period_end
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : t("subscriptions_edit.toast.load_failed"));
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
  }, [subscriptionId, t]);

  const updateValidation = useMemo(() => {
    const errors: string[] = [];
    const hasUpdates = Boolean(
      updateForm.status.trim() || updateForm.cancelAt.trim() || updateForm.canceledAt.trim() || updateForm.endedAt.trim()
    );
    if (!hasUpdates) {
      errors.push(t("subscriptions_edit.validation.no_changes"));
    }
    return errors;
  }, [updateForm, t]);

  const itemValidation = useMemo(() => {
    const errors: string[] = [];
    if (!itemForm.planPriceId.trim()) {
      errors.push(t("subscriptions_edit.validation.price_required"));
    }
    if (!Number.isFinite(itemForm.quantity) || itemForm.quantity <= 0) {
      errors.push(t("subscriptions_edit.validation.quantity_min"));
    }
    if (itemForm.startAt && itemForm.endAt) {
      const start = new Date(itemForm.startAt);
      const end = new Date(itemForm.endAt);
      if (!isNaN(start.getTime()) && !isNaN(end.getTime()) && end < start) {
        errors.push(t("subscriptions_edit.validation.item_end_after"));
      }
    }
    return errors;
  }, [itemForm, t]);

  const updateDisabled = saving || updateValidation.length > 0;
  const itemDisabled = saving || itemValidation.length > 0;

  const handleUpdate = useCallback(async () => {
    try {
      setSaving(true);
      setError(null);
      setMessage(null);
      const resp = await api.subscriptions.update(subscriptionId, {
        status: updateForm.status.trim() || undefined,
        cancel_at: updateForm.cancelAt.trim() ? normalizeDate(updateForm.cancelAt) : undefined,
        canceled_at: updateForm.canceledAt.trim() ? normalizeDate(updateForm.canceledAt) : undefined,
        ended_at: updateForm.endedAt.trim() ? normalizeDate(updateForm.endedAt) : undefined
      });
      setSubscription((prev) => ({ ...prev, status: resp.status }));
      setMessage(t("subscriptions_edit.toast.updated", { id: resp.id }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("subscriptions_edit.toast.update_failed"));
    } finally {
      setSaving(false);
    }
  }, [subscriptionId, updateForm, t]);

  const handleCreateItem = useCallback(async () => {
    try {
      setSaving(true);
      setError(null);
      setMessage(null);
      const resp = await api.subscriptions.createItem(subscriptionId, {
        plan_price_id: itemForm.planPriceId.trim(),
        quantity: itemForm.quantity,
        start_at: itemForm.startAt.trim() ? normalizeDate(itemForm.startAt) : undefined,
        end_at: itemForm.endAt.trim() ? normalizeDate(itemForm.endAt) : undefined
      });
      setMessage(t("subscriptions_edit.toast.item_created", { id: resp.id }));
      setItemForm((prev) => ({
        ...prev,
        planPriceId: "",
        quantity: 1,
        startAt: "",
        endAt: ""
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("subscriptions_edit.toast.item_create_failed"));
    } finally {
      setSaving(false);
    }
  }, [itemForm, subscriptionId, t]);

  const handleCancelNow = useCallback(async () => {
    const now = new Date().toISOString();
    setUpdateForm((prev) => ({ ...prev, canceledAt: now }));
    await handleUpdate();
  }, [handleUpdate]);

  const handleCancelAtPeriodEnd = useCallback(async () => {
    if (!subscription.currentPeriodEnd) {
      setError(t("subscriptions_edit.validation.period_end_required"));
      return;
    }
    setUpdateForm((prev) => ({ ...prev, cancelAt: subscription.currentPeriodEnd }));
    await handleUpdate();
  }, [handleUpdate, subscription.currentPeriodEnd, t]);

  const handleConfirmAction = useCallback(async () => {
    if (confirmAction === "cancel_now") {
      setConfirmAction(null);
      await handleCancelNow();
      return;
    }
    if (confirmAction === "cancel_end") {
      setConfirmAction(null);
      await handleCancelAtPeriodEnd();
    }
  }, [confirmAction, handleCancelAtPeriodEnd, handleCancelNow]);

  if (loading) {
    return <div className="page-content"><div className="loader" /></div>;
  }

  return (
    <div className="page-content">
      <PageHeader
        title={t("subscriptions_edit.header.title")}
        description={t("subscriptions_edit.header.description")}
        actions={
          <Button variant="secondary" type="button" onClick={() => navigate(orgPath("/subscriptions"))}>
            {t("subscriptions_edit.actions.back")}
          </Button>
        }
      />

      <div className="action-panel">
        <div className="action-section">
          <div className="action-section-title">{t("subscriptions_edit.sections.current_status")}</div>
          <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
            <Badge className={`status-badge ${statusClass(subscription.status)}`}>{subscription.status}</Badge>
            <span className="muted">{t("subscriptions_edit.sections.period_ends", { date: formatDateTime(subscription.currentPeriodEnd) })}</span>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("subscriptions_edit.sections.update")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("subscriptions_edit.fields.status")}</Label>
              <Select
                value={toSelectValue(updateForm.status)}
                onValueChange={(value) => setUpdateForm((prev) => ({ ...prev, status: fromSelectValue(value) }))}
              >
                <SelectTrigger className="action-select" data-testid="subscriptions-edit-status">
                  <SelectValue placeholder={t("subscriptions_edit.fields.status_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL_VALUE}>{t("common.no_change")}</SelectItem>
                  <SelectItem value="active">{t("subscriptions_edit.fields.status_options.active")}</SelectItem>
                  <SelectItem value="trialing">{t("subscriptions_edit.fields.status_options.trialing")}</SelectItem>
                  <SelectItem value="past_due">{t("subscriptions_edit.fields.status_options.past_due")}</SelectItem>
                  <SelectItem value="paused">{t("subscriptions_edit.fields.status_options.paused")}</SelectItem>
                  <SelectItem value="canceled">{t("subscriptions_edit.fields.status_options.canceled")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_edit.fields.cancel_at")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={updateForm.cancelAt} onChange={(event) => setUpdateForm((prev) => ({ ...prev, cancelAt: event.target.value }))} />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_edit.fields.canceled_at")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={updateForm.canceledAt} onChange={(event) => setUpdateForm((prev) => ({ ...prev, canceledAt: event.target.value }))} />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_edit.fields.ended_at")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={updateForm.endedAt} onChange={(event) => setUpdateForm((prev) => ({ ...prev, endedAt: event.target.value }))} />
            </div>
          </div>
          {updateValidation.length > 0 ? <div className="inline-error">{updateValidation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button type="button" onClick={handleUpdate} disabled={updateDisabled} data-testid="subscriptions-edit-submit">
              {saving ? t("common.updating") : t("common.save_changes")}
            </Button>
            <Button variant="secondary" type="button" onClick={() => setConfirmAction("cancel_now")} disabled={saving}>
              {t("subscriptions_edit.actions.cancel_now")}
            </Button>
            <Button variant="secondary" type="button" onClick={() => setConfirmAction("cancel_end")} disabled={saving}>
              {t("subscriptions_edit.actions.cancel_end")}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("subscriptions_edit.sections.items")}</div>
          <div className="action-fields">
            <div className="action-field">
              <Label className="action-label">{t("subscriptions_edit.items.price_id")}</Label>
              <Input value={itemForm.planPriceId} onChange={(event) => setItemForm((prev) => ({ ...prev, planPriceId: event.target.value }))} />
            </div>
            <div className="action-field">
              <Label className="action-label">{t("subscriptions_edit.items.quantity")}</Label>
              <Input
                type="number"
                value={itemForm.quantity}
                onChange={(event) => setItemForm((prev) => ({ ...prev, quantity: Number.parseFloat(event.target.value || "0") }))}
              />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_edit.items.start_at")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" value={itemForm.startAt} onChange={(event) => setItemForm((prev) => ({ ...prev, startAt: event.target.value }))} />
            </div>
            <div className="action-field">
              <Label className="action-label">
                {t("subscriptions_edit.items.end_at")} <HelpHint text={rfc3339Hint} />
              </Label>
              <Input type="datetime-local" min={itemForm.startAt || undefined} value={itemForm.endAt} onChange={(event) => setItemForm((prev) => ({ ...prev, endAt: event.target.value }))} />
            </div>
          </div>
          {itemValidation.length > 0 ? <div className="inline-error">{itemValidation.join(" ")}</div> : null}
          {message ? <div className="muted">{message}</div> : null}
          {error ? <div className="inline-error">{t("subscriptions_edit.errors.prefix")}: {error}</div> : null}
          <div className="action-buttons">
            <Button type="button" onClick={handleCreateItem} disabled={itemDisabled}>
              {saving ? t("subscriptions_edit.items.adding") : t("subscriptions_edit.items.add")}
            </Button>
          </div>
        </div>
      </div>

      <Dialog open={confirmAction !== null} onOpenChange={(open) => { if (!open) setConfirmAction(null); }}>
        <DialogContent style={{ maxWidth: 480 }}>
          <DialogHeader>
            <DialogTitle>
              {confirmAction === "cancel_now"
                ? t("subscriptions_edit.actions.cancel_now")
                : t("subscriptions_edit.actions.cancel_end")}
            </DialogTitle>
            <DialogDescription>
              {confirmAction === "cancel_now"
                ? t("subscriptions_edit.confirm.cancel_now")
                : t("subscriptions_edit.confirm.cancel_end")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter style={{ marginTop: "8px" }}>
            <Button variant="secondary" onClick={() => setConfirmAction(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={() => void handleConfirmAction()} disabled={saving}>
              {confirmAction === "cancel_now"
                ? t("subscriptions_edit.actions.cancel_now")
                : t("subscriptions_edit.actions.cancel_end")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
