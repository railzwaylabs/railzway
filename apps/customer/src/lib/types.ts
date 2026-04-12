export type PageInfo = {
  next_page_token?: string;
  previous_page_token?: string;
  has_more?: boolean;
};

export type Subscription = {
  id: string;
  org_id: string;
  customer_id: string;
  plan_id: string;
  status: string;
  currency: string;
  current_period_start: string;
  current_period_end: string;
};

export type ListSubscriptionsResponse = PageInfo & {
  subscriptions: Subscription[];
};

export type Invoice = {
  id: string;
  org_id: string;
  customer_id: string;
  number: string;
  status: string;
  currency: string;
  total_cents: number;
  amount_due_cents: number;
  issued_at?: string;
  due_at?: string;
  period_start: string;
  period_end: string;
};

export type ListInvoicesResponse = PageInfo & {
  invoices: Invoice[];
};
