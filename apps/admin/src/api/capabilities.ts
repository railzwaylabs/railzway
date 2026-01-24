import { admin } from "./client"

export type SystemFeatures = {
  sso: boolean
  rbac: boolean
  audit_export: boolean
  forecasting: boolean
  [key: string]: boolean
}

export type SystemCapabilities = {
  plan: "oss" | "plus"
  features: SystemFeatures
  expires_at: string
}

export const fetchSystemCapabilities = async (): Promise<SystemCapabilities> => {
  const { data } = await admin.get<SystemCapabilities>("/system/capabilities")
  return data
}
