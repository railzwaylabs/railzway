import { admin } from "@/api/client";

export interface BillingCycleSummary {
  cycle_id: string;
  period: string;
  total_revenue: number;
  invoice_count: number;
  status: string;
}

export interface BillingActivity {
  action: string;
  message: string;
  occurred_at: string;
}

export interface ActivityGroup {
  title: string;
  activities: BillingActivity[];
}

export const getBillingCycles = async () => {
  // TODO: Update URL when backend routing is confirmed
  const { data } = await admin.get<{ cycles: BillingCycleSummary[] }>("/billing/cycles");
  return data;
};

export const getBillingActivity = async (limit = 20) => {
  // TODO: Update URL when backend routing is confirmed
  const { data } = await admin.get<{ activity: ActivityGroup[] }>("/billing/activity", {
    params: { limit },
  });
  return data;
};
