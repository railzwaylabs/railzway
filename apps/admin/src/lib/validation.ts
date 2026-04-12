const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const currencyRegex = /^[A-Za-z]{3}$/;

export function isEmail(value: string): boolean {
  return emailRegex.test(value.trim());
}

export function isCurrencyCode(value: string): boolean {
  return currencyRegex.test(value.trim());
}
