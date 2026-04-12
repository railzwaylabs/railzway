const STATUS_CLASS_MAP: Record<string, string> = {
  active: "status-success",
  paid: "status-success",
  succeeded: "status-success",
  rated: "status-success",
  accepted: "status-success",
  enabled: "status-success",
  open: "status-warning",
  pending: "status-warning",
  trialing: "status-info",
  draft: "status-neutral",
  canceled: "status-muted",
  cancelled: "status-muted",
  inactive: "status-muted",
  disabled: "status-muted",
  void: "status-danger",
  failed: "status-danger",
  rejected: "status-danger",
  uncollectible: "status-danger",
  past_due: "status-danger",
  "past-due": "status-danger"
};

export function statusClass(status?: string | null): string {
  if (!status) {
    return "status-unknown";
  }
  const key = status.trim().toLowerCase();
  return STATUS_CLASS_MAP[key] ?? "status-unknown";
}
