import { useParams } from "react-router-dom"
import { getOrgId } from "./auth"

export function useOrgIdParam(): string {
  const params = useParams()
  const fromPath = typeof params.orgId === "string" ? params.orgId : ""
  return fromPath || getOrgId() || ""
}

export function useOrgPath() {
  const orgId = useOrgIdParam()
  return (path: string) => {
    if (!orgId) return path
    if (!path) return `/organizations/${orgId}`
    if (path.startsWith("/")) return `/organizations/${orgId}${path}`
    return `/organizations/${orgId}/${path}`
  }
}
