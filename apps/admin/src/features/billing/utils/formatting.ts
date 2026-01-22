// Utility functions for Billing Operations IA

export function formatCurrency(amount: number, currency: string = "USD"): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: currency,
  }).format(amount / 100)
}

export function formatDateTime(dateStr?: string | null): string {
  if (!dateStr) return "N/A"

  const d = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffMinutes = Math.floor(diffMs / (1000 * 60))

  if (diffMinutes < 1) return "Just now"
  if (diffMinutes < 60) return `${diffMinutes}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays === 0) return "Today"
  if (diffDays === 1) return "Yesterday"
  if (diffDays < 30) return `${diffDays} days ago`

  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })
}

export function formatTimeRemaining(expiresAt?: string): { minutes: number; display: string } {
  if (!expiresAt) return { minutes: 0, display: "Expired" }

  const diff = new Date(expiresAt).getTime() - new Date().getTime()
  const minutes = Math.max(0, Math.ceil(diff / (1000 * 60)))

  if (minutes === 0) return { minutes: 0, display: "Expired" }
  if (minutes < 60) return { minutes, display: `${minutes}m` }

  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60

  if (remainingMinutes === 0) return { minutes, display: `${hours}h` }
  return { minutes, display: `${hours}h ${remainingMinutes}m` }
}

export function formatAssignmentAge(claimedAt: string): string {
  const ageMs = new Date().getTime() - new Date(claimedAt).getTime()
  const ageMinutes = Math.floor(ageMs / (1000 * 60))

  if (ageMinutes < 60) return `${ageMinutes}m`

  const ageHours = Math.floor(ageMinutes / 60)
  const remainingMinutes = ageMinutes % 60

  if (ageHours < 24) {
    return remainingMinutes === 0 ? `${ageHours}h` : `${ageHours}h ${remainingMinutes}m`
  }

  const ageDays = Math.floor(ageHours / 24)
  return `${ageDays}d`
}

export function getDaysToResolve(claimedAt: string, resolvedAt: string): number {
  const diffMs = new Date(resolvedAt).getTime() - new Date(claimedAt).getTime()
  return Math.max(0, Math.floor(diffMs / (1000 * 60 * 60 * 24)))
}
