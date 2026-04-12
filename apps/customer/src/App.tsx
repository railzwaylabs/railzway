import { useCallback, useEffect, useMemo, useState } from "react";
import { customerApi } from "./lib/api";
import type { Invoice, Subscription } from "./lib/types";

const ORG_KEY = "rz_customer_org_id";
const CUSTOMER_KEY = "rz_customer_id";

const formatDate = (value?: string) => {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("en-US", { year: "numeric", month: "short", day: "2-digit" }).format(date);
};

const formatCurrency = (amountCents: number, currency: string) => {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency
  }).format(amountCents / 100);
};

export default function App() {
  const [orgId, setOrgId] = useState("");
  const [customerId, setCustomerId] = useState("");
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const url = new URL(window.location.href);
    const org = url.searchParams.get("org_id") || localStorage.getItem(ORG_KEY) || "";
    const customer = url.searchParams.get("customer_id") || localStorage.getItem(CUSTOMER_KEY) || "";
    if (org) setOrgId(org);
    if (customer) setCustomerId(customer);
  }, []);

  const canLoad = useMemo(() => orgId.trim() !== "" && customerId.trim() !== "", [orgId, customerId]);

  const loadData = useCallback(async () => {
    if (!canLoad) return;
    setLoading(true);
    setError(null);
    try {
      const [subsResp, invResp] = await Promise.all([
        customerApi.listSubscriptions(orgId, customerId),
        customerApi.listInvoices(orgId, customerId)
      ]);
      setSubscriptions(subsResp.subscriptions ?? []);
      setInvoices(invResp.invoices ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, [canLoad, orgId, customerId]);

  useEffect(() => {
    if (canLoad) {
      void loadData();
    }
  }, [canLoad, loadData]);

  const handleSave = () => {
    localStorage.setItem(ORG_KEY, orgId.trim());
    localStorage.setItem(CUSTOMER_KEY, customerId.trim());
    void loadData();
  };

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <div className="max-w-6xl mx-auto px-6 py-6">
          <div className="text-sm text-muted-foreground">Railzway</div>
          <h1 className="text-2xl font-semibold text-slate-900">Customer Portal</h1>
          <p className="text-sm text-muted-foreground mt-1">Track your subscriptions and invoices.</p>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-8 space-y-8">
        <section className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
          <h2 className="text-sm font-semibold text-slate-900 mb-4">Account context</h2>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-xs uppercase tracking-wide text-muted-foreground">Organization ID</label>
              <input
                className="mt-2 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
                value={orgId}
                onChange={(e) => setOrgId(e.target.value)}
                placeholder="org_id"
              />
            </div>
            <div>
              <label className="text-xs uppercase tracking-wide text-muted-foreground">Customer ID</label>
              <input
                className="mt-2 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
                value={customerId}
                onChange={(e) => setCustomerId(e.target.value)}
                placeholder="customer_id"
              />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-3">
            <button
              className="inline-flex items-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white"
              onClick={handleSave}
              disabled={!canLoad || loading}
            >
              {loading ? "Loading…" : "Load portal"}
            </button>
            {error ? <span className="text-sm text-rose-600">{error}</span> : null}
          </div>
        </section>

        <section className="grid gap-6 lg:grid-cols-2">
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold text-slate-900">Subscriptions</h2>
              <span className="text-xs text-muted-foreground">{subscriptions.length} total</span>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-xs text-muted-foreground border-b">
                  <tr>
                    <th className="text-left py-2">Status</th>
                    <th className="text-left py-2">Plan</th>
                    <th className="text-left py-2">Period</th>
                  </tr>
                </thead>
                <tbody>
                  {subscriptions.length === 0 ? (
                    <tr>
                      <td colSpan={3} className="py-6 text-center text-muted-foreground">No subscriptions found.</td>
                    </tr>
                  ) : subscriptions.map((sub) => (
                    <tr key={sub.id} className="border-b border-slate-100">
                      <td className="py-3">
                        <span className="inline-flex rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700">
                          {sub.status}
                        </span>
                      </td>
                      <td className="py-3 text-slate-700">{sub.plan_id}</td>
                      <td className="py-3 text-slate-700">
                        {formatDate(sub.current_period_start)} — {formatDate(sub.current_period_end)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold text-slate-900">Invoices</h2>
              <span className="text-xs text-muted-foreground">{invoices.length} total</span>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-xs text-muted-foreground border-b">
                  <tr>
                    <th className="text-left py-2">Number</th>
                    <th className="text-left py-2">Status</th>
                    <th className="text-left py-2">Amount</th>
                    <th className="text-left py-2">Due</th>
                  </tr>
                </thead>
                <tbody>
                  {invoices.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="py-6 text-center text-muted-foreground">No invoices found.</td>
                    </tr>
                  ) : invoices.map((inv) => (
                    <tr key={inv.id} className="border-b border-slate-100">
                      <td className="py-3 text-slate-700">{inv.number}</td>
                      <td className="py-3">
                        <span className="inline-flex rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700">
                          {inv.status}
                        </span>
                      </td>
                      <td className="py-3 text-slate-700">
                        {formatCurrency(inv.amount_due_cents, inv.currency)}
                      </td>
                      <td className="py-3 text-slate-700">{formatDate(inv.due_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}
