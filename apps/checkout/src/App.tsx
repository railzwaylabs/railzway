import { useCallback, useEffect, useMemo, useState } from "react";
import { InvoiceDetail, type InvoiceDetailData } from "@railzway/invoice-ui";
import { publicApi } from "./lib/api";
import type { PublicInvoiceResponse } from "./lib/types";
import { Button } from "./components/ui/button";

const parseToken = () => {
  const url = new URL(window.location.href);
  const queryToken = url.searchParams.get("token");
  if (queryToken) return queryToken;
  const hash = url.hash.replace("#", "");
  if (hash.startsWith("token=")) {
    return hash.replace("token=", "");
  }
  const segments = url.pathname.split("/").filter(Boolean);
  if (segments.length === 0) return "";
  const last = segments[segments.length - 1];
  if (["checkout", "invoice", "invoices"].includes(last)) return "";
  return last;
};

const mapInvoiceData = (payload: PublicInvoiceResponse): InvoiceDetailData => {
  const invoice = payload.invoice;
  const org = payload.organization;
  const cust = payload.customer;

  return {
    invoice: {
      number: invoice.number,
      status: invoice.status,
      currency: invoice.currency,
      issueDate: invoice.issued_at ?? undefined,
      dueDate: invoice.due_at ?? undefined,
      periodStart: invoice.period_start,
      periodEnd: invoice.period_end,
      subtotalCents: invoice.subtotal_cents,
      taxCents: invoice.tax_cents,
      totalCents: invoice.total_cents,
      amountDueCents: invoice.amount_due_cents,
      amountPaidCents: invoice.amount_paid_cents
    },
    billedFrom: org
      ? {
          name: org.name,
          country: org.country_code
        }
      : undefined,
    billedTo: cust
      ? {
          name: cust.name,
          email: cust.email
        }
      : undefined,
    lineItems: payload.items.map((item) => ({
      id: item.id,
      title: item.description ?? item.line_type.replace(/_/g, " "),
      description: item.description ?? undefined,
      quantity: item.quantity,
      unitAmountCents: item.unit_amount_cents,
      amountCents: item.amount_cents,
      currency: item.currency,
      periodStart: item.period_start ?? undefined,
      periodEnd: item.period_end ?? undefined,
      lineType: item.line_type
    })),
    paymentInfo: payload.payment_methods.length
      ? [
          `Payment methods: ${payload.payment_methods.join(", ")}`,
          payload.expires_at ? `Link expires at ${new Date(payload.expires_at).toLocaleString()}` : ""
        ].filter(Boolean)
      : ["Payment methods are not configured yet."],
    terms: ["Payments are processed securely by your billing provider."],
    footerNote: "Questions about this invoice? Contact your billing team."
  };
};

export default function App() {
  const [token, setToken] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [payload, setPayload] = useState<PublicInvoiceResponse | null>(null);
  const [supportStatus, setSupportStatus] = useState<"idle" | "sending" | "sent" | "error">("idle");
  const [checkoutStatus, setCheckoutStatus] = useState<"idle" | "loading" | "error">("idle");
  const [checkoutError, setCheckoutError] = useState<string | null>(null);

  const loadInvoice = useCallback(async (activeToken: string) => {
    if (!activeToken) {
      setError("missing_token");
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await publicApi.getInvoice(activeToken);
      setPayload(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "request_failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const parsed = parseToken();
    setToken(parsed);
    void loadInvoice(parsed);
  }, [loadInvoice]);

  const invoiceData = useMemo(() => (payload ? mapInvoiceData(payload) : null), [payload]);

  const handleSupport = async () => {
    if (!token) return;
    setSupportStatus("sending");
    try {
      await publicApi.requestSupport(token);
      setSupportStatus("sent");
    } catch {
      setSupportStatus("error");
    }
  };

  const handlePayNow = async () => {
    if (!token) return;
    setCheckoutStatus("loading");
    setCheckoutError(null);
    try {
      const resp = await publicApi.createCheckoutSession(token);
      if (resp.checkout_url) {
        window.location.href = resp.checkout_url;
        return;
      }
      setCheckoutStatus("error");
      setCheckoutError("Checkout is not available yet. Please request support.");
    } catch (err) {
      setCheckoutStatus("error");
      setCheckoutError(err instanceof Error ? err.message : "Unable to start checkout.");
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/30">
        <div className="text-sm text-muted-foreground">Loading invoice…</div>
      </div>
    );
  }

  if (error || !invoiceData || !payload) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/30">
        <div className="bg-white border border-border rounded-2xl p-8 shadow-sm max-w-md text-center">
          <div className="text-lg font-semibold text-slate-900 mb-2">Invoice unavailable</div>
          <div className="text-sm text-muted-foreground mb-6">
            {error === "missing_token"
              ? "Missing invoice token. Please use the payment link from your email."
              : "We could not load this invoice. Please refresh or contact support."}
          </div>
          <Button onClick={() => loadInvoice(token)} variant="outline">
            Try again
          </Button>
        </div>
      </div>
    );
  }

  const paymentConfigured = payload.payment_configured;

  return (
    <div className="min-h-screen bg-slate-50">
      {!paymentConfigured ? (
        <div className="max-w-5xl mx-auto px-6 pt-6">
          <div className="rounded-xl border border-amber-200 bg-amber-50 text-amber-900 px-4 py-3 text-sm flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              Payment methods are not configured yet for this organization.
            </div>
            <Button
              variant="secondary"
              size="sm"
              onClick={handleSupport}
              disabled={supportStatus === "sending" || supportStatus === "sent"}
            >
              {supportStatus === "sent" ? "Support requested" : supportStatus === "sending" ? "Sending…" : "Request payment support"}
            </Button>
          </div>
        </div>
      ) : null}
      {checkoutStatus === "error" && checkoutError ? (
        <div className="max-w-5xl mx-auto px-6 pt-4">
          <div className="rounded-xl border border-rose-200 bg-rose-50 text-rose-900 px-4 py-3 text-sm">
            {checkoutError}
          </div>
        </div>
      ) : null}

      <InvoiceDetail
        data={invoiceData}
        variant="full"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="default"
              disabled={!paymentConfigured || checkoutStatus === "loading"}
              onClick={handlePayNow}
            >
              {checkoutStatus === "loading" ? "Starting…" : "Pay now"}
            </Button>
            <Button variant="outline" onClick={handleSupport}>
              Need another method
            </Button>
          </div>
        }
      />
    </div>
  );
}
