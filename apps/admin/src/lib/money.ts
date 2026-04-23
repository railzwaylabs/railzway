export const DEFAULT_MONEY_INPUT = "0.00"
export const MONEY_INPUT_STEP = "0.01"
export const DEFAULT_USAGE_MONEY_INPUT = "0.00000000"
export const USAGE_MONEY_INPUT_STEP = "0.00000001"

const FLAT_DECIMALS = 2
const USAGE_DECIMALS = 8
const CENTS_SCALE = 12

export function moneyInputDecimalsForPriceType(priceType?: string): number {
  return priceType === "usage" || priceType === "tiered" ? USAGE_DECIMALS : FLAT_DECIMALS
}

export function defaultMoneyInputForPriceType(priceType?: string): string {
  return moneyInputDecimalsForPriceType(priceType) === USAGE_DECIMALS ? DEFAULT_USAGE_MONEY_INPUT : DEFAULT_MONEY_INPUT
}

export function moneyInputStepForPriceType(priceType?: string): string {
  return moneyInputDecimalsForPriceType(priceType) === USAGE_DECIMALS ? USAGE_MONEY_INPUT_STEP : MONEY_INPUT_STEP
}

function normalizeMoneyInput(value: unknown): string {
  const raw = String(value ?? "").trim()
  if (!raw) return ""
  if (!raw.includes(".") && raw.includes(",")) return raw.replace(",", ".")
  return raw.replace(/,/g, "")
}

function countDecimals(value: string): number {
  const [, decimals = ""] = value.split(".")
  return decimals.length
}

export function moneyInputToCents(value: unknown, decimals = FLAT_DECIMALS): number {
  if (value === "" || value == null) return 0
  const parsed = Number(normalizeMoneyInput(value))
  if (!Number.isFinite(parsed)) return 0
  const cents = parsed * 100
  return decimals <= FLAT_DECIMALS ? Math.round(cents) : Number(cents.toFixed(CENTS_SCALE))
}

export function optionalMoneyInputToCents(value: unknown, decimals = FLAT_DECIMALS): number | undefined {
  if (value === "" || value == null) return undefined
  return moneyInputToCents(value, decimals)
}

export function centsToMoneyInput(cents: number | null | undefined, decimals = FLAT_DECIMALS): string {
  if (!Number.isFinite(cents ?? NaN)) return DEFAULT_MONEY_INPUT
  return ((cents ?? 0) / 100).toFixed(decimals)
}

export function isNonNegativeMoneyInput(value: unknown, decimals = FLAT_DECIMALS): boolean {
  if (value === "" || value == null) return false
  const normalized = normalizeMoneyInput(value)
  const parsed = Number(normalized)
  return Number.isFinite(parsed) && parsed >= 0 && countDecimals(normalized) <= decimals
}
