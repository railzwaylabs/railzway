import type { CheckoutSessionResponse, PaymentOptionsResponse, PublicInvoiceResponse, SupportResponse } from "./types";

const baseUrl = (import.meta as ImportMeta & { env?: Record<string, string> }).env?.VITE_PUBLIC_API_BASE_URL ?? "";

type ApiError = {
  error?: string;
  message?: string;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${baseUrl}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    }
  });

  if (!res.ok) {
    let payload: ApiError | undefined;
    try {
      payload = (await res.json()) as ApiError;
    } catch {
      payload = undefined;
    }
    const message = payload?.error || payload?.message || res.statusText || "Request failed";
    throw new Error(message);
  }

  if (res.status === 204) {
    return {} as T;
  }

  return (await res.json()) as T;
}

export const publicApi = {
  getInvoice: (token: string) => request<PublicInvoiceResponse>(`/public/invoices/${encodeURIComponent(token)}`),
  getPaymentOptions: (token: string, country?: string) => {
    const query = country ? `?country=${encodeURIComponent(country)}` : "";
    return request<PaymentOptionsResponse>(`/public/invoices/${encodeURIComponent(token)}/payment-options${query}`);
  },
  requestSupport: (token: string) =>
    request<SupportResponse>(`/public/invoices/${encodeURIComponent(token)}/support`, { method: "POST" }),
  createCheckoutSession: (token: string) =>
    request<CheckoutSessionResponse>(
      `/public/invoices/${encodeURIComponent(token)}/checkout`,
      { method: "POST" }
    )
};
