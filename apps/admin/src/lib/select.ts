export const ALL_VALUE = "__all__";

export function toSelectValue(value: string | null | undefined): string {
  if (value == null || value === "") {
    return ALL_VALUE;
  }
  return value;
}

export function fromSelectValue(value: string): string {
  if (value === ALL_VALUE) {
    return "";
  }
  return value;
}
