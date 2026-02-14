export type IntegrationType = "notification" | "accounting" | "payment" | "crm" | "tax" | "data_warehouse" | "analytics";

export interface CatalogItem {
  id: string;
  type: IntegrationType;
  name: string;
  description: string;
  logo_url: string;
  auth_type: "oauth2" | "api_key" | "basic";
  schema: any; // JSON Schema
  is_active: boolean;
}

export interface Connection {
  id: string;
  org_id: string;
  integration_id: string;
  integration?: CatalogItem;
  name: string;
  config: Record<string, any>;
  status: "active" | "error" | "disconnected";
  error_message?: string;
  last_synced_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ConnectInput {
  integration_id: string;
  name: string;
  config: Record<string, any>;
  credentials: Record<string, any>;
}
