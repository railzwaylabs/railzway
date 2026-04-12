const ORG_KEY = "railzway_admin_org_id";
const TOKEN_KEY = "railzway_admin_token";

export function isAuthRequired(): boolean {
  return true;
}

export function getToken(): string | null {
  return window.sessionStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  window.sessionStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  window.sessionStorage.removeItem(TOKEN_KEY);
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
