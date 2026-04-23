import {
  AdminChangePasswordRequest,
  AdminChangePasswordResponse,
  AdminLoginRequest,
  AdminLoginResponse,
  AdminLogoutResponse,
  AdminMeResponse,
  AdminRevokeSessionResponse,
  AdminSessionsResponse,
  AdminSwitchOrgResponse,
  AdminSkipPasswordResponse,
  AppDefinition,
  AppInstallation,
  AppsCatalogResponse,
  AppsInstallationsResponse,
  ConfigWarningsResponse,
  ReconciliationSummaryResponse,
  AuditLogsListResponse,
  OrganizationMemberInfo,
  AuditLogsSummary,
  BillingSegmentResponse,
  BillingSegmentsListResponse,
  Customer,
  CouponResponse,
  CouponsListResponse,
  CustomersListResponse,
  CustomersSummary,
  DashboardSummary,
  FeatureFlagUpsertRequest,
  FeatureFlagUpsertResponse,
  FeatureFlagsResponse,
  Invoice,
  InvoiceDetailResponse,
  InvoiceNumberFormat,
  InvoicePublicLink,
  InvoiceResendResponse,
  InvoicesListResponse,
  InvoicesSummary,
  LedgerEntry,
  LedgerTransaction,
  LedgerTransactionsResponse,
  Meter,
  MetersListResponse,
  Organization,
  OrganizationListItem,
  PaymentsListResponse,
  PaymentsSummary,
  PlanAmount,
  Plan,
  PlanPrice,
  PlanFeature,
  PlanTier,
  PlansListResponse,
  PlansSummary,
  RatingResultsResponse,
  RatingSummary,
  ReferenceCountry,
  ReferenceCurrency,
  ReferenceTimezone,
  APIKeysResponse,
  APIKey,
  SettingsSummary,
  Subscription,
  SubscriptionItem,
  SubscriptionsListResponse,
  SubscriptionsSummary,
  Product,
  ProductsListResponse,
  Feature,
  FeaturesListResponse,
  ProductFeature,
  PromotionCodeResponse,
  PromotionCodesListResponse,
  TaxRate,
  TaxRatesListResponse,
  TaxesSummary,
  TestClock,
  TestClocksResponse,
  TestClockResponse,
  UsageAggregatesResponse,
  UsageEvent,
  UsageEventsResponse,
  UsageSummary,
  AIAssistantOverviewResponse,
  AIAssistantRunsResponse,
  AIAssistantRunDetailResponse,
  AIAssistantCreateRunRequest,
  AIAssistantWorkflowPreviewResponse,
  AIPromptCreateRequest,
  AIPromptResponse,
  AIPromptTokenSearchResponse,
  AIThreadDetailResponse,
  AIThreadListResponse,
  AIWorkflowListResponse,
  AIWorkflowDetailResponse,
  AIWorkflowCreateRequest,
  AIWorkflowApproveRequest,
  AIScheduledJob,
  AIJobsListResponse,
  CreateProductRequest,
  UpdateProductRequest,
  ProductCreateBootstrap,
  ReconciliationMismatch
} from "./types";
import { getOrgId } from "./auth";


export type ApiConfig = {
  baseUrl?: string;
  orgId?: string;
  idempotencyKey?: string;
  retry?: {
    maxAttempts?: number;
    baseDelayMs?: number;
    maxDelayMs?: number;
  };
};

type ApiErrorPayload = {
  error?: string;
  code?: string;
  message?: string;
  detail?: string;
  details?: Array<{
    field?: string;
    message?: string;
  }>;
};

const defaultConfig: ApiConfig = {
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? "",
  orgId: import.meta.env.VITE_ORG_ID ?? ""
};
const adminBasePath = "/admin/v1";
const defaultPageSize = 20;
const defaultCSRFTokenCookie = "rz_admin_csrf";

const friendlyErrorMessages: Record<string, string> = {
  invalid_email: "Invalid email address.",
  overlapping_invoice_format: "Invoice format overlaps with an existing active range.",
  invalid_sequence_scope: "Invalid sequence scope.",
  invalid_role: "Invalid role selection.",
  invalid_status: "Invalid status value.",
  invalid_unit: "Unit cannot be empty.",
  invalid_aggregation: "Invalid aggregation value.",
  invalid_interval: "Invalid billing interval.",
  invalid_country: "Invalid country code.",
  invalid_timezone: "Invalid timezone selection.",
  invalid_plan_price: "Invalid plan price selection.",
  invalid_quantity: "Quantity must be a positive number.",
  missing_items: "At least one subscription item is required.",
  invalid_cursor: "Pagination cursor is invalid.",
  not_found: "Resource not found.",
  invalid_intent: "Invalid intent selection.",
  invalid_time_range: "Invalid time range.",
  invalid_prompt: "Prompt is required.",
  invalid_credentials: "Invalid credentials.",
  no_organization: "No organization is associated with this account.",
  missing_token: "Missing authentication token.",
  missing_org_id: "Organization is required. Please select an organization.",
  invalid_session: "Session is invalid or expired.",
  session_not_found: "Session was not found.",
  password_change_required: "Password change is required before continuing.",
  skip_not_allowed: "Skipping password change is not allowed in this environment.",
  invalid_invite: "Invite is invalid or expired.",
  invalid_organization: "Organization ID is invalid.",
  invalid_test_clock: "Invalid test clock selection.",
  not_organization_member: "You do not have access to that organization.",
  invalid_meter: "Meter code is invalid.",
  invalid_coupon: "Coupon payload is invalid.",
  invalid_segment: "Segment is invalid or inactive.",
  segment_exists: "Segment already exists.",
  invalid_promotion_code: "Promotion code payload is invalid.",
  promotion_code_inactive: "Promotion code is inactive.",
  promotion_code_max_redemptions_reached: "Promotion code has reached its redemption limit.",
  invalid_json: "Payload must be valid JSON.",
  unbalanced_entry: "Ledger entries must be balanced.",
  slug_already_exists: "Organization slug already exists.",
  organization_slug_exists: "Organization slug already exists."
  ,
  ai_quota_exceeded: "AI quota is exhausted for the configured provider. Retry shortly or update the provider plan.",
  ai_rate_limited: "AI provider is rate limiting requests right now. Retry shortly.",
  ai_provider_auth_failed: "AI provider credentials are invalid or do not have access to the configured model.",
  ai_timeout: "AI provider did not finish in time. Retry in a moment.",
  ai_model_not_available: "Configured AI model is not available.",
  ai_provider_failed: "AI provider failed to generate a response. Retry in a moment."
};

function orgIdFromPath(): string {
  if (typeof window === "undefined") return "";
  const match = window.location.pathname.match(/^\/organizations\/([^/]+)/);
  if (!match) return "";
  const raw = match[1];
  if (!raw || raw === "new") return "";
  return raw;
}

function getCookieValue(name: string): string {
  if (typeof document === "undefined") return "";
  const cookies = document.cookie ? document.cookie.split(";") : [];
  for (const raw of cookies) {
    const trimmed = raw.trim();
    if (!trimmed) continue;
    const [key, ...rest] = trimmed.split("=");
    if (key === name) {
      return decodeURIComponent(rest.join("="));
    }
  }
  return "";
}

function getCSRFToken(): string {
  const direct = getCookieValue(defaultCSRFTokenCookie);
  if (direct) return direct;
  if (typeof document === "undefined") return "";
  const cookies = document.cookie ? document.cookie.split(";") : [];
  for (const raw of cookies) {
    const trimmed = raw.trim();
    if (!trimmed) continue;
    const [key, ...rest] = trimmed.split("=");
    if (key && key.endsWith("_csrf")) {
      return decodeURIComponent(rest.join("="));
    }
  }
  return "";
}

function buildQuery(params: Record<string, string | number | boolean | undefined>) {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === "") {
      return;
    }
    search.set(key, String(value));
  });
  const query = search.toString();
  return query ? `?${query}` : "";
}

function withDefaultPageSize<T extends { page_size?: number }>(params?: T): T {
  const resolved = { ...(params ?? {}) } as T;
  if (resolved.page_size == null) {
    resolved.page_size = defaultPageSize as T["page_size"];
  }
  return resolved;
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function parseRetryAfter(value: string | null): number | null {
  if (!value) {
    return null;
  }
  const seconds = Number.parseInt(value, 10);
  if (!Number.isNaN(seconds)) {
    return seconds * 1000;
  }
  const date = new Date(value);
  if (!Number.isNaN(date.getTime())) {
    const delta = date.getTime() - Date.now();
    return delta > 0 ? delta : 0;
  }
  return null;
}

function parseErrorPayload(raw: string): ApiErrorPayload | null {
  const trimmed = raw.trim();
  if (!trimmed || !trimmed.startsWith("{")) {
    return null;
  }
  try {
    return JSON.parse(trimmed) as ApiErrorPayload;
  } catch {
    return null;
  }
}

function formatValidationDetails(details?: ApiErrorPayload["details"]): string {
  if (!details || details.length === 0) {
    return "";
  }
  return details
    .map((detail) => {
      const field = typeof detail?.field === "string" ? detail.field.trim() : "";
      const message = typeof detail?.message === "string" ? detail.message.trim() : "";
      if (field && message) return `${field}: ${message}`;
      if (message) return message;
      return field;
    })
    .filter(Boolean)
    .join("\n");
}

function formatErrorMessage(raw: string, status?: number): string {
  const trimmed = raw.trim();
  if (!trimmed) {
    return status ? `Request failed (${status})` : "Request failed";
  }
  const payload = parseErrorPayload(trimmed);
  const code =
    (payload?.error && String(payload.error)) ||
    (payload?.code && String(payload.code)) ||
    "";
  const message =
    (payload?.message && String(payload.message)) ||
    (payload?.detail && String(payload.detail)) ||
    "";
  const detailsMessage = formatValidationDetails(payload?.details);

  if (code && friendlyErrorMessages[code]) {
    return detailsMessage ? `${friendlyErrorMessages[code]}\n${detailsMessage}` : friendlyErrorMessages[code];
  }
  if (detailsMessage) {
    return detailsMessage;
  }
  if (message) {
    return message;
  }
  if (friendlyErrorMessages[trimmed]) {
    return friendlyErrorMessages[trimmed];
  }
  if (trimmed.startsWith("request_failed:")) {
    const statusCode = trimmed.split(":", 2)[1];
    return statusCode ? `Request failed (${statusCode})` : "Request failed";
  }
  return trimmed;
}

function generateIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  const now = Date.now().toString(36);
  const rand = Math.random().toString(36).slice(2, 10);
  return `${now}-${rand}`;
}

function requiresOrgHeader(path: string, method?: string): boolean {
  const normalized = path.startsWith(adminBasePath) ? path : `${adminBasePath}${path.startsWith("/") ? path : `/${path}`}`
  if (normalized.startsWith(`${adminBasePath}/auth/`)) {
    return false
  }
  if (normalized.startsWith(`${adminBasePath}/reference/`)) {
    return false
  }
  if (normalized === `${adminBasePath}/organizations` && (!method || method.toUpperCase() === "GET" || method.toUpperCase() === "POST")) {
    return false
  }
  if (normalized.startsWith(`${adminBasePath}/organizations/invites/`) && normalized.endsWith("/accept")) {
    return false
  }
  return true
}

async function request<T>(path: string, init?: RequestInit, config: ApiConfig = defaultConfig): Promise<T> {
  const base = config.baseUrl ?? "";
  const url = base.endsWith("/") ? `${base}${path.replace(/^\//, "")}` : `${base}${path}`;
  const headers = new Headers(init?.headers ?? {});
  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const resolvedOrgId = (
    [config.orgId, getOrgId(), orgIdFromPath(), defaultConfig.orgId]
      .map((value) => (typeof value === "string" ? value.trim() : ""))
      .find((value) => value)
  ) || "";
  if (requiresOrgHeader(path, init?.method)) {
    if (!resolvedOrgId) {
      throw new Error("missing_org_id")
    }
    headers.set("X-Org-ID", resolvedOrgId);
  } else if (resolvedOrgId) {
    headers.set("X-Org-ID", resolvedOrgId);
  }
  const method = (init?.method ?? "GET").toUpperCase();
  const shouldAttachIdempotency = method !== "GET" && method !== "HEAD" && method !== "OPTIONS";
  const shouldAttachCSRF = shouldAttachIdempotency;
  if (shouldAttachCSRF && !headers.has("X-CSRF-Token")) {
    const csrfToken = getCSRFToken();
    if (csrfToken) {
      headers.set("X-CSRF-Token", csrfToken);
    }
  }
  if (shouldAttachIdempotency && !headers.has("Idempotency-Key")) {
    const explicitKey = config.idempotencyKey?.trim();
    if (explicitKey) {
      headers.set("Idempotency-Key", explicitKey);
    } else if (init?.body && typeof init.body === "string" && headers.get("Content-Type")?.includes("application/json")) {
      try {
        const parsed = JSON.parse(init.body) as { idempotency_key?: string };
        if (parsed?.idempotency_key && typeof parsed.idempotency_key === "string") {
          headers.set("Idempotency-Key", parsed.idempotency_key);
        } else {
          headers.set("Idempotency-Key", generateIdempotencyKey());
        }
      } catch {
        // ignore invalid JSON for idempotency header extraction
      }
    } else {
      headers.set("Idempotency-Key", generateIdempotencyKey());
    }
  }

  if (
    shouldAttachIdempotency &&
    init?.body &&
    typeof init.body === "string" &&
    headers.get("Content-Type")?.includes("application/json")
  ) {
    try {
      const parsed = JSON.parse(init.body) as Record<string, unknown>;
      if (parsed && typeof parsed === "object" && !("idempotency_key" in parsed)) {
        const key = headers.get("Idempotency-Key") ?? generateIdempotencyKey();
        headers.set("Idempotency-Key", key);
        parsed.idempotency_key = key;
        init = { ...init, body: JSON.stringify(parsed) };
      }
    } catch {
      // ignore non-JSON payloads
    }
  }

  const retryConfig = {
    maxAttempts: config.retry?.maxAttempts ?? 3,
    baseDelayMs: config.retry?.baseDelayMs ?? 250,
    maxDelayMs: config.retry?.maxDelayMs ?? 2000
  };
  const shouldRetryMethod = method === "GET" || method === "HEAD" || method === "OPTIONS";
  let attempt = 0;

  while (true) {
    attempt += 1;
    let res: Response;
    try {
      res = await fetch(url, { ...init, headers, credentials: init?.credentials ?? "include" });
    } catch (err) {
      if (!shouldRetryMethod || attempt >= retryConfig.maxAttempts) {
        throw err instanceof Error ? err : new Error("network_error");
      }
      const delay = Math.min(
        retryConfig.maxDelayMs,
        retryConfig.baseDelayMs * Math.pow(2, attempt - 1)
      );
      await sleep(delay);
      continue;
    }

    if (!res.ok) {
      const shouldRetryStatus = res.status === 429 || res.status === 500 || res.status === 502 || res.status === 503 || res.status === 504;
      if (shouldRetryMethod && shouldRetryStatus && attempt < retryConfig.maxAttempts) {
        const retryAfter = parseRetryAfter(res.headers.get("retry-after"));
        const delay = retryAfter ?? Math.min(
          retryConfig.maxDelayMs,
          retryConfig.baseDelayMs * Math.pow(2, attempt - 1)
        );
        await sleep(delay);
        continue;
      }
      const text = await res.text();
      throw new Error(formatErrorMessage(text, res.status));
    }

    if (res.status === 204) {
      return {} as T;
    }
    const contentType = res.headers.get("content-type") ?? "";
    if (contentType.includes("application/json")) {
      return (await res.json()) as T;
    }
    return (await res.text()) as T;
  }
}

export const api = {
  auth: {
    login: (payload: AdminLoginRequest, config?: ApiConfig) =>
      request<AdminLoginResponse>(
        `${adminBasePath}/auth/login`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    me: (config?: ApiConfig) =>
      request<AdminMeResponse>(`${adminBasePath}/auth/me`, undefined, { ...defaultConfig, ...config }),
    logout: (config?: ApiConfig) =>
      request<AdminLogoutResponse>(`${adminBasePath}/auth/logout`, { method: "POST" }, { ...defaultConfig, ...config }),
    switchOrg: (orgId: string, config?: ApiConfig) =>
      request<AdminSwitchOrgResponse>(
        `${adminBasePath}/auth/using/${orgId}`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
    skipPasswordChange: (config?: ApiConfig) =>
      request<AdminSkipPasswordResponse>(
        `${adminBasePath}/auth/skip-password-change`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
    changePassword: (payload: AdminChangePasswordRequest, config?: ApiConfig) =>
      request<AdminChangePasswordResponse>(
        `${adminBasePath}/auth/change-password`,
        { method: "PUT", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    listSessions: (config?: ApiConfig) =>
      request<AdminSessionsResponse>(
        `${adminBasePath}/profile/sessions`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    revokeSession: (sessionId: string, config?: ApiConfig) =>
      request<AdminRevokeSessionResponse>(
        `${adminBasePath}/profile/sessions/${sessionId}/revoke`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
    revokeOtherSessions: (config?: ApiConfig) =>
      request<AdminRevokeSessionResponse>(
        `${adminBasePath}/profile/sessions/revoke-others`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      )
  },
  organizations: {
    list: (config?: ApiConfig) =>
      request<OrganizationListItem[]>(`${adminBasePath}/organizations`, undefined, { ...defaultConfig, ...config }),
    get: (orgId: string, config?: ApiConfig) =>
      request<Organization>(
        `${adminBasePath}/organizations/${orgId}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    create: (
      payload: { name: string; country_code?: string; timezone_name?: string },
      config?: ApiConfig
    ) =>
      request<Organization>(
        `${adminBasePath}/organizations`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    update: (
      orgId: string,
      payload: { name?: string; country_code?: string; timezone_name?: string },
      config?: ApiConfig
    ) =>
      request<Organization>(
        `${adminBasePath}/organizations/${orgId}`,
        { method: "PATCH", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    setBillingPreferences: (
      orgId: string,
      payload: {
        currency?: string;
        timezone?: string;
        invoice_prefix?: string;
        invoice_number_format?: string;
        invoice_sequence_scope?: string;
      },
      config?: ApiConfig
    ) =>
      request<{ status: string }>(
        `${adminBasePath}/organizations/${orgId}/billing-preferences`,
        { method: "PUT", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    listInvoiceFormats: (orgId: string, config?: ApiConfig) =>
      request<InvoiceNumberFormat[]>(
        `${adminBasePath}/organizations/${orgId}/invoice-formats`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    listMembers: (orgId: string, config?: ApiConfig) =>
      request<OrganizationMemberInfo[]>(
        `${adminBasePath}/organizations/${orgId}/members`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    createInvoiceFormat: (
      orgId: string,
      payload: {
        format: string;
        sequence_scope: string;
        effective_from: string;
        effective_to?: string;
      },
      config?: ApiConfig
    ) =>
      request<InvoiceNumberFormat>(
        `${adminBasePath}/organizations/${orgId}/invoice-formats`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    closeInvoiceFormat: (
      orgId: string,
      formatId: string,
      payload: { effective_to: string },
      config?: ApiConfig
    ) =>
      request<InvoiceNumberFormat>(
        `${adminBasePath}/organizations/${orgId}/invoice-formats/${formatId}/close`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    inviteMembers: (
      orgId: string,
      payload: { invites: Array<{ email: string; role: string }> },
      config?: ApiConfig
    ) =>
      request<{ status: string }>(
        `${adminBasePath}/organizations/${orgId}/invites`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    listSessions: (orgId: string, config?: ApiConfig) =>
      request<AdminSessionsResponse>(
        `${adminBasePath}/organizations/${orgId}/sessions`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    revokeSession: (orgId: string, sessionId: string, config?: ApiConfig) =>
      request<AdminRevokeSessionResponse>(
        `${adminBasePath}/organizations/${orgId}/sessions/${sessionId}/revoke`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      )
  },
  reference: {
    countries: (config?: ApiConfig) =>
      request<ReferenceCountry[]>(
        `${adminBasePath}/reference/countries`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    timezones: (country: string, config?: ApiConfig) =>
      request<ReferenceTimezone[]>(
        `${adminBasePath}/reference/timezones?country=${encodeURIComponent(country)}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    currencies: (config?: ApiConfig) =>
      request<ReferenceCurrency[]>(
        `${adminBasePath}/reference/currencies`,
        undefined,
        { ...defaultConfig, ...config }
      ),
  },
  dashboard: {
    summary: (config?: ApiConfig) =>
      request<DashboardSummary>(`${adminBasePath}/dashboard`, undefined, { ...defaultConfig, ...config })
  },
  customers: {
    summary: (config?: ApiConfig) =>
      request<CustomersSummary>(`${adminBasePath}/customers/summary`, undefined, { ...defaultConfig, ...config }),
    create: (
      payload: {
        name: string;
        email: string;
        external_id?: string;
        currency?: string;
        test_clock?: string;
        test_clock_id?: string;
        idempotency_key?: string;
      },
      config?: ApiConfig
    ) =>
      request<Customer>(
        `${adminBasePath}/customers`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    update: (
      customerId: string,
      payload: { name?: string; email?: string; external_id?: string; currency?: string; test_clock?: string; test_clock_id?: string },
      config?: ApiConfig
    ) =>
      request<Customer>(
        `${adminBasePath}/customers/${customerId}`,
        { method: "PATCH", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    get: (customerId: string, config?: ApiConfig) =>
      request<Customer>(`${adminBasePath}/customers/${customerId}`, undefined, { ...defaultConfig, ...config }),
    list: (
      params?: {
        page_token?: string;
        page_size?: number;
        name?: string;
        email?: string;
        currency?: string;
        created_from?: string;
        created_to?: string;
      },
      config?: ApiConfig
    ) =>
      request<CustomersListResponse>(
        `${adminBasePath}/customers${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      )
  },
  plans: {
    summary: (config?: ApiConfig) =>
      request<PlansSummary>(`${adminBasePath}/plans/summary`, undefined, { ...defaultConfig, ...config }),
    create: (
      payload: {
        product_id?: string;
        code: string;
        name: string;
        description?: string;
        active?: boolean;
        idempotency_key?: string;
        prices?: Array<{
          code: string;
          name?: string;
          description?: string;
          price_type: string;
          billing_interval: string;
          billing_interval_count: number;
          aggregate_usage?: string;
          billing_unit?: string;
          meter_id?: string;
          meter_code?: string;
          active?: boolean;
          idempotency_key?: string;
          amounts?: Array<{
            currency: string;
            unit_amount_cents: number;
            minimum_amount_cents?: number;
            maximum_amount_cents?: number;
            effective_from?: string;
            effective_to?: string;
            idempotency_key?: string;
          }>;
          tiers?: Array<{
            tier_mode: string;
            start_quantity: number;
            end_quantity?: number;
            unit_amount_cents?: number;
            flat_amount_cents?: number;
            unit: string;
            idempotency_key?: string;
          }>;
        }>;
      },
      config?: ApiConfig
    ) =>
      request<Plan>(
        `${adminBasePath}/plans`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    update: (
      planId: string,
      payload: { name?: string; description?: string; active?: boolean },
      config?: ApiConfig
    ) =>
      request<Plan>(
        `${adminBasePath}/plans/${planId}`,
        { method: "PATCH", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    get: (planId: string, config?: ApiConfig) =>
      request<Plan>(`${adminBasePath}/plans/${planId}`, undefined, { ...defaultConfig, ...config }),
    createPrice: (
      planId: string,
      payload: {
        code: string;
        name?: string;
        description?: string;
        price_type: string;
        billing_interval: string;
        billing_interval_count: number;
        aggregate_usage?: string;
        billing_unit?: string;
        meter_id?: string;
        meter_code?: string;
        active?: boolean;
        idempotency_key?: string;
      },
      config?: ApiConfig
    ) =>
      request<PlanPrice>(
        `${adminBasePath}/plans/${planId}/prices`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    createAmount: (
      priceId: string,
      payload: {
        currency: string;
        unit_amount_cents: number;
        minimum_amount_cents?: number;
        maximum_amount_cents?: number;
        effective_from?: string;
        effective_to?: string;
        idempotency_key?: string;
      },
      config?: ApiConfig
    ) =>
      request<PlanAmount>(
        `${adminBasePath}/prices/${priceId}/amounts`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    createTier: (
      priceId: string,
      payload: {
        tier_mode: string;
        start_quantity: number;
        end_quantity?: number;
        unit_amount_cents?: number;
        flat_amount_cents?: number;
        unit: string;
        idempotency_key?: string;
      },
      config?: ApiConfig
    ) =>
      request<PlanTier>(
        `${adminBasePath}/prices/${priceId}/tiers`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    list: (
      params?: { page_token?: string; page_size?: number; code?: string; name?: string; active?: string; product_id?: string },
      config?: ApiConfig
    ) =>
      request<PlansListResponse>(
        `${adminBasePath}/plans${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    listFeatures: (planId: string, config?: ApiConfig) =>
      request<PlanFeature[]>(`${adminBasePath}/plans/${planId}/features`, undefined, { ...defaultConfig, ...config }),
    replaceFeatures: (
      planId: string,
      payload: {
        features: Array<{
          feature_id: string;
          enabled: boolean;
          limit_numeric?: number;
          limit_unit?: string;
          reset_period?: string;
        }>;
      },
      config?: ApiConfig
    ) =>
      request<PlanFeature[]>(
        `${adminBasePath}/plans/${planId}/features`,
        { method: "PUT", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  products: {
    list: (
      params?: { page_token?: string; page_size?: number; code?: string; name?: string; active?: string },
      config?: ApiConfig
    ) =>
      request<ProductsListResponse>(
        `${adminBasePath}/products${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    get: (productId: string, params?: { expand?: string }, config?: ApiConfig) =>
      request<Product>(`${adminBasePath}/products/${productId}${buildQuery(params || {})}`, undefined, { ...defaultConfig, ...config }),
    getBootstrap: (config?: ApiConfig) =>
      request<ProductCreateBootstrap>(`${adminBasePath}/products/create`, undefined, { ...defaultConfig, ...config }),
    create: (
      payload: CreateProductRequest,
      config?: ApiConfig
    ) =>
      request<Product>(
        `${adminBasePath}/products`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    update: (
      productId: string,
      payload: UpdateProductRequest,
      config?: ApiConfig
    ) =>
      request<Product>(
        `${adminBasePath}/products/${productId}`,
        { method: "PUT", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  features: {
    list: (
      params?: { page_token?: string; page_size?: number; code?: string; name?: string; active?: string; feature_type?: string },
      config?: ApiConfig
    ) =>
      request<FeaturesListResponse>(
        `${adminBasePath}/features${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    get: (featureId: string, config?: ApiConfig) =>
      request<Feature>(`${adminBasePath}/features/${featureId}`, undefined, { ...defaultConfig, ...config }),
    create: (
      payload: { code: string; name: string; description?: string; feature_type: string; meter_id?: string; active?: boolean },
      config?: ApiConfig
    ) =>
      request<Feature>(
        `${adminBasePath}/features`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    update: (
      featureId: string,
      payload: { code?: string; name?: string; description?: string; feature_type?: string; meter_id?: string; active?: boolean },
      config?: ApiConfig
    ) =>
      request<Feature>(
        `${adminBasePath}/features/${featureId}`,
        { method: "PATCH", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  productFeatures: {
    listByProduct: (productId: string, config?: ApiConfig) =>
      request<ProductFeature[]>(`${adminBasePath}/products/${productId}/features`, undefined, { ...defaultConfig, ...config }),
    replace: (
      productId: string,
      payload: { feature_ids: string[] },
      config?: ApiConfig
    ) =>
      request<{ status: string }>(
        `${adminBasePath}/products/${productId}/features`,
        { method: "PUT", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  meters: {
    list: (
      params?: { page_token?: string; page_size?: number; code?: string; name?: string; active?: string },
      config?: ApiConfig
    ) =>
      request<MetersListResponse>(
        `${adminBasePath}/meters${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    get: (meterId: string, config?: ApiConfig) =>
      request<Meter>(`${adminBasePath}/meters/${meterId}`, undefined, { ...defaultConfig, ...config }),
    create: (
      payload: { code: string; name: string; aggregation: string; unit: string; active?: boolean },
      config?: ApiConfig
    ) =>
      request<Meter>(
        `${adminBasePath}/meters`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    update: (
      meterId: string,
      payload: { code?: string; name?: string; aggregation?: string; unit?: string; active?: boolean },
      config?: ApiConfig
    ) =>
      request<Meter>(
        `${adminBasePath}/meters/${meterId}`,
        { method: "PATCH", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  ledger: {
    listTransactions: (
      params?: {
        page_token?: string;
        page_size?: number;
        customer_id?: string;
        subscription_id?: string;
        invoice_id?: string;
        currency?: string;
        source_type?: string;
        source_id?: string;
        occurred_from?: string;
        occurred_to?: string;
      },
      config?: ApiConfig
    ) =>
      request<LedgerTransactionsResponse>(
        `${adminBasePath}/ledger/transactions${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    createTransaction: (
      payload: {
        currency: string;
        source_type: string;
        source_id: string;
        customer_id?: string;
        subscription_id?: string;
        invoice_id?: string;
        entries: Array<{ account_code: string; entry_type: string; amount_cents: number; description?: string }>;
        occurred_at?: string;
      },
      config?: ApiConfig
    ) =>
      request<LedgerTransaction>(
        `${adminBasePath}/ledger/transactions`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  testClock: {
    get: (config?: ApiConfig) =>
      request<TestClockResponse>(`${adminBasePath}/test-clock`, undefined, { ...defaultConfig, ...config }),
    list: (config?: ApiConfig) =>
      request<TestClocksResponse>(`${adminBasePath}/test-clocks`, undefined, { ...defaultConfig, ...config }),
    getById: (testClockId: string, config?: ApiConfig) =>
      request<TestClock>(`${adminBasePath}/test-clocks/${testClockId}`, undefined, { ...defaultConfig, ...config }),
    upsert: (payload: { frozen_time: number; name?: string; status?: string }, config?: ApiConfig) =>
      request<TestClock>(
        `${adminBasePath}/test-clocks`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    advance: (testClockId: string, payload: { frozen_time: number }, config?: ApiConfig) =>
      request<TestClock>(
        `${adminBasePath}/test-clocks/${testClockId}/advance`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    pause: (testClockId: string, config?: ApiConfig) =>
      request<TestClock>(`${adminBasePath}/test-clocks/${testClockId}/pause`, { method: "POST" }, { ...defaultConfig, ...config }),
    resume: (testClockId: string, config?: ApiConfig) =>
      request<TestClock>(`${adminBasePath}/test-clocks/${testClockId}/resume`, { method: "POST" }, { ...defaultConfig, ...config }),
  },
  usage: {
    summary: (config?: ApiConfig) =>
      request<UsageSummary>(`${adminBasePath}/usage/summary`, undefined, { ...defaultConfig, ...config }),
    ingest: (payload: { meter_code: string; customer_id: string; value: number; recorded_at: string; idempotency_key?: string } | { events: Array<{ meter_code: string; customer_id: string; value: number; recorded_at: string; idempotency_key?: string }> }, config?: ApiConfig) =>
      request<{ status: string; accepted: number }>(
        `${adminBasePath}/usage/events`,
        { method: "POST", body: JSON.stringify(("events" in payload) ? payload : { events: [payload] }) },
        { ...defaultConfig, ...config }
      ),
    events: (params?: { page_token?: string; page_size?: number; customer_id?: string; meter_id?: string; status?: string; recorded_from?: string; recorded_to?: string }, config?: ApiConfig) =>
      request<UsageEventsResponse>(
        `${adminBasePath}/usage/events${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    listEvents: (params?: { page_token?: string; page_size?: number; customer_id?: string; meter_id?: string; status?: string; recorded_from?: string; recorded_to?: string }, config?: ApiConfig) =>
      request<UsageEventsResponse>(
        `${adminBasePath}/usage/events${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    listAggregates: (params?: { page_token?: string; page_size?: number; customer_id?: string; plan_price_id?: string }, config?: ApiConfig) =>
      request<UsageAggregatesResponse>(
        `${adminBasePath}/rating/aggregates${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      )
  },
  rating: {
    summary: (config?: ApiConfig) =>
      request<RatingSummary>(`${adminBasePath}/rating/summary`, undefined, { ...defaultConfig, ...config }),
    listResults: (params?: { page_token?: string; page_size?: number; customer_id?: string; plan_price_id?: string }, config?: ApiConfig) =>
      request<RatingResultsResponse>(
        `${adminBasePath}/rating/results${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    results: (params?: { page_token?: string; page_size?: number; customer_id?: string; subscription_id?: string; plan_price_id?: string; meter_id?: string; usage_event_id?: string; window_start_from?: string; window_start_to?: string }, config?: ApiConfig) =>
      request<RatingResultsResponse>(
        `${adminBasePath}/rating/results${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    aggregates: (params?: { page_token?: string; page_size?: number; customer_id?: string; subscription_id?: string; plan_price_id?: string; meter_id?: string; period_start_from?: string; period_start_to?: string }, config?: ApiConfig) =>
      request<UsageAggregatesResponse>(
        `${adminBasePath}/rating/aggregates${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      )
  },
  invoices: {
    summary: (config?: ApiConfig) =>
      request<InvoicesSummary>(`${adminBasePath}/invoices/summary`, undefined, { ...defaultConfig, ...config }),
    list: (params?: { page_token?: string; page_size?: number; customer_id?: string; subscription_id?: string; status?: string; number?: string }, config?: ApiConfig) =>
      request<InvoicesListResponse>(
        `${adminBasePath}/invoices${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    get: (invoiceId: string, config?: ApiConfig) =>
      request<InvoiceDetailResponse>(`${adminBasePath}/invoices/${invoiceId}`, undefined, { ...defaultConfig, ...config }),
    create: (payload: { customer_id: string; currency: string; period_start: string; period_end: string }, config?: ApiConfig) =>
      request<Invoice>(
        `${adminBasePath}/invoices`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    void: (invoiceId: string, payload?: { reason?: string; attachment_url?: string; note?: string }, config?: ApiConfig) =>
      request<InvoiceDetailResponse>(
        `${adminBasePath}/invoices/${invoiceId}/void`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    issue: (invoiceId: string, config?: ApiConfig) =>
      request<Invoice>(
        `${adminBasePath}/invoices/${invoiceId}/issue`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
    resend: (invoiceId: string, config?: ApiConfig) =>
      request<InvoiceResendResponse>(
        `${adminBasePath}/invoices/${invoiceId}/resend`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
    open: (invoiceId: string, config?: ApiConfig) =>
      request<InvoiceDetailResponse>(
        `${adminBasePath}/invoices/${invoiceId}/open`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
    pay: (invoiceId: string, payload?: { reason?: string; attachment_url?: string; note?: string }, config?: ApiConfig) =>
      request<InvoiceDetailResponse>(
        `${adminBasePath}/invoices/${invoiceId}/pay`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    markPaid: (invoiceId: string, payload?: { reason?: string; attachment_url?: string; note?: string }, config?: ApiConfig) =>
      request<InvoiceDetailResponse>(
        `${adminBasePath}/invoices/${invoiceId}/mark-paid`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    generate: (payload: { customer_id?: string; subscription_id?: string; currency?: string; period_start: string; period_end: string; issue_at?: string; due_at?: string }, config?: ApiConfig) =>
      request<InvoiceDetailResponse>(
        `${adminBasePath}/invoices/generate`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
  },
  subscriptions: {
    summary: (config?: ApiConfig) =>
      request<SubscriptionsSummary>(`${adminBasePath}/subscriptions/summary`, undefined, { ...defaultConfig, ...config }),
    list: (params?: { page_token?: string; page_size?: number; customer_id?: string; status?: string }, config?: ApiConfig) =>
      request<SubscriptionsListResponse>(
        `${adminBasePath}/subscriptions${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    get: (subscriptionId: string, config?: ApiConfig) =>
      request<Subscription>(`${adminBasePath}/subscriptions/${subscriptionId}`, undefined, { ...defaultConfig, ...config }),
    create: (payload: { 
      customer_id: string; 
      plan_id: string; 
      currency: string; 
      start_at?: string; 
      current_period_start?: string;
      current_period_end?: string;
      trial_end?: string;
      cancel_at?: string;
      status?: string;
      items: Array<{ plan_price_id: string; quantity: number }> 
    }, config?: ApiConfig) =>
      request<Subscription>(
        `${adminBasePath}/subscriptions`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    cancel: (subscriptionId: string, payload: { cancel_at?: string; cancel_now?: boolean }, config?: ApiConfig) =>
      request<Subscription>(
        `${adminBasePath}/subscriptions/${subscriptionId}/cancel`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    update: (subscriptionId: string, payload: { status?: string; cancel_at?: string; canceled_at?: string; ended_at?: string }, config?: ApiConfig) =>
      request<Subscription>(
        `${adminBasePath}/subscriptions/${subscriptionId}`,
        { method: "PATCH", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    createItem: (subscriptionId: string, payload: { plan_price_id: string; quantity: number; start_at?: string; end_at?: string }, config?: ApiConfig) =>
      request<SubscriptionItem>(
        `${adminBasePath}/subscriptions/${subscriptionId}/items`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    redeemPromotionCode: (subscriptionId: string, payload: { code: string }, config?: ApiConfig) =>
      request<CouponResponse>(
        `${adminBasePath}/subscriptions/${subscriptionId}/promotion-code`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
  },
  coupons: {
    list: (config?: ApiConfig) =>
      request<CouponsListResponse>(`${adminBasePath}/coupons`, undefined, { ...defaultConfig, ...config }),
    create: (
      payload: {
        name: string;
        type: string;
        amount_cents?: number;
        percentage?: number;
        duration: string;
        duration_months?: number;
        currency?: string;
        valid_from?: string;
        valid_until?: string;
        auto_apply?: boolean;
        target_segment?: string;
      },
      config?: ApiConfig
    ) =>
      request<CouponResponse>(
        `${adminBasePath}/coupons`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    listPromotionCodes: (config?: ApiConfig) =>
      request<PromotionCodesListResponse>(`${adminBasePath}/promotion-codes`, undefined, { ...defaultConfig, ...config }),
    createPromotionCode: (
      payload: {
        coupon_id: string;
        code: string;
        active?: boolean;
        max_redemptions?: number;
      },
      config?: ApiConfig
    ) =>
      request<PromotionCodeResponse>(
        `${adminBasePath}/promotion-codes`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    listSegments: (params?: { scope?: string; include_inactive?: boolean }, config?: ApiConfig) =>
      request<BillingSegmentsListResponse>(
        `${adminBasePath}/segments${buildQuery(params ?? {})}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    createSegment: (
      payload: {
        key: string;
        name: string;
        scope?: string;
        description?: string;
        active?: boolean;
      },
      config?: ApiConfig
    ) =>
      request<BillingSegmentResponse>(
        `${adminBasePath}/segments`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  taxes: {
    summary: (config?: ApiConfig) =>
      request<TaxesSummary>(`${adminBasePath}/taxes/summary`, undefined, { ...defaultConfig, ...config }),
    list: (params?: { page_token?: string; page_size?: number; code?: string; name?: string; active?: string; created_from?: string; created_to?: string }, config?: ApiConfig) =>
      request<TaxRatesListResponse>(
        `${adminBasePath}/taxes${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    listRates: (params?: { page_token?: string; page_size?: number; active?: string }, config?: ApiConfig) =>
      request<TaxRatesListResponse>(
        `${adminBasePath}/taxes${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    create: (payload: { code: string; name: string; percentage: number; inclusive?: boolean; active?: boolean }, config?: ApiConfig) =>
      request<TaxRate>(
        `${adminBasePath}/taxes`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    createRate: (payload: { code: string; name: string; percentage: number; inclusive?: boolean; active?: boolean }, config?: ApiConfig) =>
      request<TaxRate>(
        `${adminBasePath}/taxes`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
  },
  payments: {
    summary: (config?: ApiConfig) =>
      request<PaymentsSummary>(`${adminBasePath}/payments/summary`, undefined, { ...defaultConfig, ...config }),
    list: (params?: { page_token?: string; page_size?: number; customer_id?: string; status?: string; provider?: string; invoice_id?: string; created_from?: string; created_to?: string }, config?: ApiConfig) =>
      request<PaymentsListResponse>(
        `${adminBasePath}/payments${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      )
  },
  apps: {
    catalog: (config?: ApiConfig) =>
      request<AppsCatalogResponse>(`${adminBasePath}/apps/catalog`, undefined, { ...defaultConfig, ...config }),
    installations: (config?: ApiConfig) =>
      request<AppsInstallationsResponse>(`${adminBasePath}/apps/installations`, undefined, { ...defaultConfig, ...config }),
    install: (payload: { app_id: string; auth_method: string; credentials?: Record<string, unknown>; config?: Record<string, unknown> }, config?: ApiConfig) =>
      request<AppInstallation>(
        `${adminBasePath}/apps/installations`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    updateInstallation: (installationId: string, payload: { status?: string; auth_method?: string; credentials?: Record<string, unknown>; config?: Record<string, unknown> }, config?: ApiConfig) =>
      request<AppInstallation>(
        `${adminBasePath}/apps/installations/${installationId}`,
        { method: "PATCH", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    startStripeOAuth: (config?: ApiConfig) =>
      request<{ url: string }>(`${adminBasePath}/apps/oauth/stripe/start`, undefined, { ...defaultConfig, ...config }),
  },
  ai: {
    listThreads: (params?: { page_size?: number }, config?: ApiConfig) =>
      request<AIThreadListResponse>(
        `${adminBasePath}/ai/threads${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    createThread: (payload?: { title?: string }, config?: ApiConfig) =>
      request<{ thread: { id: string; title: string; created_at: string; updated_at: string } }>(
        `${adminBasePath}/ai/threads`,
        { method: "POST", body: JSON.stringify(payload ?? {}) },
        { ...defaultConfig, ...config }
      ),
    getThread: (threadId: string, config?: ApiConfig) =>
      request<AIThreadDetailResponse>(
        `${adminBasePath}/ai/threads/${threadId}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    deleteThread: (threadId: string, config?: ApiConfig) =>
      request<{ deleted: boolean }>(
        `${adminBasePath}/ai/threads/${threadId}`,
        { method: "DELETE" },
        { ...defaultConfig, ...config }
      ),
    createPrompt: (payload: AIPromptCreateRequest, config?: ApiConfig) =>
      request<AIPromptResponse>(
        `${adminBasePath}/ai/prompts`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    searchTokens: (payload: { query?: string; kinds?: Array<"resource" | "time"> }, config?: ApiConfig) =>
      request<AIPromptTokenSearchResponse>(
        `${adminBasePath}/ai/tokens/search`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    listJobs: (params?: { page?: number; page_size?: number; status?: string }, config?: ApiConfig) =>
      request<AIJobsListResponse>(
        `${adminBasePath}/ai/jobs${buildQuery(params ?? {})}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    retryJob: (jobId: string, config?: ApiConfig) =>
      request<{ status: string }>(
        `${adminBasePath}/ai/jobs/${jobId}/retry`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
    cancelJob: (jobId: string, config?: ApiConfig) =>
      request<{ status: string }>(
        `${adminBasePath}/ai/jobs/${jobId}`,
        { method: "DELETE" },
        { ...defaultConfig, ...config }
      ),
  },
  aiAssistant: {
    overview: (config?: ApiConfig) =>
      request<AIAssistantOverviewResponse>(`${adminBasePath}/ai-assistant/overview`, undefined, { ...defaultConfig, ...config }),
    createRun: (payload: AIAssistantCreateRunRequest, config?: ApiConfig) =>
      request<AIAssistantRunDetailResponse>(
        `${adminBasePath}/ai-assistant/runs`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    listRuns: (params?: { page_token?: string; page_size?: number }, config?: ApiConfig) =>
      request<AIAssistantRunsResponse>(
        `${adminBasePath}/ai-assistant/runs${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    getRun: (runId: string, config?: ApiConfig) =>
      request<AIAssistantRunDetailResponse>(
        `${adminBasePath}/ai-assistant/runs/${runId}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    previewWorkflowFromRun: (runId: string, config?: ApiConfig) =>
      request<AIAssistantWorkflowPreviewResponse>(
        `${adminBasePath}/ai-assistant/runs/${runId}/workflow-preview`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    createWorkflowFromRun: (runId: string, config?: ApiConfig) =>
      request<AIWorkflowDetailResponse>(
        `${adminBasePath}/ai-assistant/runs/${runId}/workflow`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
  },
  aiWorkflows: {
    list: (params?: { page_token?: string; page_size?: number }, config?: ApiConfig) =>
      request<AIWorkflowListResponse>(
        `${adminBasePath}/ai-workflows${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    create: (payload: AIWorkflowCreateRequest, config?: ApiConfig) =>
      request<AIWorkflowDetailResponse>(
        `${adminBasePath}/ai-workflows`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    get: (workflowId: string, config?: ApiConfig) =>
      request<AIWorkflowDetailResponse>(
        `${adminBasePath}/ai-workflows/${workflowId}`,
        undefined,
        { ...defaultConfig, ...config }
      ),
    approve: (workflowId: string, payload: AIWorkflowApproveRequest, config?: ApiConfig) =>
      request<AIWorkflowDetailResponse>(
        `${adminBasePath}/ai-workflows/${workflowId}/approvals`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    execute: (workflowId: string, config?: ApiConfig) =>
      request<AIWorkflowDetailResponse>(
        `${adminBasePath}/ai-workflows/${workflowId}/execute`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
  },
  auditLogs: {
    summary: (config?: ApiConfig) =>
      request<AuditLogsSummary>(`${adminBasePath}/audit-logs/summary`, undefined, { ...defaultConfig, ...config }),
    list: (params?: { page_token?: string; page_size?: number; actor_id?: string; actor_type?: string; action?: string; resource_type?: string; resource_id?: string; request_id?: string; created_from?: string; created_to?: string }, config?: ApiConfig) =>
      request<AuditLogsListResponse>(
        `${adminBasePath}/audit-logs${buildQuery(withDefaultPageSize(params))}`,
        undefined,
        { ...defaultConfig, ...config }
      )
  },
  apiKeys: {
    list: (config?: ApiConfig) =>
      request<APIKeysResponse>(`${adminBasePath}/api-keys`, undefined, { ...defaultConfig, ...config }),
    create: (payload: { name: string; scopes: string[]; key_type?: string }, config?: ApiConfig) =>
      request<APIKey>(
        `${adminBasePath}/api-keys`,
        { method: "POST", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      ),
    revoke: (keyId: string, config?: ApiConfig) =>
      request<{ status: string }>(
        `${adminBasePath}/api-keys/${keyId}/revoke`,
        { method: "POST" },
        { ...defaultConfig, ...config }
      ),
  },
  featureFlags: {
    list: (config?: ApiConfig) =>
      request<FeatureFlagsResponse>(`${adminBasePath}/feature-flags`, undefined, { ...defaultConfig, ...config }),
    upsert: (payload: FeatureFlagUpsertRequest, config?: ApiConfig) =>
      request<FeatureFlagUpsertResponse>(
        `${adminBasePath}/feature-flags`,
        { method: "PUT", body: JSON.stringify(payload) },
        { ...defaultConfig, ...config }
      )
  },
  settings: {
    summary: (config?: ApiConfig) =>
      request<SettingsSummary>(`${adminBasePath}/settings/summary`, undefined, { ...defaultConfig, ...config }),
    warnings: (config?: ApiConfig) =>
      request<ConfigWarningsResponse>(`${adminBasePath}/warnings`, undefined, { ...defaultConfig, ...config }),
  },
  reconciliation: {
    summary: (config?: ApiConfig) =>
      request<ReconciliationSummaryResponse>(`${adminBasePath}/reconciliation/summary`, undefined, { ...defaultConfig, ...config }),
  }
};
