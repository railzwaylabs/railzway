const ORG_KEY = "railzway_admin_org_id";

export function isAuthRequired(): boolean {
  return true;
}

export function getOrgId(): string | null {
  return window.localStorage.getItem(ORG_KEY);
}

export function setOrgId(orgId: string): void {
  window.localStorage.setItem(ORG_KEY, orgId);
}

export function clearOrgId(): void {
  window.localStorage.removeItem(ORG_KEY);
}
