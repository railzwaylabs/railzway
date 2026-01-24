import { useEffect } from "react"
import { QueryClientProvider } from "@tanstack/react-query"
import { RouterProvider } from "react-router-dom"

import { router } from "@/router"
import { queryClient } from "@/lib/queryClient"
import { useCapabilityStore } from "@/stores/capabilityStore"

export default function App() {
  const fetchCapabilities = useCapabilityStore((state) => state.fetchCapabilities)

  useEffect(() => {
    void fetchCapabilities()
  }, [fetchCapabilities])

  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}
