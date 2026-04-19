import { createContext, useContext, useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"

type InsightStatus = "idle" | "loading" | "ready" | "error"
type WindowState = "closed" | "open" | "minimized"

type AIAssistantContextValue = {
  enabled: boolean
  windowState: WindowState
  insightStatus: InsightStatus
  hasUnreadUpdate: boolean
  openAssistant: () => void
  closeAssistant: () => void
  minimizeAssistant: () => void
  setInsightStatus: (status: InsightStatus) => void
  markAssistantUnread: () => void
  markAssistantSeen: () => void
}

const AIAssistantContext = createContext<AIAssistantContextValue | null>(null)

export function AIAssistantProvider({
  children,
  enabled,
}: {
  children: ReactNode
  enabled: boolean
}) {

  const [windowState, setWindowState] = useState<WindowState>("closed")
  const [insightStatus, setInsightStatus] = useState<InsightStatus>("idle")
  const [hasUnreadUpdate, setHasUnreadUpdate] = useState(false)

  useEffect(() => {
    if (windowState === "open") {
      setHasUnreadUpdate(false)
    }
  }, [windowState])

  useEffect(() => {
    if (!enabled) {
      setWindowState("closed")
      setHasUnreadUpdate(false)
      setInsightStatus("idle")
    }
  }, [enabled])

  const value = useMemo<AIAssistantContextValue>(() => ({
    enabled,
    windowState,
    insightStatus,
    hasUnreadUpdate,
    openAssistant: () => setWindowState("open"),
    closeAssistant: () => setWindowState("closed"),
    minimizeAssistant: () => setWindowState("minimized"),
    setInsightStatus,
    markAssistantUnread: () => setHasUnreadUpdate(true),
    markAssistantSeen: () => setHasUnreadUpdate(false),
  }), [enabled, hasUnreadUpdate, insightStatus, windowState])

  return (
    <AIAssistantContext.Provider value={value}>
      {children}
    </AIAssistantContext.Provider>
  )
}

export function useAIAssistant() {
  const context = useContext(AIAssistantContext)
  if (!context) {
    throw new Error("useAIAssistant must be used within AIAssistantProvider")
  }
  return context
}

export type { InsightStatus, WindowState }
