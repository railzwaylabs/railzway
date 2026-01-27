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

export interface ReadinessIssue {
  id: string;
  status: "ready" | "not_ready" | "optional";
  dependency_hint?: string;
  action_href: string;
  evidence?: Record<string, string>;
}

export interface ReadinessResponse {
  system_state: "ready" | "not_ready";
  issues: ReadinessIssue[];
}

export const getSystemReadiness = async (orgId: string) => {
  const { data } = await admin.get<ReadinessResponse>(`/organizations/${orgId}/readiness`);
  return data;
};

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
