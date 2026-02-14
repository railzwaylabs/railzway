import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as api from "../api/integrations";
import type { ConnectInput } from "../types";

export function useIntegrationCatalog() {
  return useQuery({
    queryKey: ["integrations", "catalog"],
    queryFn: api.listCatalog,
  });
}

export function useIntegrationConnections() {
  return useQuery({
    queryKey: ["integrations", "connections"],
    queryFn: api.listConnections,
  });
}

export function useConnectIntegration() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: ConnectInput) => api.connectIntegration(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations", "connections"] });
    },
  });
}

export function useDisconnectIntegration() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.disconnectIntegration(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["integrations", "connections"] });
    },
  });
}

export function useConnectionConfig(id: string | null) {
  return useQuery({
    queryKey: ["integrations", "connections", id, "config"],
    queryFn: () => api.getConnectionConfig(id!),
    enabled: !!id,
  });
}
