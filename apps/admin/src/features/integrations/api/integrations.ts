import { admin } from "@/api/client";
import type { CatalogItem, Connection, ConnectInput } from "../types";

export const listCatalog = async (): Promise<CatalogItem[]> => {
  const response = await admin.get("/integrations/catalog");
  return response.data;
};

export const listConnections = async (): Promise<Connection[]> => {
  const response = await admin.get("/integrations/connections");
  return response.data;
};

export const connectIntegration = async (input: ConnectInput): Promise<Connection> => {
  const response = await admin.post("/integrations/connect", input);
  return response.data;
};

export const disconnectIntegration = async (id: string): Promise<void> => {
  await admin.delete(`/integrations/connections/${id}`);
};

export const getConnectionConfig = async (id: string): Promise<Record<string, any>> => {
  const response = await admin.get(`/integrations/connections/${id}/config`);
  return response.data;
};
