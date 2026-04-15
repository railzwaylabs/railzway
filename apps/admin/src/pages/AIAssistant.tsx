import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import PageHeader from "../components/PageHeader"
import StatCard from "../components/StatCard"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Label } from "../components/ui/label"
import { Textarea } from "../components/ui/textarea"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs"
import { toast } from "../components/Toast"
import { api } from "../lib/api"
import { formatDateTime } from "../lib/display"
import { useOrgPath } from "../lib/org"
import type {
  AIAssistantAction,
  AIAssistantConfidence,
  AIAssistantDriver,
  AIAssistantInsight,
  AIAssistantOverviewResponse,
  AIAssistantRunDetail,
  AIAssistantRunsResponse,
  AIAssistantSummary
} from "../lib/types"

type InsightStatus = "idle" | "loading" | "ready" | "error"

type ImpactLevel = AIAssistantDriver["impact"]

type ConfidenceLevel = AIAssistantConfidence["level"]

function mapWorkflowActionType(actionKey: string) {
  switch (actionKey) {
    case "create_product":
      return "create_product"
    case "create_plan":
      return "create_plan"
    case "create_meter":
      return "create_meter"
    case "create_feature":
      return "create_feature"
    case "flag_review":
      return "notify"
    default:
      return "navigate"
  }
}

function toCode(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-_]/g, "")
}

function IconSpark() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M10 2l1.4 3.6L15 7l-3.6 1.4L10 12l-1.4-3.6L5 7l3.6-1.4L10 2z" strokeLinejoin="round"/>
      <path d="M4 13l.8 2 2 .8-2 .8-.8 2-.8-2-2-.8 2-.8.8-2z" strokeLinejoin="round"/>
      <path d="M16 12l.6 1.4 1.4.6-1.4.6-.6 1.4-.6-1.4-1.4-.6 1.4-.6.6-1.4z" strokeLinejoin="round"/>
    </svg>
  )
}

function AIConfidenceBadge({ level }: { level: ConfidenceLevel }) {
  const { t } = useTranslation()
  const styles: Record<ConfidenceLevel, string> = {
    high: "bg-white text-slate-900 border-slate-200",
    medium: "bg-white text-slate-700 border-slate-200",
    low: "bg-white text-slate-500 border-slate-200"
  }
  const label = level.charAt(0).toUpperCase() + level.slice(1)
  return (
    <span className={`inline-flex items-center gap-2 rounded-full border px-2.5 py-1 text-xs font-semibold ${styles[level]}`}>
      {t("ai_assistant.output_blocks.confidence_label", { level: label })}
    </span>
  )
}

function AISummaryCard({ headline, metric, metricNote }: { headline: string; metric: string; metricNote: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white px-5 py-4">
      <div className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">
        {t("ai_assistant.output_blocks.summary")}
      </div>
      <div className="flex flex-wrap items-baseline justify-between gap-4">
        <p className="text-sm leading-6 text-slate-700">{headline}</p>
        <div className="text-right">
          <div className="text-3xl font-semibold tracking-tight text-slate-900">{metric}</div>
          <div className="text-xs text-slate-400">{metricNote}</div>
        </div>
      </div>
    </div>
  )
}

function AIDriverList({ drivers }: { drivers: AIAssistantDriver[] }) {
  const { t } = useTranslation()
  if (!drivers || drivers.length === 0) return null
  const impactStyles: Record<ImpactLevel, string> = {
    high: "bg-slate-900",
    medium: "bg-slate-500",
    low: "bg-slate-300"
  }
  return (
    <div className="rounded-xl border border-slate-200 bg-white">
      <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3">
        <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">
          {t("ai_assistant.output_blocks.drivers")}
        </p>
        <span className="text-xs text-slate-400">{t("ai_assistant.output_blocks.drivers_count", { count: drivers.length })}</span>
      </div>
      <div className="divide-y divide-slate-100">
        {drivers.map((driver) => (
          <div key={driver.label} className="flex flex-wrap items-center justify-between gap-3 px-5 py-3">
            <div className="min-w-[180px]">
              <div className="text-sm font-medium text-slate-900">{driver.label}</div>
              <div className="text-xs leading-5 text-slate-500">{driver.detail}</div>
            </div>
            <span className="inline-flex items-center gap-2 text-xs font-medium text-slate-500">
              <span className={`h-2 w-2 rounded-full ${impactStyles[driver.impact]}`} />
              {driver.impact.toUpperCase()}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

function AIAnomalyBlock({ anomalies }: { anomalies?: AIAssistantInsight["anomalies"] }) {
  const { t } = useTranslation()
  if (!anomalies || anomalies.length === 0) return null
  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50 px-5 py-4">
      <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">
        {t("ai_assistant.output_blocks.anomalies")}
      </div>
      <div className="flex flex-col gap-3">
        {anomalies.map((item) => (
          <div key={item.title} className="flex items-start justify-between gap-3">
            <div>
              <div className="text-sm font-semibold text-slate-900">{item.title}</div>
              <div className="text-xs leading-5 text-slate-500">{item.detail}</div>
            </div>
            <span className="rounded-full border border-slate-200 bg-white px-2 py-1 text-[11px] font-semibold text-slate-500">
              {item.severity === "risk" ? t("ai_assistant.output_blocks.anomaly_risk") : t("ai_assistant.output_blocks.anomaly_watch")}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

function AIConfidenceRow({ confidence, dataQuality }: { confidence: AIAssistantConfidence; dataQuality: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-5 py-3">
      <div className="text-xs text-slate-500">
        <span className="font-semibold text-slate-600">{t("ai_assistant.output_blocks.data_quality")}:</span> {dataQuality}
      </div>
      <AIConfidenceBadge level={confidence.level} />
    </div>
  )
}

function AIActionBar({ actions, onAction }: { actions: AIAssistantAction[]; onAction: (action: AIAssistantAction) => void }) {
  if (!actions || actions.length === 0) return null
  return (
    <div className="flex flex-wrap gap-2">
      {actions.map((action) => (
        <Button
          key={action.key}
          size="sm"
          variant={action.style === "primary" ? "default" : "outline"}
          onClick={() => onAction(action)}
          disabled={action.disabled}
        >
          {action.label}
        </Button>
      ))}
    </div>
  )
}

function AISnapshotCard({ snapshot }: { snapshot?: AIAssistantInsight["snapshot"] }) {
  const { t } = useTranslation()
  if (!snapshot) return null
  return (
    <div className="grid gap-2 rounded-xl border border-slate-200 bg-white px-5 py-4 sm:grid-cols-[1.2fr,1fr]">
      <div>
        <div className="text-[11px] uppercase tracking-[0.12em] text-slate-400">{snapshot.label}</div>
        <div className="mt-2 flex items-baseline gap-3">
          <div className="text-lg font-semibold text-slate-900">{snapshot.current}</div>
          <span className="rounded-full border border-slate-200 bg-white px-2 py-1 text-xs font-semibold text-slate-600">
            {snapshot.delta}
          </span>
        </div>
        <div className="text-xs text-slate-500">{t("ai_assistant.output_blocks.previous")}: {snapshot.previous}</div>
      </div>
      <div className="rounded-lg bg-slate-50 px-3 py-2 text-xs leading-5 text-slate-500">
        {t("ai_assistant.output_blocks.snapshot_note")}
      </div>
    </div>
  )
}

function AIPlanRecommendation({ rec }: { rec?: AIAssistantInsight["plan_recommendation"] }) {
  const { t } = useTranslation()
  if (!rec) return null
  return (
    <div className="rounded-xl border border-slate-200 bg-white px-5 py-4">
      <div className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">
        {t("ai_assistant.output_blocks.plan_recommendation")}
      </div>
      <div className="mt-3 grid gap-4 sm:grid-cols-[1fr,1fr]">
        <div>
          <div className="text-xs text-slate-500">{t("ai_assistant.output_blocks.current_plan")}</div>
          <div className="text-sm font-semibold text-slate-900">{rec.current_plan}</div>
        </div>
        <div>
          <div className="text-xs text-slate-500">{t("ai_assistant.output_blocks.recommended_plan")}</div>
          <div className="text-sm font-semibold text-slate-900">{rec.recommended_plan}</div>
        </div>
      </div>
      <div className="mt-3 flex flex-wrap gap-3 text-xs text-slate-500">
        <span className="rounded-full border border-slate-200 bg-white px-2 py-1">
          {rec.savings_estimate}
        </span>
        <span className="rounded-full border border-slate-200 bg-white px-2 py-1">
          {rec.billing_impact}
        </span>
      </div>
      <div className="mt-3 text-xs text-slate-500">{rec.reason_summary}</div>
    </div>
  )
}

function AIOutputSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      <div className="h-20 animate-pulse rounded-xl bg-slate-100" />
      <div className="h-40 animate-pulse rounded-xl bg-slate-100" />
      <div className="h-24 animate-pulse rounded-xl bg-slate-100" />
      <div className="h-12 animate-pulse rounded-xl bg-slate-100" />
    </div>
  )
}

function AIOutputEmpty({ onRun }: { onRun: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-start gap-3 rounded-xl border border-dashed border-slate-200 bg-white p-6 text-sm text-slate-600">
      <div className="text-sm font-semibold text-slate-900">{t("ai_assistant.output_blocks.empty_title")}</div>
      <div className="text-xs leading-5 text-slate-500">{t("ai_assistant.output_blocks.empty_desc")}</div>
      <Button size="sm" onClick={onRun}>{t("ai_assistant.actions.run")}</Button>
    </div>
  )
}

function AIOutputError({ message, onRetry }: { message?: string; onRetry: () => void }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-rose-200 bg-rose-50/60 p-5">
      <div className="text-sm font-semibold text-rose-700">{t("ai_assistant.output_blocks.error_title")}</div>
      <div className="text-xs text-rose-600">{message ?? t("ai_assistant.output_blocks.error_desc")}</div>
      <Button size="sm" variant="outline" onClick={onRetry}>{t("ai_assistant.output_blocks.retry")}</Button>
    </div>
  )
}

function AIProductRecommendations({ items }: { items?: AIAssistantInsight["product_recommendations"] }) {
  const { t } = useTranslation()
  if (!items || items.length === 0) return null
  return (
    <div className="rounded-xl border border-slate-200 bg-white">
      <div className="flex items-center justify-between border-b border-slate-100 px-5 py-3">
        <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-slate-400">
          {t("ai_assistant.output_blocks.product_recs")}
        </p>
        <span className="text-xs text-slate-400">{t("ai_assistant.output_blocks.drivers_count", { count: items.length })}</span>
      </div>
      <div className="divide-y divide-slate-100">
        {items.map((rec) => (
          <div key={rec.name} className="flex flex-col gap-3 px-5 py-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">{rec.name}</div>
                <div className="text-xs text-slate-500">{rec.value_proposition}</div>
              </div>
              <span className="rounded-full border border-slate-200 bg-white px-2 py-1 text-[11px] font-semibold text-slate-500">
                {rec.priority.toUpperCase()}
              </span>
            </div>
            <div className="grid gap-3 text-xs text-slate-500 sm:grid-cols-[1.1fr,1fr]">
              <div>
                <div className="font-semibold text-slate-600">{t("ai_assistant.output_blocks.target_segment")}</div>
                <div>{rec.target_segment}</div>
              </div>
              <div>
                <div className="font-semibold text-slate-600">{t("ai_assistant.output_blocks.pricing")}</div>
                <div>{rec.pricing_model} · {rec.pricing_hint}</div>
              </div>
            </div>
            <div className="grid gap-3 text-xs text-slate-500 sm:grid-cols-[1.1fr,1fr]">
              <div>
                <div className="font-semibold text-slate-600">{t("ai_assistant.output_blocks.capabilities")}</div>
                <div>{rec.required_capabilities.join(", ")}</div>
              </div>
              <div>
                <div className="font-semibold text-slate-600">{t("ai_assistant.output_blocks.expected_impact")}</div>
                <div>{rec.expected_impact}</div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default function AIAssistant() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()

  const [overview, setOverview] = useState<AIAssistantOverviewResponse | null>(null)
  const [overviewLoading, setOverviewLoading] = useState(true)
  const [overviewError, setOverviewError] = useState<string | null>(null)
  const [runs, setRuns] = useState<AIAssistantRunsResponse | null>(null)
  const [activeRun, setActiveRun] = useState<AIAssistantRunDetail | null>(null)
  const [insightStatus, setInsightStatus] = useState<InsightStatus>("idle")
  const [runError, setRunError] = useState<string | null>(null)
  const [creatingWorkflow, setCreatingWorkflow] = useState(false)

  const [customer, setCustomer] = useState("")
  const [timeRange, setTimeRange] = useState("")
  const [intent, setIntent] = useState("")
  const [prompt, setPrompt] = useState("")

  const pollingRef = useRef<number | null>(null)

  const timeRanges = overview?.workspace.time_ranges ?? []
  const intentOptions = overview?.workspace.intents ?? []

  const intentLabel = intentOptions.find((opt) => opt.value === intent)?.label ?? intent
  const rangeLabel = timeRanges.find((opt) => opt.value === timeRange)?.label ?? timeRange

  const stopPolling = () => {
    if (pollingRef.current != null) {
      window.clearTimeout(pollingRef.current)
      pollingRef.current = null
    }
  }

  const resolveInsightStatus = (run?: AIAssistantRunDetail | null): InsightStatus => {
    if (!run) return "idle"
    if (run.status.code === "queued" || run.status.code === "running") return "loading"
    if (run.status.code === "failed") return "error"
    if (run.status.code === "done") return "ready"
    return "idle"
  }

  const loadRun = useCallback(async (runId: string) => {
    try {
      const resp = await api.aiAssistant.getRun(runId)
      setActiveRun(resp.run)
      const status = resolveInsightStatus(resp.run)
      setInsightStatus(status)
      if (status === "error") {
        setRunError(resp.run.error?.message ?? "Failed to generate insight")
      } else {
        setRunError(null)
      }
      return resp.run
    } catch (err) {
      setInsightStatus("error")
      setRunError(err instanceof Error ? err.message : "Failed to load run")
      return null
    }
  }, [])

  const pollRun = useCallback((runId: string, attempt = 0) => {
    stopPolling()
    pollingRef.current = window.setTimeout(async () => {
      const run = await loadRun(runId)
      if (!run) return
      if (run.status.code === "queued" || run.status.code === "running") {
        if (attempt < 10) {
          pollRun(runId, attempt + 1)
        }
      }
    }, attempt === 0 ? 0 : 800)
  }, [loadRun])

  const refreshRuns = useCallback(async () => {
    try {
      const resp = await api.aiAssistant.listRuns({ page_size: 5 })
      setRuns(resp)
    } catch {
      // silent for now
    }
  }, [])

  const loadOverview = useCallback(async () => {
    try {
      setOverviewLoading(true)
      setOverviewError(null)
      const resp = await api.aiAssistant.overview()
      setOverview(resp)
      setRuns(resp.runs)

      const defaultIntent = resp.workspace.intents[0]?.value ?? ""
      const defaultRange = resp.workspace.time_ranges[0]?.value ?? ""
      setIntent(defaultIntent)
      setTimeRange(defaultRange)
      setPrompt(resp.workspace.default_prompt || "")

      if (resp.active_run_id) {
        pollRun(resp.active_run_id)
      } else {
        setActiveRun(null)
        setInsightStatus("idle")
      }
    } catch (err) {
      setOverviewError(err instanceof Error ? err.message : "Failed to load")
    } finally {
      setOverviewLoading(false)
    }
  }, [pollRun])

  useEffect(() => {
    void loadOverview()
    return () => stopPolling()
  }, [loadOverview])

  const handleIntentChange = (value: string) => {
    setIntent(value)
  }

  const handleRun = async () => {
    if (!intent || !timeRange || !prompt) return
    setInsightStatus("loading")
    setRunError(null)
    try {
      const resp = await api.aiAssistant.createRun({
        customer_ref: customer,
        time_range: timeRange,
        intent,
        prompt
      })
      setActiveRun(resp.run)
      setInsightStatus(resolveInsightStatus(resp.run))
      void refreshRuns()
      pollRun(resp.run.id)
    } catch (err) {
      setInsightStatus("error")
      setRunError(err instanceof Error ? err.message : "Failed to run analysis")
    }
  }

  const handleReset = () => {
    setCustomer("")
    setPrompt(overview?.workspace.default_prompt ?? "")
  }

  const handleAction = (action: AIAssistantAction) => {
    const target = orgPath(action.path)
    navigate(target)
  }

  const handleCreateWorkflow = async () => {
    if (!activeRun?.insight) return
    const primaryRec = activeRun.insight.product_recommendations?.[0]
    const actions = (activeRun.insight.actions ?? []).map((action) => {
      const type = mapWorkflowActionType(action.key)
      const payload: Record<string, unknown> = { key: action.key, path: action.path }
      if (primaryRec && type === "create_product") {
        payload.name = primaryRec.name
        payload.code = toCode(primaryRec.name)
        payload.description = primaryRec.value_proposition
      }
      if (primaryRec && type === "create_plan") {
        const planName = `${primaryRec.name} Plan`
        payload.name = planName
        payload.code = toCode(planName)
        payload.description = primaryRec.pricing_model
      }
      if (primaryRec && type === "create_meter") {
        const meterName = `${primaryRec.name} Usage`
        payload.name = meterName
        payload.code = toCode(meterName)
        payload.aggregation = "sum"
        payload.unit = "event"
      }
      if (primaryRec && type === "create_feature") {
        const featureName = `${primaryRec.name} Feature`
        payload.name = featureName
        payload.code = toCode(featureName)
        payload.feature_type = "boolean"
      }
      return {
        type,
        label: action.label,
        payload
      }
    })
    if (actions.length === 0) return
    try {
      setCreatingWorkflow(true)
      const title = `${intentLabel || t("ai_assistant.workflow.default_title")} workflow`
      await api.aiWorkflows.create({
        title,
        summary: activeRun.insight.summary.headline,
        intent: activeRun.intent,
        source_run_id: activeRun.id,
        actions
      })
      toast.success(t("ai_assistant.workflow.created"))
      navigate(orgPath("/ai-workflows"))
    } catch (err) {
      toast.error(t("ai_assistant.workflow.failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setCreatingWorkflow(false)
    }
  }

  const handleSelectRun = (runId: string) => {
    setInsightStatus("loading")
    pollRun(runId)
  }

  const summaryCards = overview?.summary_cards ?? []
  const guardrails = overview?.guardrails.items ?? []
  const signals = overview?.signals.items ?? []
  const showSignals = intent === "churn"

  const toneColors: Record<string, string> = {
    info: "hsl(var(--status-info))",
    warning: "hsl(var(--status-warning))",
    danger: "hsl(var(--status-danger))",
    success: "hsl(var(--status-success))",
    neutral: "hsl(var(--accent-primary))",
    muted: "hsl(var(--muted-foreground))"
  }

  return (
    <div className="page-content ai-assistant-shell">
      <PageHeader
        icon={<IconSpark />}
        title={t("ai_assistant.header.title")}
        description={t("ai_assistant.header.description")}
        actions={(
          <div className="page-header-actions">
            <Button variant="outline" size="sm" onClick={handleReset}>{t("common.reset")}</Button>
            <Button size="sm" onClick={handleRun}>{t("ai_assistant.actions.run")}</Button>
          </div>
        )}
      />

      {overviewLoading ? (
        <div className="panel">
          <div className="panel-body" style={{ color: "var(--ink-2)" }}>{t("ai_assistant.loading")}</div>
        </div>
      ) : null}
      {overviewError ? (
        <div className="panel">
          <div className="panel-body" style={{ color: "var(--status-danger-text)" }}>{overviewError}</div>
        </div>
      ) : null}

      <div className="stat-grid">
        {summaryCards.map((card) => (
          <StatCard
            key={card.id}
            label={card.label}
            value={card.value}
            sub={card.delta ? `${card.sub} · ${card.delta}` : card.sub}
            accentColor={toneColors[card.tone] ?? "hsl(var(--accent-primary))"}
          />
        ))}
      </div>

      <div className="ai-assistant-grid">
        <div className="ai-assistant-main">
          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="panel-title">{t("ai_assistant.workspace.title")}</p>
                <p className="panel-desc">{t("ai_assistant.workspace.description")}</p>
              </div>
              <div className="ai-assistant-badges">
                {overview?.workspace.masking_enabled ? (
                  <span className="badge badge-muted">{t("ai_assistant.badges.masking")}</span>
                ) : null}
                <span className="badge badge-muted">{t("ai_assistant.badges.provider")}</span>
                <span className="badge badge-muted">{t("ai_assistant.badges.scope")}</span>
              </div>
            </div>
            <div className="panel-body">
              <div className="action-fields">
                <div className="action-field">
                  <Label className="action-label">{t("ai_assistant.fields.customer")}</Label>
                  <Input
                    className="action-input"
                    value={customer}
                    onChange={(event) => setCustomer(event.target.value)}
                    placeholder={overview?.workspace.customer_placeholder ?? t("ai_assistant.fields.customer_placeholder")}
                  />
                </div>
                <div className="action-field">
                  <Label className="action-label">{t("ai_assistant.fields.time_range")}</Label>
                  <Select value={timeRange} onValueChange={setTimeRange}>
                    <SelectTrigger className="action-select">
                      <SelectValue placeholder={t("common.select_placeholder")} />
                    </SelectTrigger>
                    <SelectContent>
                      {timeRanges.map((range) => (
                        <SelectItem key={range.value} value={range.value}>{range.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="action-field">
                  <Label className="action-label">{t("ai_assistant.fields.intent")}</Label>
                  <Select value={intent} onValueChange={handleIntentChange}>
                    <SelectTrigger className="action-select">
                      <SelectValue placeholder={t("common.select_placeholder")} />
                    </SelectTrigger>
                    <SelectContent>
                      {intentOptions.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="ai-assistant-tabs">
                <Tabs value={intent} onValueChange={handleIntentChange}>
                  <TabsList>
                    {intentOptions.map((opt) => (
                      <TabsTrigger key={opt.value} value={opt.value}>{opt.label}</TabsTrigger>
                    ))}
                  </TabsList>
                  <TabsContent value={intent}>
                    <div className="ai-assistant-hint">
                      <span className="badge badge-muted">{t("ai_assistant.prompt_hint")}</span>
                      <span className="muted">{t("ai_assistant.prompt_detail", { intent: intentLabel, range: rangeLabel })}</span>
                    </div>
                    <div className="action-field" style={{ marginTop: 12 }}>
                      <Label className="action-label">{t("ai_assistant.fields.prompt")}</Label>
                      <Textarea
                        className="action-input ai-assistant-textarea"
                        value={prompt}
                        onChange={(event) => setPrompt(event.target.value)}
                        placeholder={overview?.workspace.prompt_placeholder ?? t("ai_assistant.fields.prompt_placeholder")}
                      />
                    </div>
                    <div className="ai-assistant-actions">
                      <Button onClick={handleRun}>{t("ai_assistant.actions.run")}</Button>
                      <Button variant="secondary" onClick={handleReset}>{t("common.reset")}</Button>
                    </div>
                  </TabsContent>
                </Tabs>
              </div>
            </div>
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="panel-title">{t("ai_assistant.output.title")}</p>
                <p className="panel-desc">{t("ai_assistant.output.description")}</p>
              </div>
              <span className="badge badge-muted">{intentLabel}</span>
            </div>
            <div className="panel-body">
              <div className="flex flex-col gap-4">
                <div className="flex flex-wrap items-center justify-between gap-3 text-xs text-slate-500">
                  <div>
                    {activeRun?.updated_at
                      ? `${t("ai_assistant.output.last_run")}: ${formatDateTime(activeRun.updated_at)}`
                      : t("ai_assistant.output.placeholder")}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="rounded-full border border-slate-200 px-2 py-1">{activeRun?.customer_label || t("common.empty_dash")}</span>
                    <span className="rounded-full border border-slate-200 px-2 py-1">{rangeLabel || t("common.empty_dash")}</span>
                  </div>
                </div>

                {insightStatus === "loading" ? <AIOutputSkeleton /> : null}
                {insightStatus === "error" ? <AIOutputError message={runError ?? undefined} onRetry={handleRun} /> : null}
                {insightStatus === "idle" ? <AIOutputEmpty onRun={handleRun} /> : null}

                {insightStatus === "ready" && activeRun?.insight ? (
                  <div className="flex flex-col gap-4">
                    <AISummaryCard
                      headline={activeRun.insight.summary.headline}
                      metric={activeRun.insight.summary.metric}
                      metricNote={activeRun.insight.summary.metric_note}
                    />
                    <AIPlanRecommendation rec={activeRun.insight.plan_recommendation} />
                    <AISnapshotCard snapshot={activeRun.insight.snapshot} />
                    <AIDriverList drivers={activeRun.insight.drivers ?? []} />
                    <AIProductRecommendations items={activeRun.insight.product_recommendations} />
                    <AIAnomalyBlock anomalies={activeRun.insight.anomalies} />
                    {activeRun.insight.proration ? (
                      <div className="rounded-xl border border-slate-200 bg-white p-4">
                        <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">{t("ai_assistant.output_blocks.proration")}</div>
                        <div className="mt-2 text-sm font-semibold text-slate-900">{activeRun.insight.proration.title}</div>
                        <div className="text-xs text-slate-500">{activeRun.insight.proration.detail}</div>
                      </div>
                    ) : null}
                    <AIConfidenceRow confidence={activeRun.insight.confidence} dataQuality={activeRun.insight.data_quality} />
                    <AIActionBar actions={activeRun.insight.actions ?? []} onAction={handleAction} />
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                        {t("ai_assistant.workflow.title")}
                      </div>
                      <div className="mt-2 text-xs text-slate-500">{t("ai_assistant.workflow.description")}</div>
                      <div className="mt-3">
                        <Button size="sm" variant="outline" onClick={handleCreateWorkflow} disabled={creatingWorkflow}>
                          {t("ai_assistant.workflow.create")}
                        </Button>
                      </div>
                    </div>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        </div>

        <div className="ai-assistant-sidebar">
          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="panel-title">{t("ai_assistant.guardrails.title")}</p>
                <p className="panel-desc">{t("ai_assistant.guardrails.description")}</p>
              </div>
            </div>
            <div className="panel-body">
              <div className="ai-guardrail-list">
                {guardrails.map((item) => (
                  <div key={item.id} className="ai-guardrail-item">
                    <span className="ai-guardrail-dot" />
                    <span>{item.title}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="panel-title">{t("ai_assistant.how.title")}</p>
                <p className="panel-desc">{t("ai_assistant.how.description")}</p>
              </div>
            </div>
            <div className="panel-body">
              <ol className="ai-how-list">
                <li>{t("ai_assistant.how.step_1")}</li>
                <li>{t("ai_assistant.how.step_2")}</li>
                <li>{t("ai_assistant.how.step_3")}</li>
                <li>{t("ai_assistant.how.step_4")}</li>
              </ol>
            </div>
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="panel-title">{t("ai_assistant.recent.title")}</p>
                <p className="panel-desc">{t("ai_assistant.recent.description")}</p>
              </div>
              <span className="badge badge-info">{runs?.runs.length ?? 0}</span>
            </div>
            <div className="panel-body" style={{ padding: 0 }}>
              <div className="list">
                {runs?.runs.map((run) => (
                  <button
                    key={run.id}
                    type="button"
                    className="list-item ai-run-item"
                    onClick={() => handleSelectRun(run.id)}
                  >
                    <div>
                      <div style={{ fontWeight: 600, color: "var(--ink)" }}>{run.title}</div>
                      <div className="ai-recent-meta">{run.subtitle} · {formatDateTime(run.created_at)}</div>
                    </div>
                    <span className={`badge badge-${run.status.tone}`}>{run.status.label}</span>
                  </button>
                ))}
                {!runs?.runs.length ? (
                  <div className="list-item">
                    <div>
                      <div style={{ fontWeight: 600, color: "var(--ink)" }}>{t("ai_assistant.runs.empty_title")}</div>
                      <div className="ai-recent-meta">{t("ai_assistant.runs.empty_desc")}</div>
                    </div>
                  </div>
                ) : null}
              </div>
            </div>
          </div>

          {showSignals ? (
            <div className="panel">
              <div className="panel-header">
                <div>
                  <p className="panel-title">{t("ai_assistant.signals.title")}</p>
                  <p className="panel-desc">{t("ai_assistant.signals.description")}</p>
                </div>
              </div>
              <div className="panel-body">
                <div className="ai-signal-list">
                  {signals.map((signal) => (
                    <div key={signal.id} className="ai-signal">
                      <div>
                        <div style={{ fontWeight: 600, color: "var(--ink)" }}>{signal.title}</div>
                        <div className="ai-recent-meta">{signal.detail}</div>
                      </div>
                      <span className={`badge badge-${signal.severity}`}>{t("ai_assistant.signals.badge")}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
