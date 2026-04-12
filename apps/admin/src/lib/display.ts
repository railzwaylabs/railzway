/**
 * display.ts — Single source of truth for common display utilities.
 * Import from here instead of defining inline in every page.
 */

/** Human-readable hint for RFC3339 timestamp input fields */
export const rfc3339Hint =
  "Stored as RFC3339/UTC. Date pickers use your local time."

/**
 * Format an ISO timestamp string to a localized date string.
 * Returns "—" if the value is missing.
 */
export function formatDate(value?: string | null): string {
  if (!value) return "—"
  const d = new Date(value)
  return isNaN(d.getTime()) ? value : d.toLocaleDateString()
}

/**
 * Format an ISO timestamp string to a localized date+time string.
 * Returns "—" if the value is missing.
 */
export function formatDateTime(value?: string | null): string {
  if (!value) return "—"
  const d = new Date(value)
  return isNaN(d.getTime()) ? value : d.toLocaleString()
}

/**
 * Shorten a UUID-style ID to first 8 chars with ellipsis.
 * Returns "—" if absent.
 */
export function shortID(value?: string | null): string {
  if (!value) return "—"
  return `${value.slice(0, 8)}…`
}

/**
 * Normalize a date string to ISO 8601 / RFC3339.
 * Returns empty string for empty input; returns original string if unparseable.
 */
export function normalizeDate(value: string): string {
  if (!value) return ""
  const d = new Date(value)
  return isNaN(d.getTime()) ? value : d.toISOString()
}
