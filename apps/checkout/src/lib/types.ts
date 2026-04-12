export type PublicInvoice = {
  id: string;
  org_id: string;
  customer_id: string;
  subscription_id?: string | null;
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
  issued_at?: string | null;
  due_at?: string | null;
  paid_at?: string | null;
  voided_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type PublicInvoiceItem = {
  id: string;
  invoice_id: string;
  org_id: string;
  customer_id: string;
  subscription_id?: string | null;
  plan_price_id?: string | null;
  meter_id?: string | null;
  rating_result_id?: string | null;
  line_type: string;
  description?: string | null;
  quantity: number;
  unit_amount_cents: number;
  amount_cents: number;
  currency: string;
  period_start?: string | null;
  period_end?: string | null;
  created_at: string;
};

export type PublicOrganization = {
  id: string;
  name: string;
  slug: string;
  country_code: string;
  timezone_name: string;
};

export type PublicCustomer = {
  id: string;
  org_id: string;
  external_id?: string;
  name: string;
  email: string;
  currency?: string;
  created_at: string;
  updated_at: string;
};

export type PublicInvoiceResponse = {
  invoice: PublicInvoice;
  items: PublicInvoiceItem[];
  organization?: PublicOrganization;
  customer?: PublicCustomer;
  payment_methods: string[];
  payment_configured: boolean;
  billing_country: string;
  expires_at: string;
};

export type PaymentOptionsResponse = {
  payment_methods: string[];
  payment_configured: boolean;
  billing_country: string;
  expires_at: string;
};

export type SupportResponse = {
  status: string;
};

export type CheckoutSessionResponse = {
  status: string;
  checkout_url?: string;
};
