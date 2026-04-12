import type { ListInvoicesResponse, ListSubscriptionsResponse } from "./types";

const baseUrl = (import.meta as ImportMeta & { env?: Record<string, string> }).env?.VITE_CUSTOMER_API_BASE_URL ?? "";

async function request<T>(path: string): Promise<T> {
  const res = await fetch(`${baseUrl}${path}`, {
    headers: {
      "Content-Type": "application/json"
    }
  });
  if (!res.ok) {
    let message = res.statusText || "Request failed";
    try {
      const payload = (await res.json()) as { error?: string };
      if (payload?.error) message = payload.error;
    } catch {
      // ignore
    }
    throw new Error(message);
  }
  return (await res.json()) as T;
}

const buildQuery = (params: Record<string, string | undefined>) => {
  const query = Object.entries(params)
    .filter(([, value]) => value)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value ?? "")}`)
    .join("&");
  return query ? `?${query}` : "";
};

export const customerApi = {
  listSubscriptions: (orgId: string, customerId: string, pageToken?: string, pageSize = 20) =>
    request<ListSubscriptionsResponse>(
      `/customer/v1/subscriptions${buildQuery({ org_id: orgId, customer_id: customerId, page_token: pageToken, page_size: String(pageSize) })}`
    ),
  listInvoices: (orgId: string, customerId: string, pageToken?: string, pageSize = 20) =>
    request<ListInvoicesResponse>(
      `/customer/v1/invoices${buildQuery({ org_id: orgId, customer_id: customerId, page_token: pageToken, page_size: String(pageSize) })}`
    )
};
