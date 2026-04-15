export type DashboardSummary = {
  mrr_cents: number;
  usage_cents: number;
  open_invoices: number;
  late_events: number;
  alerts: Array<{ title: string; subtitle: string; tag: string }>;
};

export type CustomersSummary = {
  active: number;
  at_risk: number;
  nrr_pct: number;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type Customer = {
  id: string;
  org_id: string;
  external_id?: string;
  name: string;
  email: string;
  currency?: string;
  created_at: string;
  updated_at: string;
};

export type CustomersListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  customers: Customer[];
};

export type Organization = {
  id: string;
  name: string;
  slug: string;
  country_code: string;
  timezone_name: string;
};

export type OrganizationListItem = {
  id: string;
  name: string;
  role: string;
  created_at: string;
};

export type ReferenceCountry = {
  code: string;
  name: string;
};

export type ReferenceTimezone = {
  name: string;
};

export type ReferenceCurrency = {
  code: string;
  name: string;
};

export type InvoiceNumberFormat = {
  id: string;
  org_id: string;
  format: string;
  sequence_scope: string;
  effective_from: string;
  effective_to?: string;
  created_at: string;
  updated_at: string;
};

export type LedgerAccount = {
  id: string;
  org_id: string;
  code: string;
  type: string;
  name: string;
  created_at: string;
};

export type LedgerEntry = {
  id: string;
  transaction_id: string;
  org_id: string;
  account_id?: string;
  account_code: string;
  entry_type: string;
  amount_cents: number;
  currency: string;
  description?: string;
  created_at: string;
};

export type LedgerTransaction = {
  id: string;
  org_id: string;
  currency: string;
  source_type: string;
  source_id: string;
  customer_id?: string;
  subscription_id?: string;
  invoice_id?: string;
  plan_price_id?: string;
  meter_id?: string;
  occurred_at: string;
  posted_at: string;
  created_at: string;
};

export type LedgerTransactionsResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  transactions: LedgerTransaction[];
};

export type PlansSummary = {
  active: number;
  draft: number;
  tiered: number;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type Plan = {
  id: string;
  org_id: string;
  code: string;
  name: string;
  description?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  prices?: PlanPrice[];
  product_id?: string;
};

export type PlanResponse = Plan;

export type PlanPrice = {
  id: string;
  org_id: string;
  plan_id: string;
  meter_id?: string;
  code: string;
  name?: string;
  description?: string;
  price_type: string;
  billing_interval: string;
  billing_interval_count: number;
  aggregate_usage?: string;
  billing_unit?: string;
  meter_code?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  amounts?: PlanAmount[];
  tiers?: PlanTier[];
};

export type PlanAmount = {
  id: string;
  org_id: string;
  plan_price_id: string;
  currency: string;
  unit_amount_cents: number;
  minimum_amount_cents?: number;
  maximum_amount_cents?: number;
  effective_from: string;
  effective_to?: string;
  created_at: string;
  updated_at: string;
};

export type PlanTier = {
  id: string;
  org_id: string;
  plan_price_id: string;
  tier_mode: string;
  start_quantity: number;
  end_quantity?: number;
  unit_amount_cents?: number;
  flat_amount_cents?: number;
  unit: string;
  created_at: string;
  updated_at: string;
};

export type PlansListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  plans: Plan[];
};

export type SubscriptionsSummary = {
  active: number;
  trialing: number;
  pastDue: number;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type Subscription = {
  id: string;
  org_id: string;
  customer_id: string;
  plan_id: string;
  status: string;
  currency: string;
  start_at: string;
  current_period_start: string;
  current_period_end: string;
  trial_end?: string;
  cancel_at?: string;
  canceled_at?: string;
  ended_at?: string;
  created_at: string;
  updated_at: string;
  items?: SubscriptionItem[];
};

export type SubscriptionItem = {
  id: string;
  org_id: string;
  subscription_id: string;
  plan_price_id: string;
  quantity: number;
  start_at: string;
  end_at?: string;
  created_at: string;
  updated_at: string;
};

export type SubscriptionsListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  subscriptions: Subscription[];
};

export type UsageSummary = {
  eventsPerHour: number;
  latePct: number;
  activeMeters: number;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type Meter = {
  id: string;
  org_id: string;
  code: string;
  name: string;
  aggregation: string;
  unit: string;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type MetersListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  meters: Meter[];
};

export type UsageEvent = {
  id: string;
  org_id: string;
  meter_id: string;
  meter_code: string;
  customer_id: string;
  value: number;
  recorded_at: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type UsageEventsResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  events: UsageEvent[];
};

export type RatingSummary = {
  ratedEvents: number;
  avgLatencySec: number;
  replaysToday: number;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type RatingResult = {
  id: string;
  usage_event_id: string;
  customer_id: string;
  subscription_id?: string;
  plan_price_id: string;
  meter_id: string;
  currency: string;
  quantity: number;
  unit_amount_cents: number;
  amount_cents: number;
  source: string;
  window_start: string;
  window_end: string;
  created_at: string;
};

export type RatingResultsResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  results: RatingResult[];
};

export type UsageAggregate = {
  id: string;
  org_id: string;
  customer_id: string;
  subscription_id?: string;
  plan_price_id: string;
  plan_amount_id?: string;
  meter_id: string;
  currency: string;
  period_start: string;
  period_end: string;
  quantity: number;
  amount_cents: number;
  last_event_at?: string;
  created_at: string;
  updated_at: string;
};

export type UsageAggregatesResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  aggregates: UsageAggregate[];
};

export type InvoicesSummary = {
  draft: number;
  open: number;
  paidCents: number;
  highlights: Array<{ number: string; note: string; tag: string }>;
};

export type Invoice = {
  id: string;
  org_id: string;
  customer_id: string;
  subscription_id?: string;
  number: string;
  status: string;
  currency: string;
  subtotal_cents: number;
  tax_cents: number;
  total_cents: number;
  amount_due_cents: number;
  amount_paid_cents: number;
  period_start: string;
  period_end: string;
  issued_at?: string;
  due_at?: string;
  paid_at?: string;
  voided_at?: string;
  created_at: string;
  updated_at: string;
};

export type InvoiceItem = {
  id: string;
  invoice_id: string;
  org_id: string;
  customer_id: string;
  subscription_id?: string;
  plan_price_id?: string;
  meter_id?: string;
  rating_result_id?: string;
  line_type: string;
  description?: string;
  quantity: number;
  unit_amount_cents: number;
  amount_cents: number;
  currency: string;
  period_start?: string;
  period_end?: string;
  created_at: string;
};

export type InvoiceDetailResponse = {
  invoice: Invoice;
  items: InvoiceItem[];
};

export type InvoicePublicLink = {
  token: string;
  url?: string;
  expires_at: string;
};

export type InvoiceResendResponse = {
  status: string;
  public_link?: InvoicePublicLink;
};

export type InvoicesListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  invoices: Invoice[];
};

export type PaymentsSummary = {
  collectedCents: number;
  failed: number;
  retries: number;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type AppDefinition = {
  id: string;
  name: string;
  category: string;
  provider: string;
  description: string;
  capabilities: string[];
  auth_methods: string[];
  credentials_schema?: Record<string, string[]>;
  status: string;
  version: string;
};

export type ConfigWarning = {
  module?: string;
  code: string;
  link?: string;
  metadata?: Record<string, unknown>;
};

export type ConfigWarningsResponse = {
  warnings: ConfigWarning[];
};

export type APIKey = {
  id: string;
  org_id: string;
  name: string;
  key_prefix: string;
  key_type: string;
  scopes: string[];
  allowed_ips: string[];
  allowed_domains: string[];
  status: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at: string;
  updated_at: string;
  key?: string;
};

export type APIKeysResponse = {
  keys: APIKey[];
};

export type ReconciliationMismatch = {
  action: string;
  invoice_id: string;
  created_at: string;
};

export type ReconciliationSummaryResponse = {
  windowDays: number;
  usageMismatches: number;
  ledgerMismatches: number;
  totalMismatches: number;
  latest: ReconciliationMismatch[];
};

export type AppsCatalogResponse = {
  apps: AppDefinition[];
  warnings?: ConfigWarning[];
};

export type AppInstallation = {
  id: string;
  org_id: string;
  app_id: string;
  status: string;
  auth_method: string;
  config: Record<string, unknown>;
  credentials: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type AppsInstallationsResponse = {
  installations: AppInstallation[];
  warnings?: ConfigWarning[];
};

export type Payment = {
  id: string;
  org_id: string;
  customer_id: string;
  invoice_id?: string;
  payment_method_id?: string;
  provider: string;
  provider_ref?: string;
  status: string;
  amount_cents: number;
  currency: string;
  created_at: string;
  updated_at: string;
};

export type PaymentsListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  payments: Payment[];
};

export type TaxesSummary = {
  profiles: number;
  exemptCustomers: number;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type TaxRate = {
  id: string;
  org_id: string;
  code: string;
  name: string;
  percentage: number;
  inclusive: boolean;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type TaxRatesListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  rates: TaxRate[];
};

export type AuditLogsSummary = {
  entries: Array<{ title: string; note: string; tag: string }>;
};

export type AuditLog = {
  id: string;
  org_id: string;
  actor_type: string;
  actor_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  reason?: string;
  request_id?: string;
  created_at: string;
};

export type AuditLogsListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  logs: AuditLog[];
};

export type OrganizationMemberInfo = {
  user_id: string;
  email: string;
  display_name: string;
  role: string;
  created_at: string;
};

export type SettingsSummary = {
  apiKeys: number;
  invoiceFormat: string;
  timezone: string;
  highlights: Array<{ name: string; note: string; tag: string }>;
};

export type FeatureFlag = {
  key: string;
  enabled: boolean;
  rollout: number;
  source: string;
};

export type FeatureFlagsResponse = {
  orgId?: string;
  flags: FeatureFlag[];
};

export type TestClock = {
  id: string;
  org_id: string;
  status: string;
  current_time: string;
  created_at: string;
  updated_at: string;
};

export type TestClockResponse = {
  clock: TestClock | null;
};

export type FeatureFlagUpsertRequest = {
  org_id?: string;
  key: string;
  enabled: boolean;
  rollout: number;
  actor_id?: string;
};

export type FeatureFlagUpsertResponse = {
  status: string;
};

export type AdminLoginRequest = {
  email: string;
  password: string;
};

export type AdminLoginResponse = {
  userId: string;
  email: string;
  orgId: string;
  orgIds: string[];
  mustChangePassword: boolean;
  sessionExpiresAt: string;
};

export type AdminMeResponse = {
  userId: string;
  orgId: string;
  mustChangePassword: boolean;
};

export type AdminChangePasswordRequest = {
  currentPassword: string;
  newPassword: string;
};

export type AdminChangePasswordResponse = {
  status: string;
};

export type AdminLogoutResponse = {
  status: string;
};

export type AdminSkipPasswordResponse = {
  status: string;
};

export type AdminSwitchOrgResponse = {
  status: string;
  orgId: string;
};

export type AdminSession = {
  id: string;
  userId: string;
  email?: string;
  displayName?: string;
  activeOrgId?: string;
  userAgent?: string;
  ipAddress?: string;
  createdAt: string;
  lastSeenAt: string;
  expiresAt: string;
  revokedAt?: string;
  current: boolean;
};

export type AdminSessionsResponse = {
  sessions: AdminSession[];
};

export type AdminRevokeSessionResponse = {
  status: string;
  revokedCount?: number;
  revokedCurrent?: boolean;
};

export type Product = {
  id: string;
  org_id: string;
  code: string;
  name: string;
  description?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  // Expanded read model fields
  features?: Feature[];
  plans?: Plan[];
};

export type ProductsListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  products: Product[];
};

export type CreateProductRequest = {
  code: string;
  name: string;
  description?: string;
  active?: boolean;
  idempotency_key: string;
  feature_ids?: string[];
  plans?: CreateProductPlanInput[];
};

export type CreateProductPlanInput = {
  code: string;
  name: string;
  description?: string;
  active?: boolean;
  prices?: CreateProductPlanPriceInput[];
};

export type CreateProductPlanPriceInput = {
  code: string;
  name?: string;
  description?: string;
  active?: boolean;
  price_type: string;
  billing_interval: string;
  billing_interval_count: number;
  meter_id?: string;
  amounts?: CreateProductPlanAmountInput[];
  tiers?: CreateProductPlanTierInput[];
};

export type CreateProductPlanAmountInput = {
  currency: string;
  unit_amount_cents: number;
  minimum_amount_cents?: number;
  maximum_amount_cents?: number;
  effective_from?: string;
  effective_to?: string;
};

export type CreateProductPlanTierInput = {
  tier_mode: string;
  start_quantity: number;
  end_quantity?: number;
  unit_amount_cents?: number;
  flat_amount_cents?: number;
  unit: string;
};

export type UpdateProductRequest = {
  name?: string;
  description?: string;
  active?: boolean;
  feature_ids?: string[];
};

export type ProductCreateBootstrap = {
  features: Feature[];
  meters: Meter[];
  defaults: {
    active: boolean;
  };
};

export type Feature = {
  id: string;
  org_id: string;
  code: string;
  name: string;
  description?: string;
  feature_type: string;
  meter_id?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type FeaturesListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  features: Feature[];
};

export type ProductFeature = {
  id: string;
  code: string;
  name: string;
  feature_type: string;
  meter_id?: string;
  active: boolean;
};

export type AIAssistantOption = {
  value: string;
  label: string;
  description?: string;
};

export type AIAssistantWorkspace = {
  customer_placeholder: string;
  prompt_placeholder: string;
  default_prompt: string;
  time_ranges: AIAssistantOption[];
  intents: AIAssistantOption[];
  masking_enabled: boolean;
};

export type AIAssistantSummaryCard = {
  id: string;
  label: string;
  value: string;
  sub: string;
  tone: string;
  delta?: string;
};

export type AIAssistantSignalItem = {
  id: string;
  title: string;
  detail: string;
  severity: string;
};

export type AIAssistantSignalPanel = {
  title: string;
  description: string;
  items: AIAssistantSignalItem[];
};

export type AIAssistantGuardrailItem = {
  id: string;
  title: string;
  detail: string;
};

export type AIAssistantGuardrailPanel = {
  title: string;
  description: string;
  items: AIAssistantGuardrailItem[];
};

export type AIAssistantStatusBadge = {
  code: string;
  label: string;
  tone: string;
};

export type AIAssistantRunError = {
  code: string;
  message: string;
};

export type AIAssistantSummary = {
  headline: string;
  metric: string;
  metric_note: string;
};

export type AIAssistantSnapshot = {
  label: string;
  previous: string;
  current: string;
  delta: string;
};

export type AIAssistantDriver = {
  label: string;
  detail: string;
  impact: "high" | "medium" | "low";
};

export type AIAssistantAnomaly = {
  title: string;
  detail: string;
  severity: "watch" | "risk";
};

export type AIAssistantProration = {
  title: string;
  detail: string;
};

export type AIAssistantPlanRecommendation = {
  current_plan: string;
  recommended_plan: string;
  savings_estimate: string;
  billing_impact: string;
  reason_summary: string;
};

export type AIAssistantProductRecommendation = {
  name: string;
  target_segment: string;
  value_proposition: string;
  pricing_model: string;
  pricing_hint: string;
  required_capabilities: string[];
  expected_impact: string;
  priority: string;
};

export type AIAssistantConfidence = {
  level: "high" | "medium" | "low";
  note: string;
};

export type AIAssistantAction = {
  key: string;
  label: string;
  style: "primary" | "secondary";
  path: string;
  disabled?: boolean;
};

export type AIAssistantInsight = {
  summary: AIAssistantSummary;
  snapshot?: AIAssistantSnapshot;
  drivers: AIAssistantDriver[];
  anomalies?: AIAssistantAnomaly[];
  proration?: AIAssistantProration;
  plan_recommendation?: AIAssistantPlanRecommendation;
  product_recommendations?: AIAssistantProductRecommendation[];
  confidence: AIAssistantConfidence;
  data_quality: string;
  actions: AIAssistantAction[];
  generated_at: string;
};

export type AIAssistantRunDetail = {
  id: string;
  status: AIAssistantStatusBadge;
  intent: string;
  time_range: string;
  prompt: string;
  customer_label: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  finished_at?: string;
  duration_ms?: number;
  insight?: AIAssistantInsight | null;
  error?: AIAssistantRunError | null;
};

export type AIAssistantRunHistoryItem = {
  id: string;
  title: string;
  subtitle: string;
  intent: string;
  customer_label: string;
  status: AIAssistantStatusBadge;
  created_at: string;
  duration_ms?: number;
};

export type AIAssistantRunsResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  runs: AIAssistantRunHistoryItem[];
};

export type AIAssistantOverviewResponse = {
  workspace: AIAssistantWorkspace;
  summary_cards: AIAssistantSummaryCard[];
  signals: AIAssistantSignalPanel;
  guardrails: AIAssistantGuardrailPanel;
  runs: AIAssistantRunsResponse;
  active_run_id?: string;
};

export type AIAssistantRunDetailResponse = {
  run: AIAssistantRunDetail;
};

export type AIAssistantCreateRunRequest = {
  customer_ref: string;
  time_range: string;
  intent: string;
  prompt: string;
};

export type AIWorkflowStatusBadge = {
  code: string;
  label: string;
  tone: string;
};

export type AIWorkflowAction = {
  id: string;
  type: string;
  label: string;
  status: "pending" | "running" | "done" | "failed";
  payload: Record<string, unknown>;
  order: number;
  error?: string;
  created_at: string;
  updated_at: string;
};

export type AIWorkflowApproval = {
  id: string;
  actor_id: string;
  status: "approved" | "rejected";
  note?: string;
  created_at: string;
};

export type AIWorkflowListItem = {
  id: string;
  title: string;
  intent: string;
  summary: string;
  status: AIWorkflowStatusBadge;
  created_at: string;
  updated_at: string;
  actions: number;
  source_run_id?: string;
};

export type AIWorkflowDetail = {
  id: string;
  title: string;
  intent: string;
  summary: string;
  status: AIWorkflowStatusBadge;
  source_run_id?: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  finished_at?: string;
  actions: AIWorkflowAction[];
  approvals: AIWorkflowApproval[];
};

export type AIWorkflowListResponse = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
  workflows: AIWorkflowListItem[];
};

export type AIWorkflowDetailResponse = {
  workflow: AIWorkflowDetail;
};

export type AIWorkflowCreateAction = {
  type: string;
  label: string;
  payload?: Record<string, unknown>;
};

export type AIWorkflowCreateRequest = {
  title: string;
  summary: string;
  intent: string;
  source_run_id?: string;
  actions: AIWorkflowCreateAction[];
};

export type AIWorkflowApproveRequest = {
  status: "approved" | "rejected";
  note?: string;
};
