import { create } from "zustand"
import { fetchSystemCapabilities, type SystemCapabilities, type SystemFeatures } from "../api/capabilities"

interface CapabilityState {
  capabilities: SystemCapabilities | null
  isLoading: boolean
  error: Error | null
  fetchCapabilities: () => Promise<void>
}

export const useCapabilityStore = create<CapabilityState>((set) => ({
  capabilities: null,
  isLoading: false,
  error: null,
  fetchCapabilities: async () => {
    set({ isLoading: true, error: null })
    try {
      const data = await fetchSystemCapabilities()
      set({ capabilities: data, isLoading: false })
    } catch (err: any) {
      set({ error: err, isLoading: false })
    }
  },
}))

// Selectors
export const useIsPlus = () => {
  const caps = useCapabilityStore((state) => state.capabilities)
  return caps?.plan === "plus"
}

export const useCapability = (feature: keyof SystemFeatures) => {
  const caps = useCapabilityStore((state) => state.capabilities)
  // Default to false if not loaded or not present
  return caps?.features?.[feature] ?? false
}

export const useSystemPlan = () => {
  const caps = useCapabilityStore((state) => state.capabilities)
  return caps?.plan ?? "oss"
}
