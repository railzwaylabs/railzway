import { useCallback, useEffect, useRef, useState } from "react"

export type ToastVariant = "success" | "error" | "info" | "warning"

export interface ToastItem {
  id: string
  title: string
  description?: string
  variant: ToastVariant
}

type ToastListener = (toasts: ToastItem[]) => void

class ToastStore {
  private toasts: ToastItem[] = []
  private listeners: ToastListener[] = []

  subscribe(listener: ToastListener) {
    this.listeners.push(listener)
    return () => {
      this.listeners = this.listeners.filter((l) => l !== listener)
    }
  }

  private notify() {
    const snapshot = [...this.toasts]
    this.listeners.forEach((l) => l(snapshot))
  }

  add(item: Omit<ToastItem, "id">) {
    const id = Math.random().toString(36).slice(2)
    this.toasts = [...this.toasts, { ...item, id }]
    this.notify()
    return id
  }

  remove(id: string) {
    this.toasts = this.toasts.filter((t) => t.id !== id)
    this.notify()
  }
}

export const toastStore = new ToastStore()

export const toast = {
  success: (title: string, description?: string) =>
    toastStore.add({ title, description, variant: "success" }),
  error: (title: string, description?: string) =>
    toastStore.add({ title, description, variant: "error" }),
  info: (title: string, description?: string) =>
    toastStore.add({ title, description, variant: "info" }),
  warning: (title: string, description?: string) =>
    toastStore.add({ title, description, variant: "warning" }),
}

export function useToast() {
  const [toasts, setToasts] = useState<ToastItem[]>([])

  useEffect(() => {
    return toastStore.subscribe(setToasts)
  }, [])

  const dismiss = useCallback((id: string) => {
    toastStore.remove(id)
  }, [])

  return { toasts, dismiss }
}

// ── Icons ─────────────────────────────────────
function IconCheck() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
      <circle cx="8" cy="8" r="7.5" stroke="var(--status-success-text)" strokeWidth="1.5" fill="var(--status-success-bg)" />
      <path d="M5 8l2 2 4-4" stroke="var(--status-success-text)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function IconError() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
      <circle cx="8" cy="8" r="7.5" stroke="var(--status-danger-text)" strokeWidth="1.5" fill="var(--status-danger-bg)" />
      <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="var(--status-danger-text)" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

function IconInfo() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
      <circle cx="8" cy="8" r="7.5" stroke="var(--status-info-text)" strokeWidth="1.5" fill="var(--status-info-bg)" />
      <path d="M8 7v4M8 5.5v.5" stroke="var(--status-info-text)" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

function IconWarning() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
      <path d="M8 2L14 14H2L8 2z" stroke="var(--status-warning-text)" strokeWidth="1.5" fill="var(--status-warning-bg)" strokeLinejoin="round" />
      <path d="M8 7v3M8 11.5v.5" stroke="var(--status-warning-text)" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

function IconClose() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
      <path d="M3 3l8 8M11 3l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

function ToastIcon({ variant }: { variant: ToastVariant }) {
  if (variant === "success") return <IconCheck />
  if (variant === "error") return <IconError />
  if (variant === "warning") return <IconWarning />
  return <IconInfo />
}

// ── Single Toast ─────────────────────────────
function Toast({ item, onDismiss }: { item: ToastItem; onDismiss: (id: string) => void }) {
  const [dismissing, setDismissing] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout>>()

  const dismiss = useCallback(() => {
    setDismissing(true)
    timerRef.current = setTimeout(() => onDismiss(item.id), 240)
  }, [item.id, onDismiss])

  useEffect(() => {
    const t = setTimeout(dismiss, 5000)
    return () => {
      clearTimeout(t)
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [dismiss])

  return (
    <div className={`toast toast-${item.variant}${dismissing ? " toast-dismissing" : ""}`}>
      <div className="toast-icon">
        <ToastIcon variant={item.variant} />
      </div>
      <div className="toast-body">
        <div className="toast-title">{item.title}</div>
        {item.description ? <div className="toast-desc">{item.description}</div> : null}
      </div>
      <button className="toast-close" onClick={dismiss} aria-label="Dismiss">
        <IconClose />
      </button>
    </div>
  )
}

// ── Container ────────────────────────────────
export function ToastContainer() {
  const { toasts, dismiss } = useToast()

  return (
    <div className="toast-container">
      {toasts.map((t) => (
        <Toast key={t.id} item={t} onDismiss={dismiss} />
      ))}
    </div>
  )
}
