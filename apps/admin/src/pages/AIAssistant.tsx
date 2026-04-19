import { startTransition, useDeferredValue, useEffect, useMemo, useRef, useState } from "react"
import type { KeyboardEvent } from "react"
import Mention from "@tiptap/extension-mention"
import StarterKit from "@tiptap/starter-kit"
import { EditorContent, NodeViewWrapper, ReactNodeViewRenderer, useEditor } from "@tiptap/react"
import type { Editor, NodeViewProps } from "@tiptap/react"
import { useNavigate } from "react-router-dom"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { api } from "../lib/api"
import { formatDateTime, shortID } from "../lib/display"
import { useOrgPath } from "../lib/org"
import type {
  AIPromptCreateRequest,
  AIPromptMessage,
  AIPromptMessageBlock,
  AIPromptResponse,
  AIPromptToken,
  AIPromptTokenResourceType,
  AIPromptTokenTimeType,
  AuditLog,
  Customer,
  Feature,
  Invoice,
  Meter,
  Product,
  Subscription,
  UsageEvent,
} from "../lib/types"
import { cn } from "../lib/utils"

type TokenOption = {
  id: string
  key: string
  kind: "resource" | "time"
  type: AIPromptTokenResourceType | AIPromptTokenTimeType
  label: string
  secondaryLabel?: string
  descriptor: string
  token: AIPromptToken
}

type ThreadRecord = {
  id: string
  title: string
  conversationId?: string
  messages: AIPromptMessage[]
  createdAt: string
  updatedAt: string
}

type PromptTokenMatch = {
  start: number
  end: number
  query: string
}

type ResourceFamily = {
  type: AIPromptTokenResourceType
  key: string
  label: string
  helper: string
}

type MentionSearchIntent = {
  family?: ResourceFamily
  query: string
}

type TimePickerState = {
  option: TokenOption
  match: PromptTokenMatch
}

const storageKey = "railzway:admin:ai-assistant:threads:v3"

const resourceFamilies: ResourceFamily[] = [
  { type: "customer", key: "@customer", label: "Customer", helper: "Find a customer by name or email." },
  { type: "invoice", key: "@invoice", label: "Invoice", helper: "Inspect a specific invoice by number, status, or customer." },
  { type: "subscription", key: "@subscription", label: "Subscription", helper: "Resolve a live subscription context." },
  { type: "product", key: "@product", label: "Product", helper: "Ground the prompt to one product." },
  { type: "meter", key: "@meter", label: "Meter", helper: "Inspect a specific usage meter." },
  { type: "usage", key: "@usage", label: "Usage", helper: "Reference a usage event or usage signal." },
  { type: "feature", key: "@feature", label: "Feature", helper: "Mention an entitlement or feature flag." },
  { type: "audit_log", key: "@audit_log", label: "Audit Log", helper: "Pull an operational change trail." },
]

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

function IconSpark() {
  return (
    <svg width="18" height="18" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M10 2l1.5 3.8L15.3 7.3 11.5 8.8 10 12.6 8.5 8.8 4.7 7.3l3.8-1.5L10 2z" strokeLinejoin="round" />
      <path d="M4 13l.8 2 .8.3-.8.3L4 18l-.8-2.4-.8-.3.8-.3L4 13z" strokeLinejoin="round" />
      <path d="M16 12l.7 1.6 1.6.7-1.6.7L16 16.6l-.7-1.6-1.6-.7 1.6-.7L16 12z" strokeLinejoin="round" />
    </svg>
  )
}

function IconSend() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 13.5L14 8 2 2.5l1.9 4.2L10 8l-6.1 1.3L2 13.5z" strokeLinejoin="round" />
    </svg>
  )
}

function IconCopy() {
  return (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="5" y="3" width="8" height="10" rx="2" />
      <path d="M3.5 10.5V5.5A2.5 2.5 0 0 1 6 3" strokeLinecap="round" />
    </svg>
  )
}

function IconThread() {
  return (
    <svg width="18" height="18" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2.5" y="3" width="15" height="11" rx="2.5" />
      <path d="M6 7h8M6 10h5" strokeLinecap="round" />
      <path d="M7.2 14v3l2.8-2.2h4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function IconBack() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6">
      <path d="M10.5 3.5L6 8l4.5 4.5" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M6 8h6.5" strokeLinecap="round" />
    </svg>
  )
}

function IconTrash() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2.5 4h11" strokeLinecap="round" />
      <path d="M6 2.5h4" strokeLinecap="round" />
      <path d="M5 5.5v6M8 5.5v6M11 5.5v6" strokeLinecap="round" />
      <path d="M4 4l.6 8.1a1 1 0 0 0 1 .9h4.8a1 1 0 0 0 1-.9L12 4" strokeLinejoin="round" />
    </svg>
  )
}

function parsePromptToken(value: string, caret: number): PromptTokenMatch | null {
  const before = value.slice(0, caret)
  const match = before.match(/(^|\s)@([a-z0-9_:-]*)$/i)
  if (!match) return null
  const query = match[2] ?? ""
  const start = before.lastIndexOf(`@${query}`)
  if (start < 0) return null
  return { start, end: caret, query }
}

function resolveMentionIntent(query: string): MentionSearchIntent {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return { query: "" }

  for (const family of resourceFamilies) {
    const familyKey = family.key.slice(1).toLowerCase()
    if (normalized === familyKey) {
      return { family, query: "" }
    }
    if (normalized.startsWith(`${familyKey}:`)) {
      return { family, query: normalized.slice(familyKey.length + 1).trim() }
    }
  }

  return { query: normalized }
}

function getExactTimeOption(query: string): TokenOption | null {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return null
  const option = buildTimeOptions("").find((item) => item.key.slice(1).toLowerCase() === normalized)
  return option ?? null
}

function getActiveOrgIdFromRoute() {
  if (typeof window === "undefined") return ""
  const match = window.location.pathname.match(/^\/organizations\/([^/]+)/)
  return match?.[1] ?? ""
}

function createEmptyThread(): ThreadRecord {
  const now = new Date().toISOString()
  return {
    id: `thread-${crypto.randomUUID()}`,
    title: "New conversation",
    messages: [],
    createdAt: now,
    updatedAt: now,
  }
}

function buildAssistantErrorBlocks(message: string): AIPromptMessageBlock[] {
  const normalized = message.trim() || "The AI endpoint did not return a valid response."
  const lower = normalized.toLowerCase()

  if (lower.includes("quota") || lower.includes("rate limit")) {
    return [
      { type: "heading", text: "AI temporarily unavailable" },
      { type: "quote", text: normalized },
      {
        type: "list",
        title: "What to do next",
        data: {
          items: [
            "Wait for the retry window, then send the prompt again.",
            "If this happens often, switch the configured model or upgrade the provider quota.",
            "Your prompt and mentions are still valid. This failure came from the provider, not the billing context.",
          ],
        },
      },
    ]
  }

  if (lower.includes("credential") || lower.includes("model") || lower.includes("not available")) {
    return [
      { type: "heading", text: "AI provider configuration issue" },
      { type: "quote", text: normalized },
      {
        type: "text",
        title: "Action",
        text: "Check the configured API key, provider access, and model name in the server environment before retrying.",
      },
    ]
  }

  if (lower.includes("time") || lower.includes("timeout")) {
    return [
      { type: "heading", text: "AI request timed out" },
      { type: "quote", text: normalized },
      {
        type: "text",
        title: "Action",
        text: "Retry the prompt. If it keeps timing out, narrow the request or reduce the amount of context.",
      },
    ]
  }

  return [
    { type: "heading", text: "AI request failed" },
    { type: "quote", text: normalized },
    {
      type: "text",
      title: "Action",
      text: "Retry in a moment. If the problem repeats, check the AI provider configuration on the server.",
    },
  ]
}

function countOccurrences(prompt: string, key: string) {
  const escaped = key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
  const matches = prompt.match(new RegExp(escaped, "g"))
  return matches?.length ?? 0
}

function getTokenPromptText(token: AIPromptToken) {
  return formatTokenLabel(token)
}

function resolveUsedTokens(prompt: string, tokenPool: AIPromptToken[]) {
  const grouped = new Map<string, AIPromptToken[]>()
  for (const token of tokenPool) {
    const tokenText = getTokenPromptText(token)
    const list = grouped.get(tokenText) ?? []
    list.push(token)
    grouped.set(tokenText, list)
  }
  const used = new Map<string, number>()
  const ordered: AIPromptToken[] = []
  for (const token of tokenPool) {
    const tokenText = getTokenPromptText(token)
    const occurrences = countOccurrences(prompt, tokenText)
    const current = used.get(tokenText) ?? 0
    if (current >= occurrences) continue
    ordered.push(token)
    used.set(tokenText, current + 1)
  }
  return ordered
}

function buildPromptSegments(prompt: string, tokens: AIPromptToken[]) {
  const segments: Array<{ type: "text" | "token"; value: string; token?: AIPromptToken }> = []
  const orderedTokens = resolveUsedTokens(prompt, tokens)
  let cursor = 0
  for (const token of orderedTokens) {
    const value = getTokenPromptText(token)
    const start = prompt.indexOf(value, cursor)
    if (start < 0) continue
    if (start > cursor) {
      segments.push({ type: "text", value: prompt.slice(cursor, start) })
    }
    segments.push({ type: "token", value, token })
    cursor = start + value.length
  }
  if (cursor < prompt.length) {
    segments.push({ type: "text", value: prompt.slice(cursor) })
  }
  if (segments.length === 0) {
    segments.push({ type: "text", value: prompt })
  }
  return segments
}

function buildUserMessage(prompt: string, tokens: AIPromptToken[]): AIPromptMessage {
  return {
    id: `local-user-${crypto.randomUUID()}`,
    role: "user",
    prompt,
    tokens,
    created_at: new Date().toISOString(),
  }
}

function resolvePersistedThreadID(thread: ThreadRecord) {
  if (thread.conversationId && uuidPattern.test(thread.conversationId)) return thread.conversationId
  if (uuidPattern.test(thread.id)) return thread.id
  return ""
}

function ensureAssistantMessage(resp: AIPromptResponse): AIPromptMessage {
  return {
    id: resp.message.id || `assistant-${crypto.randomUUID()}`,
    role: resp.message.role || "assistant",
    prompt: resp.message.prompt,
    tokens: resp.message.tokens,
    blocks: resp.message.blocks,
    created_at: resp.message.created_at || new Date().toISOString(),
  }
}

function summarizeThread(messages: AIPromptMessage[]) {
  const firstUser = messages.find((message) => message.role === "user" && message.prompt?.trim())
  if (!firstUser?.prompt) return "New conversation"
  const singleLine = firstUser.prompt.replace(/\s+/g, " ").trim()
  return singleLine.length > 44 ? `${singleLine.slice(0, 44)}…` : singleLine
}

function formatTokenLabel(token: AIPromptToken) {
  if (token.kind === "time") {
    if (token.type === "range") return token.label || `${token.from || ""} -> ${token.to || ""}`.trim()
    return token.label
  }
  return token.secondary_label ? `${token.label} · ${token.secondary_label}` : token.label
}

function readCardItems(data: unknown) {
  if (typeof data === "string") {
    try {
      return readCardItems(JSON.parse(data))
    } catch {
      return []
    }
  }
  if (!Array.isArray(data)) return []
  return data
    .map((item) => {
      if (!item || typeof item !== "object") return null
      const card = item as Record<string, unknown>
      return {
        label: typeof card.label === "string" ? card.label : "",
        value: typeof card.value === "string" ? card.value : "",
        tone: typeof card.tone === "string" ? card.tone : "neutral",
      }
    })
    .filter((item): item is { label: string; value: string; tone: string } => Boolean(item && item.label && item.value))
}

function readListItems(data: unknown) {
  if (typeof data === "string") {
    try {
      return readListItems(JSON.parse(data))
    } catch {
      return []
    }
  }
  if (!data || typeof data !== "object") return []
  const payload = data as Record<string, unknown>
  if (!Array.isArray(payload.items)) return []
  return payload.items.filter((item): item is string => typeof item === "string" && item.trim().length > 0)
}

function readChartItems(data: unknown) {
  if (typeof data === "string") {
    try {
      return readChartItems(JSON.parse(data))
    } catch {
      return { kind: "bar", items: [] as Array<{ label: string; value: number }> }
    }
  }
  if (!data || typeof data !== "object") return { kind: "bar", items: [] as Array<{ label: string; value: number }> }
  const payload = data as Record<string, unknown>
  const items = Array.isArray(payload.items)
    ? payload.items
      .map((item) => {
        if (!item || typeof item !== "object") return null
        const point = item as Record<string, unknown>
        return {
          label: typeof point.label === "string" ? point.label : "",
          value: typeof point.value === "number" ? point.value : Number(point.value ?? 0),
        }
      })
      .filter((item): item is { label: string; value: number } => Boolean(item && item.label))
    : []
  return {
    kind: typeof payload.kind === "string" ? payload.kind : "bar",
    items,
  }
}

function readTableData(data: unknown) {
  if (typeof data === "string") {
    try {
      return readTableData(JSON.parse(data))
    } catch {
      return { columns: [] as string[], rows: [] as string[][] }
    }
  }
  if (!data || typeof data !== "object") {
    return { columns: [] as string[], rows: [] as string[][] }
  }
  const payload = data as Record<string, unknown>
  const columns = Array.isArray(payload.columns)
    ? payload.columns.filter((item): item is string => typeof item === "string" && item.trim().length > 0)
    : []
  const rows = Array.isArray(payload.rows)
    ? payload.rows
      .map((row) => Array.isArray(row) ? row.map((cell) => String(cell ?? "")) : null)
      .filter((row): row is string[] => Boolean(row))
    : []
  return { columns, rows }
}

function readTimelineItems(data: unknown) {
  if (typeof data === "string") {
    try {
      return readTimelineItems(JSON.parse(data))
    } catch {
      return []
    }
  }
  if (!data || typeof data !== "object") return []
  const payload = data as Record<string, unknown>
  if (!Array.isArray(payload.items)) return []
  return payload.items
    .map((item) => {
      if (!item || typeof item !== "object") return null
      const entry = item as Record<string, unknown>
      return {
        timestamp: typeof entry.timestamp === "string" ? entry.timestamp : "",
        label: typeof entry.label === "string" ? entry.label : "",
        description: typeof entry.description === "string" ? entry.description : "",
        tone: typeof entry.tone === "string" ? entry.tone : "neutral",
      }
    })
    .filter((item): item is { timestamp: string; label: string; description: string; tone: string } => Boolean(item && item.label))
}

function readBadgeItems(data: unknown) {
  if (typeof data === "string") {
    try {
      return readBadgeItems(JSON.parse(data))
    } catch {
      return []
    }
  }
  if (!data || typeof data !== "object") return []
  const payload = data as Record<string, unknown>
  if (!Array.isArray(payload.items)) return []
  return payload.items
    .map((item) => {
      if (!item || typeof item !== "object") return null
      const entry = item as Record<string, unknown>
      return {
        label: typeof entry.label === "string" ? entry.label : "",
        tone: typeof entry.tone === "string" ? entry.tone : "neutral",
      }
    })
    .filter((item): item is { label: string; tone: string } => Boolean(item && item.label))
}

function toneSurfaceClass(tone?: string) {
  if (tone === "positive") return "border-[hsl(var(--status-success)/0.26)] bg-[hsl(var(--status-success)/0.06)] text-[hsl(var(--text-primary))]"
  if (tone === "negative" || tone === "error") return "border-[hsl(var(--status-error)/0.22)] bg-[hsl(var(--status-error)/0.06)] text-[hsl(var(--text-primary))]"
  if (tone === "warning") return "border-[hsl(var(--status-warning)/0.24)] bg-[hsl(var(--status-warning)/0.08)] text-[hsl(var(--text-primary))]"
  return "border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] text-[hsl(var(--text-primary))]"
}

function toneDotClass(tone?: string) {
  if (tone === "positive") return "border-[hsl(var(--status-success)/0.26)] bg-[hsl(var(--status-success)/0.22)]"
  if (tone === "negative" || tone === "error") return "border-[hsl(var(--status-error)/0.22)] bg-[hsl(var(--status-error)/0.2)]"
  if (tone === "warning") return "border-[hsl(var(--status-warning)/0.24)] bg-[hsl(var(--status-warning)/0.24)]"
  return "border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface-strong))]"
}

function tableToMarkdown(data: unknown) {
  const table = readTableData(data)
  if (table.columns.length === 0) return ""
  const header = `| ${table.columns.join(" | ")} |`
  const divider = `| ${table.columns.map(() => "---").join(" | ")} |`
  const rows = table.rows.map((row) => `| ${row.map((cell) => String(cell ?? "")).join(" | ")} |`)
  return [header, divider, ...rows].join("\n")
}

function timelineToMarkdown(data: unknown) {
  return readTimelineItems(data)
    .map((item) => {
      const prefix = item.timestamp ? `- **${item.timestamp}** — ${item.label}` : `- **${item.label}**`
      return item.description ? `${prefix}\n  ${item.description}` : prefix
    })
    .join("\n")
}

function badgeToMarkdown(data: unknown) {
  return readBadgeItems(data).map((item) => `- ${item.label} (${item.tone})`).join("\n")
}

function chartToMarkdown(data: unknown) {
  const chart = readChartItems(data)
  return chart.items.map((item) => `- ${item.label}: ${item.value}`).join("\n")
}

function cardsToMarkdown(data: unknown) {
  return readCardItems(data).map((card) => `- **${card.label}:** ${card.value}`).join("\n")
}

function listToMarkdown(data: unknown) {
  return readListItems(data).map((item) => `- ${item}`).join("\n")
}

function alertToMarkdown(block: AIPromptMessageBlock) {
  const tone = (block.tone || "info").toUpperCase()
  const text = normalizeBlockText(block.text)
  if (!text) return ""
  return `> [!${tone}]\n> ${text.replace(/\n/g, "\n> ")}`
}

function blockToMarkdown(block: AIPromptMessageBlock) {
  const parts: string[] = []
  if (shouldShowBlockTitle(block.title)) {
    parts.push(`### ${normalizeBlockText(block.title)}`)
  }

  if (block.type === "heading") {
    const text = normalizeBlockText(block.text || block.title)
    if (text) parts.push(`## ${text}`)
    return parts.join("\n\n")
  }

  if (block.type === "quote") {
    const text = normalizeBlockText(block.text)
    if (text) parts.push(`> ${text.replace(/\n/g, "\n> ")}`)
    return parts.join("\n\n")
  }

  if (block.type === "text") {
    const text = normalizeBlockText(block.text)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "markdown") {
    const text = normalizeBlockText(block.text)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "list") {
    const text = listToMarkdown(block.data)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "cards") {
    const text = cardsToMarkdown(block.data)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "chart") {
    const text = chartToMarkdown(block.data)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "alert") {
    const text = alertToMarkdown(block)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "table") {
    const text = tableToMarkdown(block.data)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "timeline") {
    const text = timelineToMarkdown(block.data)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "badge") {
    const text = badgeToMarkdown(block.data)
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  if (block.type === "steps") {
    const text = listToMarkdown(block.data)
      .split("\n")
      .map((item, index) => item ? `${index + 1}. ${item.replace(/^- /, "")}` : "")
      .filter(Boolean)
      .join("\n")
    if (text) parts.push(text)
    return parts.join("\n\n")
  }

  const fallback = block.data ?? block.text ?? null
  parts.push("```json\n" + JSON.stringify(fallback, null, 2) + "\n```")
  return parts.join("\n\n")
}

function blocksToMarkdown(blocks: AIPromptMessageBlock[]) {
  return blocks
    .map((block) => blockToMarkdown(block))
    .filter((value) => value.trim().length > 0)
    .join("\n\n")
}

async function copyText(text: string, successTitle: string) {
  const normalized = text.trim()
  if (!normalized) {
    toast.error("Nothing to copy.")
    return
  }

  try {
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(normalized)
    } else if (typeof document !== "undefined") {
      const textarea = document.createElement("textarea")
      textarea.value = normalized
      textarea.setAttribute("readonly", "")
      textarea.style.position = "absolute"
      textarea.style.left = "-9999px"
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand("copy")
      document.body.removeChild(textarea)
    } else {
      throw new Error("Clipboard is unavailable.")
    }
    toast.success(successTitle)
  } catch (error) {
    toast.error("Copy failed.", error instanceof Error ? error.message : "Clipboard is unavailable.")
  }
}

function normalizeBlockText(value?: string) {
  return (value || "").trim()
}

function normalizeBlockType(type?: string) {
  return normalizeBlockText(type).toLowerCase()
}

function shouldShowBlockTitle(title?: string) {
  const normalized = normalizeBlockText(title).toLowerCase()
  return normalized !== "" && normalized !== "answer" && normalized !== "response"
}

function isShortBlockText(value?: string, limit = 140) {
  const normalized = normalizeBlockText(value)
  return normalized.length > 0 && normalized.length <= limit
}

function shouldUseCompactAssistantLayout(blocks: AIPromptMessageBlock[]) {
  if (blocks.length === 0 || blocks.length > 2) return false
  if (blocks.length === 1) {
    const block = blocks[0]
    const type = normalizeBlockType(block.type)
    return (type === "heading" || type === "text" || type === "quote") && isShortBlockText(block.text, 180)
  }

  const [first, second] = blocks
  return normalizeBlockType(first.type) === "heading" &&
    normalizeBlockType(second.type) === "quote" &&
    isShortBlockText(first.text, 60) &&
    isShortBlockText(second.text, 180)
}

function CompactAssistantMessage({ blocks }: { blocks: AIPromptMessageBlock[] }) {
  const heading = normalizeBlockText(blocks.find((block) => normalizeBlockType(block.type) === "heading")?.text)
  const quote = normalizeBlockText(blocks.find((block) => normalizeBlockType(block.type) === "quote")?.text)
  const text = normalizeBlockText(blocks.find((block) => {
    const type = normalizeBlockType(block.type)
    return type === "text" || type === "markdown"
  })?.text)
  const alert = blocks.find((block) => normalizeBlockType(block.type) === "alert")
  const body = quote || text || heading || ""

  return (
    <div className="max-w-[720px] rounded-[24px] bg-[hsl(var(--bg-primary))] px-5 py-4 shadow-[var(--shadow-xs)]">
      {heading ? (
        <div className="text-[22px] font-semibold tracking-tight text-[hsl(var(--text-primary))]">
          {heading}
        </div>
      ) : null}
      {body && body !== heading ? (
        <div className={cn("mt-2 text-[16px] leading-8 text-[hsl(var(--text-secondary))]", !heading && "text-[hsl(var(--text-primary))]")}>
          {body}
        </div>
      ) : null}
      {alert?.text ? (
        <div className="mt-3 text-[14px] leading-7 text-[hsl(var(--text-muted))]">
          {alert.text}
        </div>
      ) : null}
    </div>
  )
}

function renderBlock(block: AIPromptMessageBlock) {
  const type = normalizeBlockType(block.type)

  if (type === "heading") {
    return <div className="text-[28px] font-semibold tracking-tight text-[hsl(var(--text-primary))]">{block.text || block.title || ""}</div>
  }

  if (type === "quote") {
    return (
      <blockquote className="border-l-2 border-[hsl(var(--border-strong))] pl-4 text-[17px] font-medium leading-8 text-[hsl(var(--text-primary))]">
        {block.text || ""}
      </blockquote>
    )
  }

  if (type === "cards") {
    const cards = readCardItems(block.data)
    return (
      <div className="space-y-3">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {cards.map((card, index) => (
            <div
              key={`${card.label}-${index}`}
              className={cn(
                "rounded-[20px] border bg-[hsl(var(--bg-primary))] px-4 py-4",
                card.tone === "positive" && "border-[hsl(var(--status-success)/0.26)] bg-[hsl(var(--status-success)/0.06)]",
                card.tone === "negative" && "border-[hsl(var(--status-error)/0.22)] bg-[hsl(var(--status-error)/0.06)]",
                card.tone === "warning" && "border-[hsl(var(--status-warning)/0.24)] bg-[hsl(var(--status-warning)/0.08)]",
                card.tone === "neutral" && "border-[hsl(var(--border-subtle))]",
              )}
            >
              <div className="text-xs font-semibold uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">{card.label}</div>
              <div className="mt-2 text-lg font-semibold text-[hsl(var(--text-primary))]">{card.value}</div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (type === "list") {
    const items = readListItems(block.data)
    return (
      <div className="space-y-3">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="space-y-2">
          {items.map((item, index) => (
            <div key={`${item}-${index}`} className="flex items-start gap-3 rounded-[18px] bg-[hsl(var(--bg-primary))] px-4 py-3 text-[15px] leading-7 text-[hsl(var(--text-secondary))]">
              <span className="mt-2 h-2 w-2 rounded-full bg-[hsl(var(--text-primary))]" />
              <span>{item}</span>
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (type === "chart") {
    const chart = readChartItems(block.data)
    const maxValue = Math.max(...chart.items.map((item) => item.value), 1)
    return (
      <div className="space-y-3">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="rounded-[22px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] px-4 py-4">
          <div className="space-y-3">
            {chart.items.map((item) => (
              <div key={item.label} className="space-y-1">
                <div className="flex items-center justify-between text-sm text-[hsl(var(--text-secondary))]">
                  <span>{item.label}</span>
                  <span className="font-medium text-[hsl(var(--text-primary))]">{item.value}</span>
                </div>
                <div className="h-2 rounded-full bg-[hsl(var(--bg-surface-strong))]">
                  <div className="h-2 rounded-full bg-[hsl(var(--text-primary))]" style={{ width: `${Math.max((item.value / maxValue) * 100, 8)}%` }} />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (type === "text" || type === "markdown") {
    return (
      <div className="space-y-2">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="whitespace-pre-wrap text-[16px] leading-8 text-[hsl(var(--text-secondary))]">{block.text || ""}</div>
      </div>
    )
  }

  if (type === "alert") {
    const toneStyles =
      block.tone === "error"
        ? "border-[hsl(var(--status-error)/0.22)] bg-[hsl(var(--status-error)/0.08)] text-[hsl(var(--text-primary))]"
        : block.tone === "warning"
          ? "border-[hsl(var(--status-warning)/0.26)] bg-[hsl(var(--status-warning)/0.1)] text-[hsl(var(--text-primary))]"
          : "border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] text-[hsl(var(--text-primary))]"
    return (
      <div className={cn("rounded-[20px] border px-4 py-3 text-[15px] leading-7", toneStyles)}>
        {block.text || ""}
      </div>
    )
  }

  if (type === "table") {
    const table = readTableData(block.data)
    return (
      <div className="space-y-3">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="overflow-x-auto rounded-[22px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))]">
          <table className="min-w-full border-collapse text-left text-[14px] leading-6 text-[hsl(var(--text-secondary))]">
            <thead>
              <tr className="border-b border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))]">
                {table.columns.map((column) => (
                  <th key={column} className="px-4 py-3 font-semibold text-[hsl(var(--text-primary))]">{column}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {table.rows.map((row, index) => (
                <tr key={`${row.join("-")}-${index}`} className="border-b border-[hsl(var(--border-subtle))] last:border-b-0">
                  {row.map((cell, cellIndex) => (
                    <td key={`${cell}-${cellIndex}`} className="px-4 py-3 align-top">{cell}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    )
  }

  if (type === "timeline") {
    const items = readTimelineItems(block.data)
    return (
      <div className="space-y-3">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="space-y-3">
          {items.map((item, index) => (
            <div key={`${item.timestamp}-${item.label}-${index}`} className="flex gap-3">
              <div className="flex w-24 shrink-0 flex-col items-end pt-1 text-right text-xs text-[hsl(var(--text-muted))]">
                <span>{item.timestamp ? formatDateTime(item.timestamp) : "Event"}</span>
              </div>
                <div className="flex flex-1 gap-3">
                <div className="flex flex-col items-center">
                  <span className={cn("mt-1 inline-flex h-3 w-3 rounded-full border", toneDotClass(item.tone))} />
                  {index < items.length - 1 ? <span className="mt-1 h-full w-px bg-[hsl(var(--border-subtle))]" /> : null}
                </div>
                <div className={cn("flex-1 rounded-[18px] border px-4 py-3", toneSurfaceClass(item.tone))}>
                  <div className="text-[15px] font-medium">{item.label}</div>
                  {item.description ? <div className="mt-1 text-sm leading-6 text-[hsl(var(--text-secondary))]">{item.description}</div> : null}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (type === "badge") {
    const items = readBadgeItems(block.data)
    return (
      <div className="space-y-3">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="flex flex-wrap gap-2">
          {items.map((item, index) => (
            <span
              key={`${item.label}-${index}`}
              className={cn("inline-flex items-center rounded-full border px-3 py-1.5 text-sm font-medium", toneSurfaceClass(item.tone))}
            >
              {item.label}
            </span>
          ))}
        </div>
      </div>
    )
  }

  if (type === "steps") {
    const items = readListItems(block.data)
    return (
      <div className="space-y-3">
        {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
        <div className="space-y-3">
          {items.map((item, index) => (
            <div key={`${item}-${index}`} className="flex items-start gap-3 rounded-[18px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] px-4 py-3">
              <span className="inline-flex h-7 min-w-[28px] items-center justify-center rounded-full bg-[hsl(var(--bg-surface-strong))] text-sm font-semibold text-[hsl(var(--text-primary))]">
                {index + 1}
              </span>
              <span className="pt-0.5 text-[15px] leading-7 text-[hsl(var(--text-secondary))]">{item}</span>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {shouldShowBlockTitle(block.title) ? <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">{block.title}</div> : null}
      <pre className="overflow-x-auto rounded-2xl border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] p-4 text-xs leading-6 text-[hsl(var(--text-secondary))]">
        {JSON.stringify(block.data ?? block.text ?? null, null, 2)}
      </pre>
    </div>
  )
}

async function searchCustomers(query: string) {
  const trimmed = query.trim()
  const orgId = getActiveOrgIdFromRoute()
  const resp = await api.customers.list(
    trimmed ? (trimmed.includes("@") ? { page_size: 4, email: trimmed } : { page_size: 4, name: trimmed }) : { page_size: 4 },
    orgId ? { orgId } : undefined,
  )
  return resp.customers
}

async function searchInvoices(query: string) {
  const trimmed = query.trim().toLowerCase()
  const orgId = getActiveOrgIdFromRoute()
  const config = orgId ? { orgId } : undefined
  const primary = await api.invoices.list(trimmed ? { page_size: 8, number: trimmed } : { page_size: 8 }, config)
  if (!trimmed) return primary.invoices.slice(0, 4)
  if (primary.invoices.length > 0) return primary.invoices.slice(0, 4)

  const fallback = await api.invoices.list({ page_size: 8 }, config)
  return fallback.invoices.filter((invoice) =>
    invoice.number.toLowerCase().includes(trimmed) ||
    invoice.status.toLowerCase().includes(trimmed) ||
    invoice.customer_id.toLowerCase().includes(trimmed) ||
    (invoice.subscription_id || "").toLowerCase().includes(trimmed),
  ).slice(0, 4)
}

async function searchProducts(query: string) {
  const trimmed = query.trim()
  const orgId = getActiveOrgIdFromRoute()
  const config = orgId ? { orgId } : undefined
  const primary = await api.products.list(trimmed ? { page_size: 4, name: trimmed } : { page_size: 4 }, config)
  if (primary.products.length > 0 || !trimmed) return primary.products
  const fallback = await api.products.list({ page_size: 4, code: trimmed }, config)
  return fallback.products
}

async function searchMeters(query: string) {
  const trimmed = query.trim()
  const orgId = getActiveOrgIdFromRoute()
  const config = orgId ? { orgId } : undefined
  const primary = await api.meters.list(trimmed ? { page_size: 4, name: trimmed } : { page_size: 4 }, config)
  if (primary.meters.length > 0 || !trimmed) return primary.meters
  const fallback = await api.meters.list({ page_size: 4, code: trimmed }, config)
  return fallback.meters
}

async function searchFeatures(query: string) {
  const trimmed = query.trim()
  const orgId = getActiveOrgIdFromRoute()
  const config = orgId ? { orgId } : undefined
  const primary = await api.features.list(trimmed ? { page_size: 4, name: trimmed } : { page_size: 4 }, config)
  if (primary.features.length > 0 || !trimmed) return primary.features
  const fallback = await api.features.list({ page_size: 4, code: trimmed }, config)
  return fallback.features
}

async function searchSubscriptions(query: string) {
  const trimmed = query.trim().toLowerCase()
  const orgId = getActiveOrgIdFromRoute()
  const resp = await api.subscriptions.list({ page_size: 8 }, orgId ? { orgId } : undefined)
  if (!trimmed) return resp.subscriptions.slice(0, 4)
  return resp.subscriptions.filter((subscription) =>
    subscription.id.toLowerCase().includes(trimmed) ||
    subscription.status.toLowerCase().includes(trimmed) ||
    subscription.customer_id.toLowerCase().includes(trimmed),
  ).slice(0, 4)
}

async function searchUsageEvents(query: string) {
  const trimmed = query.trim().toLowerCase()
  const orgId = getActiveOrgIdFromRoute()
  const resp = await api.usage.listEvents({ page_size: 8 }, orgId ? { orgId } : undefined)
  if (!trimmed) return resp.events.slice(0, 4)
  return resp.events.filter((event) =>
    event.id.toLowerCase().includes(trimmed) ||
    event.customer_id.toLowerCase().includes(trimmed) ||
    event.meter_code.toLowerCase().includes(trimmed),
  ).slice(0, 4)
}

async function searchAuditLogs(query: string) {
  const trimmed = query.trim().toLowerCase()
  const orgId = getActiveOrgIdFromRoute()
  const resp = await api.auditLogs.list({ page_size: 8 }, orgId ? { orgId } : undefined)
  if (!trimmed) return resp.logs.slice(0, 4)
  return resp.logs.filter((log) =>
    log.id.toLowerCase().includes(trimmed) ||
    log.action.toLowerCase().includes(trimmed) ||
    (log.resource_id || "").toLowerCase().includes(trimmed) ||
    log.resource_type.toLowerCase().includes(trimmed),
  ).slice(0, 4)
}

function buildTimeOptions(query: string) {
  const all = [
    {
      id: "time-date",
      key: "@date",
      kind: "time" as const,
      type: "date" as const,
      label: "Specific date",
      secondaryLabel: "Pick a calendar date",
      descriptor: "Time token",
      token: {
        id: `time-date-${crypto.randomUUID()}`,
        kind: "time" as const,
        type: "date" as const,
        key: "@date",
        label: "Specific date",
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
      },
    },
    {
      id: "time-month",
      key: "@month",
      kind: "time" as const,
      type: "month" as const,
      label: "Billing month",
      secondaryLabel: "Pick a billing month",
      descriptor: "Time token",
      token: {
        id: `time-month-${crypto.randomUUID()}`,
        kind: "time" as const,
        type: "month" as const,
        key: "@month",
        label: "Billing month",
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
      },
    },
    {
      id: "time-range",
      key: "@range",
      kind: "time" as const,
      type: "range" as const,
      label: "Date range",
      secondaryLabel: "Pick a start and end date",
      descriptor: "Time token",
      token: {
        id: `time-range-${crypto.randomUUID()}`,
        kind: "time" as const,
        type: "range" as const,
        key: "@range",
        label: "Date range",
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
      },
    },
    {
      id: "time-last-30-days",
      key: "@last_30_days",
      kind: "time" as const,
      type: "relative" as const,
      label: "Last 30 days",
      secondaryLabel: "Relative window",
      descriptor: "Time token",
      token: {
        id: `time-relative-${crypto.randomUUID()}`,
        kind: "time" as const,
        type: "relative" as const,
        key: "@last_30_days",
        label: "Last 30 days",
        preset: "last_30_days",
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
      },
    },
  ]
  if (!query.trim()) return all
  const lowered = query.trim().toLowerCase()
  return all.filter((option) =>
    option.key.toLowerCase().includes(lowered) ||
    option.label.toLowerCase().includes(lowered) ||
    (option.secondaryLabel || "").toLowerCase().includes(lowered),
  )
}

function mapCustomerOption(customer: Customer): TokenOption {
  return {
    id: `customer-${customer.id}`,
    key: "@customer",
    kind: "resource",
    type: "customer",
    label: customer.name,
    secondaryLabel: customer.email,
    descriptor: "Customer",
    token: {
      id: `customer-${crypto.randomUUID()}`,
      kind: "resource",
      type: "customer",
      key: "@customer",
      label: customer.name,
      secondary_label: customer.email,
      resource_id: customer.id,
      metadata: { email: customer.email },
    },
  }
}

function mapInvoiceOption(invoice: Invoice): TokenOption {
  return {
    id: `invoice-${invoice.id}`,
    key: "@invoice",
    kind: "resource",
    type: "invoice",
    label: invoice.number,
    secondaryLabel: `${invoice.status} · ${invoice.currency} ${invoice.total_cents / 100}`,
    descriptor: "Invoice",
    token: {
      id: `invoice-${crypto.randomUUID()}`,
      kind: "resource",
      type: "invoice",
      key: "@invoice",
      label: invoice.number,
      secondary_label: invoice.status,
      resource_id: invoice.id,
      metadata: {
        customer_id: invoice.customer_id,
        subscription_id: invoice.subscription_id,
        status: invoice.status,
        total_cents: invoice.total_cents,
        currency: invoice.currency,
      },
    },
  }
}

function mapProductOption(product: Product): TokenOption {
  return {
    id: `product-${product.id}`,
    key: "@product",
    kind: "resource",
    type: "product",
    label: product.name,
    secondaryLabel: product.code,
    descriptor: "Product",
    token: {
      id: `product-${crypto.randomUUID()}`,
      kind: "resource",
      type: "product",
      key: "@product",
      label: product.name,
      secondary_label: product.code,
      resource_id: product.id,
    },
  }
}

function mapSubscriptionOption(subscription: Subscription): TokenOption {
  return {
    id: `subscription-${subscription.id}`,
    key: "@subscription",
    kind: "resource",
    type: "subscription",
    label: subscription.status,
    secondaryLabel: shortID(subscription.id),
    descriptor: "Subscription",
    token: {
      id: `subscription-${crypto.randomUUID()}`,
      kind: "resource",
      type: "subscription",
      key: "@subscription",
      label: subscription.status,
      secondary_label: subscription.id,
      resource_id: subscription.id,
      metadata: { customer_id: subscription.customer_id, plan_id: subscription.plan_id },
    },
  }
}

function mapMeterOption(meter: Meter): TokenOption {
  return {
    id: `meter-${meter.id}`,
    key: "@meter",
    kind: "resource",
    type: "meter",
    label: meter.name,
    secondaryLabel: meter.code,
    descriptor: "Meter",
    token: {
      id: `meter-${crypto.randomUUID()}`,
      kind: "resource",
      type: "meter",
      key: "@meter",
      label: meter.name,
      secondary_label: meter.code,
      resource_id: meter.id,
    },
  }
}

function mapUsageOption(event: UsageEvent): TokenOption {
  return {
    id: `usage-${event.id}`,
    key: "@usage",
    kind: "resource",
    type: "usage",
    label: event.meter_code,
    secondaryLabel: `${event.value} · ${shortID(event.id)}`,
    descriptor: "Usage event",
    token: {
      id: `usage-${crypto.randomUUID()}`,
      kind: "resource",
      type: "usage",
      key: "@usage",
      label: event.meter_code,
      secondary_label: event.id,
      resource_id: event.id,
      metadata: { customer_id: event.customer_id, value: event.value, recorded_at: event.recorded_at },
    },
  }
}

function mapFeatureOption(feature: Feature): TokenOption {
  return {
    id: `feature-${feature.id}`,
    key: "@feature",
    kind: "resource",
    type: "feature",
    label: feature.name,
    secondaryLabel: feature.code,
    descriptor: "Feature",
    token: {
      id: `feature-${crypto.randomUUID()}`,
      kind: "resource",
      type: "feature",
      key: "@feature",
      label: feature.name,
      secondary_label: feature.code,
      resource_id: feature.id,
    },
  }
}

function mapAuditOption(log: AuditLog): TokenOption {
  return {
    id: `audit-${log.id}`,
    key: "@audit_log",
    kind: "resource",
    type: "audit_log",
    label: log.action,
    secondaryLabel: `${log.resource_type} · ${shortID(log.id)}`,
    descriptor: "Audit log",
    token: {
      id: `audit-${crypto.randomUUID()}`,
      kind: "resource",
      type: "audit_log",
      key: "@audit_log",
      label: log.action,
      secondary_label: log.resource_type,
      resource_id: log.id,
      metadata: { resource_id: log.resource_id, resource_type: log.resource_type, created_at: log.created_at },
    },
  }
}

function getMentionBadge(option: TokenOption) {
  if (option.kind === "time") return "T"
  return option.key.replace("@", "").slice(0, 2).toUpperCase()
}

function getFamilyBadge(family: ResourceFamily) {
  return family.key.replace("@", "").slice(0, 2).toUpperCase()
}

function getSuggestionEmptyText(mentionIntent: MentionSearchIntent, loading: boolean) {
  if (loading) {
    if (mentionIntent.family) {
      return `Loading ${mentionIntent.family.label.toLowerCase()} records...`
    }
    return "Loading token suggestions..."
  }
  if (mentionIntent.family) {
    return `No ${mentionIntent.family.label.toLowerCase()} records found in this organization yet.`
  }
  return "No token records matched yet."
}

function buildTimeTokenFromValues(option: TokenOption, values: { value?: string; from?: string; to?: string }): AIPromptToken {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"
  if (option.type === "date") {
    return {
      ...option.token,
      label: values.value || "Specific date",
      value: values.value,
      timezone,
    } as AIPromptToken
  }
  if (option.type === "month") {
    return {
      ...option.token,
      label: values.value || "Billing month",
      value: values.value,
      timezone,
    } as AIPromptToken
  }
  if (option.type === "range") {
    const label = values.from && values.to ? `${values.from} -> ${values.to}` : "Date range"
    return {
      ...option.token,
      label,
      from: values.from,
      to: values.to,
      timezone,
    } as AIPromptToken
  }
  return option.token
}

type MentionAttrs = {
  token: AIPromptToken
  label: string
}

function MentionPillView(props: NodeViewProps) {
  const attrs = props.node.attrs as MentionAttrs
  return (
    <NodeViewWrapper
      as="span"
      data-token-text={attrs.label}
      className="mx-[1px] inline-flex translate-y-[-1px] items-center gap-2 rounded-full border border-[hsl(var(--border-strong))] bg-[hsl(var(--bg-primary))] px-4 py-1.5 text-[14px] font-medium text-[hsl(var(--text-primary))] shadow-[var(--shadow-xs)]"
    >
      <span>{attrs.label}</span>
      <button
        type="button"
        contentEditable={false}
        className="inline-flex h-5 w-5 items-center justify-center rounded-full text-[hsl(var(--text-muted))] transition hover:bg-[hsl(var(--bg-surface))] hover:text-[hsl(var(--text-primary))]"
        onMouseDown={(event) => {
          event.preventDefault()
          props.deleteNode()
          props.editor.commands.focus()
        }}
        aria-label={`Remove ${attrs.label}`}
      >
        ×
      </button>
    </NodeViewWrapper>
  )
}

const AssistantMention = Mention.extend({
  name: "assistantMention",
  addAttributes() {
    return {
      token: {
        default: null,
      },
      label: {
        default: "",
      },
    }
  },
  renderText({ node }) {
    const attrs = node.attrs as MentionAttrs
    return attrs.label || ""
  },
  renderHTML({ node, HTMLAttributes }) {
    const attrs = node.attrs as MentionAttrs
    return ["span", HTMLAttributes, attrs.label || ""]
  },
  addNodeView() {
    return ReactNodeViewRenderer(MentionPillView)
  },
})

function createPromptDoc(prompt: string) {
  return {
    type: "doc",
    content: [
      {
        type: "paragraph",
        content: prompt ? [{ type: "text", text: prompt }] : [],
      },
    ],
  }
}

function serializeEditor(editor: Editor | null) {
  if (!editor) return { prompt: "", tokens: [] as AIPromptToken[] }

  let promptText = ""
  const tokens: AIPromptToken[] = []

  editor.state.doc.descendants((node) => {
    if (node.type.name === "assistantMention") {
      const attrs = node.attrs as MentionAttrs
      if (attrs.token) {
        tokens.push(attrs.token)
        promptText += attrs.label || formatTokenLabel(attrs.token)
      }
      return false
    }
    if (node.type.name === "hardBreak") {
      promptText += "\n"
      return false
    }
    if (node.isText) {
      promptText += node.text || ""
    }
    return true
  })

  return { prompt: promptText, tokens }
}

function getEditorTokenMatch(editor: Editor | null): PromptTokenMatch | null {
  if (!editor) return null
  const { from, $from } = editor.state.selection
  const textBefore = $from.parent.textBetween(0, $from.parentOffset, undefined, "\ufffc")
  const match = textBefore.match(/(^|\s)@([a-z0-9_:-]*)$/i)
  if (!match) return null
  const query = match[2] ?? ""
  return {
    start: from - query.length - 1,
    end: from,
    query,
  }
}

function TimeTokenPicker({
  option,
  onCancel,
  onApply,
}: {
  option: TokenOption
  onCancel: () => void
  onApply: (values: { value?: string; from?: string; to?: string }) => void
}) {
  const today = new Date().toISOString().slice(0, 10)
  const [value, setValue] = useState(option.type === "month" ? today.slice(0, 7) : today)
  const [from, setFrom] = useState(() => {
    const base = new Date()
    base.setDate(base.getDate() - 29)
    return base.toISOString().slice(0, 10)
  })
  const [to, setTo] = useState(today)

  const apply = () => {
    if (option.type === "range") {
      if (!from || !to) return
      onApply({ from, to })
      return
    }
    if (!value) return
    onApply({ value })
  }

  return (
    <div className="absolute left-3 right-3 bottom-[calc(100%+12px)] z-40 rounded-[24px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] p-4 shadow-[var(--shadow-lg)]">
      <div className="mb-4 text-xs font-semibold uppercase tracking-[0.22em] text-[hsl(var(--text-muted))]">{option.label}</div>
      {option.type === "date" ? (
        <Input type="date" value={value} onChange={(event) => setValue(event.target.value)} />
      ) : null}
      {option.type === "month" ? (
        <Input type="month" value={value} onChange={(event) => setValue(event.target.value)} />
      ) : null}
      {option.type === "range" ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-2">
            <div className="text-xs font-medium uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">Start date</div>
            <Input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
          </div>
          <div className="space-y-2">
            <div className="text-xs font-medium uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">End date</div>
            <Input type="date" value={to} min={from} onChange={(event) => setTo(event.target.value)} />
          </div>
        </div>
      ) : null}
      <div className="mt-4 flex items-center justify-end gap-2">
        <Button type="button" variant="secondary" onClick={onCancel}>Cancel</Button>
        <Button type="button" variant="default" onClick={apply}>Apply</Button>
      </div>
    </div>
  )
}

function AssistantMessage({ message }: { message: AIPromptMessage }) {
  const blocks = message.blocks?.length ? message.blocks : [{ type: "text", text: message.prompt || "" } as AIPromptMessageBlock]
  return (
    <div className="flex justify-start">
      <div className="max-w-[820px] px-2 py-1">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.22em] text-[hsl(var(--text-muted))]">
            <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] text-[hsl(var(--text-primary))]">
              <IconSpark />
            </span>
            AI Assistant
          </div>
          <button
            type="button"
            onClick={() => void copyText(blocksToMarkdown(blocks), "Response copied as Markdown.")}
            className="inline-flex items-center gap-1.5 rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] px-3 py-1.5 text-xs font-medium text-[hsl(var(--text-secondary))] transition hover:border-[hsl(var(--border-strong))] hover:text-[hsl(var(--text-primary))]"
          >
            <IconCopy />
            Copy markdown
          </button>
        </div>
        {shouldUseCompactAssistantLayout(blocks) ? (
          <CompactAssistantMessage blocks={blocks} />
        ) : (
          <div className="space-y-6">
            {blocks.map((block, index) => (
              <div key={`${message.id}-block-${index}`}>{renderBlock(block)}</div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function UserMessage({ message }: { message: AIPromptMessage }) {
  const segments = buildPromptSegments(message.prompt || "", message.tokens || [])
  return (
    <div className="flex justify-end">
      <div className="max-w-[520px] rounded-[24px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] px-5 py-4 text-[hsl(var(--text-primary))] shadow-[var(--shadow-sm)]">
        <div className="mb-3 flex items-center justify-end">
          <button
            type="button"
            onClick={() => void copyText(message.prompt || "", "Prompt copied.")}
            className="inline-flex items-center gap-1.5 rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] px-3 py-1 text-xs font-medium text-[hsl(var(--text-secondary))] transition hover:border-[hsl(var(--border-strong))] hover:text-[hsl(var(--text-primary))]"
          >
            <IconCopy />
            Copy
          </button>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-[15px] leading-7">
          {segments.map((segment, index) =>
            segment.type === "token" && segment.token ? (
              <span key={`${message.id}-segment-${index}`} className="inline-flex items-center rounded-full border border-[hsl(var(--border-strong))] bg-[hsl(var(--bg-surface))] px-3 py-1 text-sm text-[hsl(var(--text-primary))]">
                {formatTokenLabel(segment.token)}
              </span>
            ) : (
              <span key={`${message.id}-segment-${index}`} className="whitespace-pre-wrap">
                {segment.value}
              </span>
            ),
          )}
        </div>
        <div className="mt-3 text-right text-xs text-[hsl(var(--text-muted))]">{formatDateTime(message.created_at)}</div>
      </div>
    </div>
  )
}

function EmptyPrompt({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] px-4 py-2 text-sm text-[hsl(var(--text-secondary))] transition hover:border-[hsl(var(--border-strong))] hover:bg-[hsl(var(--bg-surface))] hover:text-[hsl(var(--text-primary))]"
    >
      {label}
    </button>
  )
}

export function AIAssistantWorkspace() {
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [threads, setThreads] = useState<ThreadRecord[]>(() => [createEmptyThread()])
  const [activeThreadId, setActiveThreadId] = useState<string>(() => threads[0]?.id || "")
  const [prompt, setPrompt] = useState("")
  const [tokenPool, setTokenPool] = useState<AIPromptToken[]>([])
  const [suggestions, setSuggestions] = useState<TokenOption[]>([])
  const [suggestionsOpen, setSuggestionsOpen] = useState(false)
  const [suggestionsLoading, setSuggestionsLoading] = useState(false)
  const [suggestionIndex, setSuggestionIndex] = useState(0)
  const [tokenMatch, setTokenMatch] = useState<PromptTokenMatch | null>(null)
  const [timePickerState, setTimePickerState] = useState<TimePickerState | null>(null)
  const [familySearchQuery, setFamilySearchQuery] = useState("")
  const [resourceFilter, setResourceFilter] = useState<AIPromptTokenResourceType | "all">("all")
  const [submitting, setSubmitting] = useState(false)
  const [deletingThreadId, setDeletingThreadId] = useState<string | null>(null)
  const syncingEditorRef = useRef(false)
  const threadEndRef = useRef<HTMLDivElement | null>(null)
  const editor = useEditor({
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({
        blockquote: false,
        bulletList: false,
        codeBlock: false,
        code: false,
        dropcursor: false,
        gapcursor: false,
        heading: false,
        horizontalRule: false,
        listItem: false,
        orderedList: false,
      }),
      AssistantMention.configure({
        deleteTriggerWithBackspace: true,
      }),
    ],
    content: createPromptDoc(""),
    editorProps: {
      attributes: {
        class: "min-h-full whitespace-pre-wrap break-words outline-none",
      },
    },
    onUpdate: ({ editor: nextEditor }) => {
      if (syncingEditorRef.current) return
      const serialized = serializeEditor(nextEditor)
      setPrompt(serialized.prompt)
      setTokenPool(serialized.tokens)
      setTokenMatch(getEditorTokenMatch(nextEditor))
    },
    onSelectionUpdate: ({ editor: nextEditor }) => {
      if (syncingEditorRef.current) return
      setTokenMatch(getEditorTokenMatch(nextEditor))
    },
  })

  useEffect(() => {
    if (typeof window === "undefined") return
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) return
    try {
      const parsed = JSON.parse(raw) as ThreadRecord[]
      if (parsed.length > 0) {
        setThreads(parsed)
        setActiveThreadId(parsed[0].id)
      }
    } catch {
      // ignore malformed local state
    }
  }, [])

  useEffect(() => {
    if (typeof window === "undefined") return
    window.localStorage.setItem(storageKey, JSON.stringify(threads.slice(0, 20)))
  }, [threads])

  const activeThread = threads.find((thread) => thread.id === activeThreadId) ?? threads[0]
  const deferredQuery = useDeferredValue(tokenMatch?.query ?? "")
  const mentionIntent = useMemo(() => resolveMentionIntent(deferredQuery), [deferredQuery])
  const exactTimeOption = useMemo(() => getExactTimeOption(deferredQuery), [deferredQuery])
  const visibleFamilies = useMemo(() => {
    if (mentionIntent.family) return [mentionIntent.family]
    const lowered = mentionIntent.query
    if (!lowered) return resourceFamilies
    return resourceFamilies.filter((family) =>
      family.key.toLowerCase().includes(lowered) ||
      family.label.toLowerCase().includes(lowered) ||
      family.helper.toLowerCase().includes(lowered),
    )
  }, [mentionIntent])
  const effectiveResourceFilter = mentionIntent.family?.type ?? resourceFilter
  const visibleSuggestions = useMemo(() => {
    if (effectiveResourceFilter === "all") return suggestions
    return suggestions.filter((option) => option.kind === "resource" && option.type === effectiveResourceFilter)
  }, [effectiveResourceFilter, suggestions])

  useEffect(() => {
    if (!tokenMatch) {
      setSuggestions([])
      setSuggestionsOpen(false)
      setFamilySearchQuery("")
      setResourceFilter("all")
      return
    }

    let cancelled = false
    setSuggestionsLoading(true)
    const run = async () => {
      try {
        const searchQuery = mentionIntent.family ? familySearchQuery : mentionIntent.query
        const exactFamily = mentionIntent.family?.type

        const searches = await Promise.allSettled([
          exactFamily === undefined || exactFamily === "customer" ? searchCustomers(searchQuery) : Promise.resolve([] as Customer[]),
          exactFamily === undefined || exactFamily === "invoice" ? searchInvoices(searchQuery) : Promise.resolve([] as Invoice[]),
          exactFamily === undefined || exactFamily === "product" ? searchProducts(searchQuery) : Promise.resolve([] as Product[]),
          exactFamily === undefined || exactFamily === "subscription" ? searchSubscriptions(searchQuery) : Promise.resolve([] as Subscription[]),
          exactFamily === undefined || exactFamily === "meter" ? searchMeters(searchQuery) : Promise.resolve([] as Meter[]),
          exactFamily === undefined || exactFamily === "usage" ? searchUsageEvents(searchQuery) : Promise.resolve([] as UsageEvent[]),
          exactFamily === undefined || exactFamily === "feature" ? searchFeatures(searchQuery) : Promise.resolve([] as Feature[]),
          exactFamily === undefined || exactFamily === "audit_log" ? searchAuditLogs(searchQuery) : Promise.resolve([] as AuditLog[]),
        ])
        if (cancelled) return
        const [customers, invoices, products, subscriptions, meters, usageEvents, features, auditLogs] = searches
        const includeTimeOptions = !exactFamily
        const next = [
          ...(includeTimeOptions ? buildTimeOptions(searchQuery) : []),
          ...(customers.status === "fulfilled" ? customers.value.map(mapCustomerOption) : []),
          ...(invoices.status === "fulfilled" ? invoices.value.map(mapInvoiceOption) : []),
          ...(products.status === "fulfilled" ? products.value.map(mapProductOption) : []),
          ...(subscriptions.status === "fulfilled" ? subscriptions.value.map(mapSubscriptionOption) : []),
          ...(meters.status === "fulfilled" ? meters.value.map(mapMeterOption) : []),
          ...(usageEvents.status === "fulfilled" ? usageEvents.value.map(mapUsageOption) : []),
          ...(features.status === "fulfilled" ? features.value.map(mapFeatureOption) : []),
          ...(auditLogs.status === "fulfilled" ? auditLogs.value.map(mapAuditOption) : []),
        ].slice(0, 12)
        startTransition(() => {
          setSuggestions(next)
          setSuggestionsOpen(next.length > 0 || visibleFamilies.length > 0 || Boolean(mentionIntent.family))
          setSuggestionIndex(0)
        })
      } catch {
        if (!cancelled) {
          setSuggestions([])
          setSuggestionsOpen(false)
        }
      } finally {
        if (!cancelled) setSuggestionsLoading(false)
      }
    }
    void run()

    return () => {
      cancelled = true
    }
  }, [familySearchQuery, mentionIntent, tokenMatch, visibleFamilies.length])

  useEffect(() => {
    if (mentionIntent.family) return
    if (resourceFilter === "all") return
    if (!visibleFamilies.some((family) => family.type === resourceFilter)) {
      setResourceFilter("all")
    }
  }, [mentionIntent.family, resourceFilter, visibleFamilies])

  useEffect(() => {
    if (visibleSuggestions.length === 0) {
      setSuggestionIndex(0)
      return
    }
    if (suggestionIndex >= visibleSuggestions.length) {
      setSuggestionIndex(0)
    }
  }, [suggestionIndex, visibleSuggestions.length])

  useEffect(() => {
    threadEndRef.current?.scrollIntoView({ behavior: "smooth", block: "end" })
  }, [activeThread?.messages.length, submitting])

  useEffect(() => {
    if (!tokenMatch || timePickerState || mentionIntent.family || !exactTimeOption) return
    openTimePicker(exactTimeOption)
  }, [exactTimeOption, mentionIntent.family, timePickerState, tokenMatch])

  const setEditorPlainText = (value: string) => {
    if (!editor) {
      setPrompt(value)
      setTokenPool([])
      setTokenMatch(null)
      return
    }
    syncingEditorRef.current = true
    editor.commands.setContent(createPromptDoc(value), { emitUpdate: false })
    editor.commands.focus("end")
    syncingEditorRef.current = false
    setPrompt(value)
    setTokenPool([])
    setTokenMatch(getEditorTokenMatch(editor))
  }

  const replaceActiveToken = (option: TokenOption) => {
    if (!editor || !tokenMatch) return
    const label = getTokenPromptText(option.token)
    editor
      .chain()
      .focus()
      .deleteRange({ from: tokenMatch.start, to: tokenMatch.end })
      .insertContent({
        type: "assistantMention",
        attrs: {
          token: option.token,
          label,
        } satisfies MentionAttrs,
      })
      .insertContent(" ")
      .run()
    setSuggestionsOpen(false)
    setFamilySearchQuery("")
  }

  const openTimePicker = (option: TokenOption) => {
    if (!tokenMatch) return
    setSuggestionsOpen(false)
    setTimePickerState({ option, match: tokenMatch })
  }

  const applyTimePicker = (values: { value?: string; from?: string; to?: string }) => {
    if (!editor || !timePickerState) return
    const nextToken = buildTimeTokenFromValues(timePickerState.option, values)
    const label = getTokenPromptText(nextToken)
    editor
      .chain()
      .focus()
      .deleteRange({ from: timePickerState.match.start, to: timePickerState.match.end })
      .insertContent({
        type: "assistantMention",
        attrs: {
          token: nextToken,
          label,
        } satisfies MentionAttrs,
      })
      .insertContent(" ")
      .run()
    setTimePickerState(null)
  }

  const selectSuggestion = (option: TokenOption) => {
    if (option.kind === "time" && option.type !== "relative") {
      openTimePicker(option)
      return
    }
    replaceActiveToken(option)
  }

  const activateMentionFamily = (family: ResourceFamily) => {
    if (!editor || !tokenMatch) return
    editor
      .chain()
      .focus()
      .deleteRange({ from: tokenMatch.start, to: tokenMatch.end })
      .insertContent(family.key)
      .run()
    setFamilySearchQuery("")
    setResourceFilter(family.type)
    setSuggestionsOpen(true)
    setSuggestionIndex(0)
  }

  const handlePromptKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (suggestionsOpen && visibleSuggestions.length > 0) {
      if (event.key === "ArrowDown") {
        event.preventDefault()
        setSuggestionIndex((current) => (current + 1) % visibleSuggestions.length)
        return
      }
      if (event.key === "ArrowUp") {
        event.preventDefault()
        setSuggestionIndex((current) => (current - 1 + visibleSuggestions.length) % visibleSuggestions.length)
        return
      }
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault()
        selectSuggestion(visibleSuggestions[suggestionIndex])
        return
      }
      if (event.key === "Escape") {
        event.preventDefault()
        setSuggestionsOpen(false)
        return
      }
    }

    if (event.key === "Enter" && event.shiftKey) {
      event.preventDefault()
      editor?.chain().focus().setHardBreak().run()
      return
    }

    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault()
      void handleSubmit()
    }
  }

  const updateThread = (threadId: string, updater: (thread: ThreadRecord) => ThreadRecord) => {
    setThreads((current) => current.map((thread) => (thread.id === threadId ? updater(thread) : thread)))
  }

  const openFreshThread = () => {
    const next = createEmptyThread()
    setThreads((current) => [next, ...current])
    setActiveThreadId(next.id)
    setEditorPlainText("")
    setSuggestions([])
    setSuggestionsOpen(false)
    setFamilySearchQuery("")
    setTokenMatch(null)
    setResourceFilter("all")
  }

  const loadThread = (threadId: string) => {
    setActiveThreadId(threadId)
    setEditorPlainText("")
    setSuggestions([])
    setSuggestionsOpen(false)
    setFamilySearchQuery("")
    setTokenMatch(null)
    setResourceFilter("all")
  }

  const deleteThreadByID = async (threadId: string) => {
    const target = threads.find((thread) => thread.id === threadId)
    if (!target) return
    const persistedThreadID = resolvePersistedThreadID(target)
    const activeOrgId = getActiveOrgIdFromRoute()

    if (persistedThreadID && activeOrgId) {
      setDeletingThreadId(threadId)
      try {
        await api.ai.deleteThread(persistedThreadID, { orgId: activeOrgId })
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to delete thread."
        toast.error("Failed to delete thread.", message)
        setDeletingThreadId(null)
        return
      }
      setDeletingThreadId(null)
    }

    const remaining = threads.filter((thread) => thread.id !== threadId)
    if (remaining.length === 0) {
      const next = createEmptyThread()
      setThreads([next])
      setActiveThreadId(next.id)
    } else {
      setThreads(remaining)
      if (activeThreadId === threadId) {
        setActiveThreadId(remaining[0].id)
      }
    }
    setEditorPlainText("")
    setSuggestions([])
    setSuggestionsOpen(false)
    setFamilySearchQuery("")
    setTokenMatch(null)
    setResourceFilter("all")
  }

  const handleSubmit = async () => {
    if (!prompt.trim()) {
      toast.error("Prompt is required.")
      return
    }
    const thread = activeThread ?? createEmptyThread()
    const threadId = thread.id
    if (!threads.some((item) => item.id === threadId)) {
      setThreads((current) => [thread, ...current])
      setActiveThreadId(threadId)
    }

    const payload: AIPromptCreateRequest = {
      prompt: prompt.trim(),
      tokens: resolveUsedTokens(prompt.trim(), tokenPool),
      conversation_id: thread.conversationId,
    }
    const userMessage = buildUserMessage(payload.prompt, payload.tokens)

    updateThread(threadId, (current) => {
      const messages = [...current.messages, userMessage]
      return {
        ...current,
        title: summarizeThread(messages),
        messages,
        updatedAt: new Date().toISOString(),
      }
    })

    setSubmitting(true)
    setEditorPlainText("")
    setSuggestions([])
    setSuggestionsOpen(false)
    setTokenMatch(null)

    try {
      const activeOrgId = getActiveOrgIdFromRoute()
      const resp = await api.ai.createPrompt(payload, activeOrgId ? { orgId: activeOrgId } : undefined)
      const assistantMessage = ensureAssistantMessage(resp)
      updateThread(threadId, (current) => ({
        ...current,
        conversationId: resp.conversation_id || current.conversationId,
        title: summarizeThread(current.messages),
        messages: [...current.messages, assistantMessage],
        updatedAt: new Date().toISOString(),
      }))
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "The AI endpoint did not return a valid response."
      updateThread(threadId, (current) => ({
        ...current,
        messages: [
          ...current.messages,
          {
            id: `assistant-error-${crypto.randomUUID()}`,
            role: "assistant",
            created_at: new Date().toISOString(),
            blocks: buildAssistantErrorBlocks(errorMessage),
          },
        ],
        updatedAt: new Date().toISOString(),
      }))
      toast.error("Failed to send prompt.", errorMessage)
    } finally {
      setSubmitting(false)
    }
  }

  const activeMessages = activeThread?.messages ?? []
  const hasConversation = activeMessages.length > 0
  const starterPrompts = [
    "Why is the invoice for @customer higher than usual this month?",
    "Compare usage across @subscription over @last_30_days",
    "What are the most important changes in @audit_log since @date?",
    "Summarize all overdue invoices for @customer",
    "Which @product has the highest usage this @month?",
    "Show me a breakdown of @meter usage for @subscription",
  ]
  const displayed = useMemo(() =>
    [...starterPrompts].sort(() => Math.random() - 0.5).slice(0, 3)
    , [])

  const launchTokenPicker = (filter: AIPromptTokenResourceType | "all" = "all") => {
    if (!editor) return
    const text = prompt
    const needsSpace = text.trim() !== "" && !/\s$/.test(text)
    editor.chain().focus().insertContent(`${needsSpace ? " " : ""}@`).run()
    setResourceFilter(filter)
    setSuggestionsOpen(true)
  }

  const launchTimeShortcut = (option: TokenOption) => {
    if (!editor) return
    const text = prompt
    const needsSpace = text.trim() !== "" && !/\s$/.test(text)
    editor.chain().focus().insertContent(`${needsSpace ? " " : ""}${option.key}`).run()
    const nextMatch = getEditorTokenMatch(editor)
    if (nextMatch) {
      setTimePickerState({ option, match: nextMatch })
    }
    setSuggestionsOpen(false)
  }

  return (
    <div
      className="h-screen overflow-hidden p-3 sm:p-4"
      style={{
        background:
          "radial-gradient(circle at top, hsl(var(--accent-primary) / 0.12), transparent 34%), hsl(var(--bg-surface))",
      }}
    >
      <div className="mx-auto flex h-full max-w-[1680px] gap-3 sm:gap-4">
        <aside className="hidden w-[280px] shrink-0 flex-col overflow-hidden rounded-[32px] border bg-[hsl(var(--bg-primary))] shadow-[var(--shadow-lg)] lg:flex">
          <div className="border-b border-[hsl(var(--border-subtle))] px-5 py-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs font-semibold uppercase tracking-[0.24em] text-[hsl(var(--text-muted))]">Conversations</div>
                <div className="mt-2 text-lg font-semibold text-[hsl(var(--text-primary))]">AI Assistant</div>
              </div>
              <Button variant="default" onClick={openFreshThread}>New</Button>
            </div>
          </div>
          <div className="min-h-0 flex-1 space-y-2 overflow-y-auto px-4 py-4">
            {threads.map((thread) => (
              <div
                key={thread.id}
                className={cn(
                  "rounded-[22px] border px-4 py-3 transition",
                  thread.id === activeThread?.id
                    ? "border-[hsl(var(--border-strong))] bg-[hsl(var(--bg-surface-strong))]"
                    : "border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] hover:border-[hsl(var(--border-strong))] hover:bg-[hsl(var(--bg-surface-strong))]",
                )}
              >
                <div className="flex items-start justify-between gap-3">
                  <button type="button" onClick={() => loadThread(thread.id)} className="min-w-0 flex-1 text-left">
                    <div className="flex items-center gap-2 text-[11px] uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">
                      <IconThread />
                      Thread
                    </div>
                    <div className="mt-2 text-sm font-semibold text-[hsl(var(--text-primary))]">{thread.title}</div>
                    <div className="mt-1 text-xs text-[hsl(var(--text-muted))]">{formatDateTime(thread.updatedAt)}</div>
                  </button>
                  <button
                    type="button"
                    onClick={() => void deleteThreadByID(thread.id)}
                    disabled={deletingThreadId === thread.id}
                    className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] text-[hsl(var(--text-muted))] transition hover:border-[hsl(var(--border-strong))] hover:text-[hsl(var(--text-primary))]"
                    aria-label={`Delete ${thread.title}`}
                  >
                    <IconTrash />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </aside>

        <section className="flex min-w-0 flex-1 flex-col overflow-visible rounded-[32px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] shadow-[var(--shadow-lg)]">
          <div className="flex shrink-0 items-center justify-between gap-3 border-b border-[hsl(var(--border-subtle))] px-4 py-4 sm:px-6">
            <div className="flex items-center gap-3">
              <Button variant="secondary" onClick={() => navigate(orgPath("/dashboard"))} className="gap-2">
                <IconBack />
                Back
              </Button>
              <div className="hidden sm:block">
                <div className="text-lg font-semibold text-[hsl(var(--text-primary))]">AI Assistant</div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="default" onClick={openFreshThread}>New chat</Button>
            </div>
          </div>

          <div className="border-b border-[hsl(var(--border-subtle))] px-4 py-3 lg:hidden">
            <div className="flex gap-2 overflow-x-auto">
              {threads.map((thread) => (
                <div
                  key={thread.id}
                  className={cn(
                    "flex shrink-0 items-center gap-2 rounded-full border pl-3 pr-2 py-2 text-sm transition",
                    thread.id === activeThread?.id
                      ? "border-[hsl(var(--border-strong))] bg-[hsl(var(--bg-surface-strong))] text-[hsl(var(--text-primary))]"
                      : "border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] text-[hsl(var(--text-secondary))]",
                  )}
                >
                  <button type="button" onClick={() => loadThread(thread.id)} className="truncate text-left">
                    {thread.title}
                  </button>
                  <button
                    type="button"
                    onClick={() => void deleteThreadByID(thread.id)}
                    disabled={deletingThreadId === thread.id}
                    className="inline-flex h-7 w-7 items-center justify-center rounded-full text-[hsl(var(--text-muted))] transition hover:bg-[hsl(var(--bg-primary))] hover:text-[hsl(var(--text-primary))]"
                    aria-label={`Delete ${thread.title}`}
                  >
                    <IconTrash />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {!hasConversation ? (
            <div className="flex min-h-0 flex-1 overflow-y-auto px-6 py-8 sm:px-10">
              <div className="mx-auto flex w-full max-w-4xl flex-col justify-center text-center">
                <h1 className="mt-8 text-4xl font-semibold tracking-tight text-[hsl(var(--text-primary))] sm:text-5xl">
                  Your billing intelligence,{" "}
                  <span className="text-[hsl(var(--color-primary))]">in one place.</span>
                </h1>
                <p className="mx-auto mt-4 max-w-2xl text-base leading-8 text-[hsl(var(--text-secondary))]">
                  Use natural language, then ground your request with context tokens like{" "}
                  {["@customer", "@subscription", "@product", "@meter", "@usage", "@feature", "@audit_log"].map((token) => (
                    <span
                      key={token}
                      className="inline-block rounded-md bg-[hsl(var(--surface-raised))] px-1.5 py-0.5 text-sm font-mono text-[hsl(var(--text-primary))] mx-0.5"
                    >
                      {token}
                    </span>
                  ))}
                  {" "}and time tokens.
                </p>

                <div className="mt-10 rounded-[28px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] p-2.5 shadow-[var(--shadow-md)]">
                  <div className="relative text-left">
                    {!prompt ? (
                      <div className="pointer-events-none absolute left-4 right-14 top-3.5 text-[15px] leading-6 text-[hsl(var(--text-muted))]">
                        Ask anything about billing — use{" "}
                        <span className="text-[hsl(var(--color-primary))] opacity-60">@mentions</span>
                        {" "}to add context...
                      </div>
                    ) : null}
                    <div className="min-h-[68px] max-h-[180px] overflow-y-auto rounded-[22px] bg-[hsl(var(--bg-primary))] px-4 py-3 pr-14 text-[15px] leading-6 text-[hsl(var(--text-primary))]">
                      <EditorContent editor={editor} onKeyDown={handlePromptKeyDown} />
                    </div>
                    {timePickerState ? (
                      <TimeTokenPicker
                        option={timePickerState.option}
                        onCancel={() => {
                          setTimePickerState(null)
                          requestAnimationFrame(() => editor?.commands.focus())
                        }}
                        onApply={applyTimePicker}
                      />
                    ) : null}
                    <Button
                      type="button"
                      variant="default"
                      disabled={submitting || !prompt.trim()}
                      onClick={() => void handleSubmit()}
                      className="absolute bottom-2.5 right-2.5 h-9 min-w-[36px] rounded-full px-2.5"
                    >
                      <IconSend />
                    </Button>

                    {suggestionsOpen ? (
                      <div className="absolute left-0 right-0 bottom-[calc(100%+12px)] z-20 overflow-hidden rounded-[24px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] shadow-[var(--shadow-lg)]">
                        <div className="border-b border-[hsl(var(--border-subtle))] px-4 py-3 text-xs font-semibold uppercase tracking-[0.22em] text-[hsl(var(--text-muted))]">
                          {suggestionsLoading ? "Searching…" : "Insert token"}
                        </div>
                        <div className="max-h-[320px] overflow-y-auto p-2">
                          {!mentionIntent.family && visibleFamilies.length > 0 ? (
                            <div className="space-y-1 border-b border-[hsl(var(--border-subtle))] pb-2">
                              {visibleFamilies.map((family) => (
                                <button
                                  key={family.type}
                                  type="button"
                                  onMouseDown={(event) => {
                                    event.preventDefault()
                                    activateMentionFamily(family)
                                  }}
                                  className={cn(
                                    "flex w-full items-center gap-3 rounded-[18px] px-3 py-3 text-left transition",
                                    effectiveResourceFilter === family.type
                                      ? "bg-[hsl(var(--bg-surface-strong))] text-[hsl(var(--text-primary))]"
                                      : "text-[hsl(var(--text-primary))] hover:bg-[hsl(var(--bg-surface))]",
                                  )}
                                >
                                  <div className="inline-flex h-10 min-w-[40px] items-center justify-center rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] text-[11px] font-semibold uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">
                                    {getFamilyBadge(family)}
                                  </div>
                                  <div className="min-w-0">
                                    <div className="text-[15px] font-medium">{family.label}</div>
                                    <div className="text-sm text-[hsl(var(--text-muted))]">{family.helper}</div>
                                  </div>
                                </button>
                              ))}
                            </div>
                          ) : null}
                          {mentionIntent.family ? (
                            <div className="space-y-3 border-b border-[hsl(var(--border-subtle))] px-4 py-3">
                              <div className="text-sm text-[hsl(var(--text-secondary))]">
                                Search within <span className="font-medium text-[hsl(var(--text-primary))]">{mentionIntent.family.label}</span>.
                              </div>
                              <Input
                                value={familySearchQuery}
                                onChange={(event) => setFamilySearchQuery(event.target.value)}
                                placeholder={`Search ${mentionIntent.family.label.toLowerCase()}...`}
                                className="h-10 rounded-full border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))]"
                                autoFocus
                              />
                            </div>
                          ) : null}
                          {visibleSuggestions.length === 0 ? (
                            <div className="px-4 py-4 text-sm leading-6 text-[hsl(var(--text-secondary))]">
                              {getSuggestionEmptyText(mentionIntent, suggestionsLoading)}
                            </div>
                          ) : null}
                          {visibleSuggestions.map((option, index) => (
                            <button
                              key={option.id}
                              type="button"
                              onMouseDown={(event) => {
                                event.preventDefault()
                                selectSuggestion(option)
                              }}
                              className={cn(
                                "flex w-full items-start gap-3 rounded-[18px] px-3 py-3 text-left transition",
                                index === suggestionIndex
                                  ? "bg-[hsl(var(--bg-surface-strong))] text-[hsl(var(--text-primary))]"
                                  : "text-[hsl(var(--text-primary))] hover:bg-[hsl(var(--bg-surface))]",
                              )}
                            >
                              <div className="mt-0.5 inline-flex h-10 min-w-[40px] items-center justify-center rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] text-[11px] font-semibold uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">
                                {getMentionBadge(option)}
                              </div>
                              <div className="min-w-0">
                                <div className="text-[15px] font-medium">{option.label}</div>
                                <div className="text-sm text-[hsl(var(--text-muted))]">{option.secondaryLabel || option.descriptor}</div>
                              </div>
                            </button>
                          ))}
                        </div>
                      </div>
                    ) : null}
                  </div>

                  <div className="mt-4 flex flex-wrap gap-2">
                    {resourceFamilies.map((family) => (
                      <EmptyPrompt key={family.type} label={family.key} onClick={() => launchTokenPicker(family.type)} />
                    ))}
                    <EmptyPrompt label="@date" onClick={() => launchTimeShortcut(buildTimeOptions("").find((option) => option.key === "@date")!)} />
                    <EmptyPrompt label="@month" onClick={() => launchTimeShortcut(buildTimeOptions("").find((option) => option.key === "@month")!)} />
                  </div>
                </div>

                <div className="mt-6 flex flex-wrap justify-center gap-2">
                  {displayed.map((item) => (
                    <EmptyPrompt key={item} label={item} onClick={() => setEditorPlainText(item)} />
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <>
              <div className="min-h-0 flex-1 overflow-y-auto bg-[hsl(var(--bg-surface))] px-4 py-6 sm:px-6">
                <div className="mx-auto flex min-h-full w-full max-w-4xl flex-col justify-end space-y-5">
                  {activeMessages.map((message) =>
                    message.role === "assistant" ? (
                      <AssistantMessage key={message.id} message={message} />
                    ) : (
                      <UserMessage key={message.id} message={message} />
                    ),
                  )}
                  {submitting ? (
                    <div className="flex justify-start">
                      <div className="rounded-[24px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] px-5 py-4 text-sm text-[hsl(var(--text-muted))]">
                        Waiting...
                      </div>
                    </div>
                  ) : null}
                  <div ref={threadEndRef} />
                </div>
              </div>

              <div className="shrink-0 border-t border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] px-4 py-4 sm:px-6">
                <div className="mx-auto w-full max-w-4xl">
                  <div className="relative overflow-visible rounded-[26px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] p-2 shadow-[var(--shadow-md)]">
                    {!prompt ? (
                      <div className="pointer-events-none absolute left-4 right-14 top-3.5 text-[14px] leading-6 text-[hsl(var(--text-muted))]">
                        Ask anything — use @ to add context like @customer or @invoice
                      </div>
                    ) : null}
                    <div className="min-h-[56px] max-h-[168px] overflow-y-auto rounded-[22px] bg-[hsl(var(--bg-primary))] px-4 py-2.5 pr-14 text-[15px] leading-6 text-[hsl(var(--text-primary))]">
                      <EditorContent editor={editor} onKeyDown={handlePromptKeyDown} />
                    </div>
                    {timePickerState ? (
                      <TimeTokenPicker
                        option={timePickerState.option}
                        onCancel={() => {
                          setTimePickerState(null)
                          requestAnimationFrame(() => editor?.commands.focus())
                        }}
                        onApply={applyTimePicker}
                      />
                    ) : null}
                    <Button
                      type="button"
                      variant="default"
                      disabled={submitting || !prompt.trim()}
                      onClick={() => void handleSubmit()}
                      className="absolute bottom-2.5 right-2.5 h-9 min-w-[36px] rounded-full px-2.5"
                    >
                      <IconSend />
                    </Button>

                    {suggestionsOpen ? (
                      <div className="absolute left-3 right-3 bottom-[calc(100%+12px)] z-30 overflow-hidden rounded-[24px] border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-primary))] shadow-[var(--shadow-lg)]">
                        <div className="border-b border-[hsl(var(--border-subtle))] px-4 py-3 text-xs font-semibold uppercase tracking-[0.22em] text-[hsl(var(--text-muted))]">
                          {suggestionsLoading ? "Searching…" : "Insert token"}
                        </div>
                        <div className="max-h-[280px] overflow-y-auto p-2">
                          {!mentionIntent.family && visibleFamilies.length > 0 ? (
                            <div className="space-y-1 border-b border-[hsl(var(--border-subtle))] pb-2">
                              {visibleFamilies.map((family) => (
                                <button
                                  key={family.type}
                                  type="button"
                                  onMouseDown={(event) => {
                                    event.preventDefault()
                                    activateMentionFamily(family)
                                  }}
                                  className={cn(
                                    "flex w-full items-center gap-3 rounded-[18px] px-3 py-3 text-left transition",
                                    effectiveResourceFilter === family.type
                                      ? "bg-[hsl(var(--bg-surface-strong))] text-[hsl(var(--text-primary))]"
                                      : "text-[hsl(var(--text-primary))] hover:bg-[hsl(var(--bg-surface))]",
                                  )}
                                >
                                  <div className="inline-flex h-10 min-w-[40px] items-center justify-center rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] text-[11px] font-semibold uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">
                                    {getFamilyBadge(family)}
                                  </div>
                                  <div className="min-w-0">
                                    <div className="text-[15px] font-medium">{family.label}</div>
                                    <div className="text-sm text-[hsl(var(--text-muted))]">{family.helper}</div>
                                  </div>
                                </button>
                              ))}
                            </div>
                          ) : null}
                          {mentionIntent.family ? (
                            <div className="space-y-3 border-b border-[hsl(var(--border-subtle))] px-4 py-3">
                              <div className="text-sm text-[hsl(var(--text-secondary))]">
                                Search within <span className="font-medium text-[hsl(var(--text-primary))]">{mentionIntent.family.label}</span>.
                              </div>
                              <Input
                                value={familySearchQuery}
                                onChange={(event) => setFamilySearchQuery(event.target.value)}
                                placeholder={`Search ${mentionIntent.family.label.toLowerCase()}...`}
                                className="h-10 rounded-full border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))]"
                                autoFocus
                              />
                            </div>
                          ) : null}
                          {visibleSuggestions.length === 0 ? (
                            <div className="px-4 py-4 text-sm leading-6 text-[hsl(var(--text-secondary))]">
                              {getSuggestionEmptyText(mentionIntent, suggestionsLoading)}
                            </div>
                          ) : null}
                          {visibleSuggestions.map((option, index) => (
                            <button
                              key={option.id}
                              type="button"
                              onMouseDown={(event) => {
                                event.preventDefault()
                                selectSuggestion(option)
                              }}
                              className={cn(
                                "flex w-full items-start gap-3 rounded-[18px] px-3 py-3 text-left transition",
                                index === suggestionIndex
                                  ? "bg-[hsl(var(--bg-surface-strong))] text-[hsl(var(--text-primary))]"
                                  : "text-[hsl(var(--text-primary))] hover:bg-[hsl(var(--bg-surface))]",
                              )}
                            >
                              <div className="mt-0.5 inline-flex h-10 min-w-[40px] items-center justify-center rounded-full border border-[hsl(var(--border-subtle))] bg-[hsl(var(--bg-surface))] text-[11px] font-semibold uppercase tracking-[0.18em] text-[hsl(var(--text-muted))]">
                                {getMentionBadge(option)}
                              </div>
                              <div className="min-w-0">
                                <div className="text-[15px] font-medium">{option.label}</div>
                                <div className="text-sm text-[hsl(var(--text-muted))]">{option.secondaryLabel || option.descriptor}</div>
                              </div>
                            </button>
                          ))}
                        </div>
                      </div>
                    ) : null}
                  </div>

                  <div className="mt-3 flex flex-wrap gap-2">
                    {resourceFamilies.slice(0, 5).map((family) => (
                      <EmptyPrompt key={family.type} label={family.key} onClick={() => launchTokenPicker(family.type)} />
                    ))}
                    <EmptyPrompt label="@date" onClick={() => launchTimeShortcut(buildTimeOptions("").find((option) => option.key === "@date")!)} />
                    <EmptyPrompt label="@range" onClick={() => launchTimeShortcut(buildTimeOptions("").find((option) => option.key === "@range")!)} />
                  </div>
                </div>
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  )
}

export default function AIAssistant() {
  return <AIAssistantWorkspace />
}
