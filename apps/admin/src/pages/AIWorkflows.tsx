import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import PageHeader from "../components/PageHeader"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { toast } from "../components/Toast"
import { api } from "../lib/api"
import { formatDateTime } from "../lib/display"
import { useOrgPath } from "../lib/org"
import type {
  AIWorkflowAction,
  AIWorkflowDetail,
  AIWorkflowListItem,
  AIWorkflowListResponse
} from "../lib/types"

type ActionStatus = AIWorkflowAction["status"]

function IconWorkflow() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="3" y="3" width="14" height="14" rx="2" />
      <path d="M6 7h8M6 10h8" strokeLinecap="round"/>
      <path d="M7 14l2-2 2 2" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

const actionStatusTone: Record<ActionStatus, string> = {
  pending: "muted",
  running: "info",
  done: "success",
  failed: "danger"
}

function resolveActionMeta(action: AIWorkflowAction) {
  const payload = action.payload ?? {}
  if (typeof payload.path === "string" && payload.path.trim() !== "") {
    return payload.path
  }
  return ""
}

export default function AIWorkflows() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()

  const [workflows, setWorkflows] = useState<AIWorkflowListResponse | null>(null)
  const [activeWorkflow, setActiveWorkflow] = useState<AIWorkflowDetail | null>(null)
  const [activeId, setActiveId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [note, setNote] = useState("")
  const [mutating, setMutating] = useState(false)

  const pollRef = useRef<number | null>(null)

  const stopPolling = () => {
    if (pollRef.current != null) {
      window.clearTimeout(pollRef.current)
      pollRef.current = null
    }
  }

  const loadWorkflows = useCallback(async (keepActive = true) => {
    try {
      setLoading(true)
      setError(null)
      const resp = await api.aiWorkflows.list({ page_size: 20 })
      setWorkflows(resp)
      const first = resp.workflows[0]
      if (!keepActive || !activeId) {
        if (first) {
          setActiveId(first.id)
        } else {
          setActiveId(null)
          setActiveWorkflow(null)
        }
      }
      return resp
    } catch (err) {
      setError(err instanceof Error ? err.message : t("ai_workflows.errors.load_list"))
      return null
    } finally {
      setLoading(false)
    }
  }, [activeId, t])

  const loadWorkflow = useCallback(async (workflowId: string) => {
    try {
      setDetailLoading(true)
      const resp = await api.aiWorkflows.get(workflowId)
      setActiveWorkflow(resp.workflow)
      return resp.workflow
    } catch (err) {
      toast.error(t("ai_workflows.errors.load_detail"), err instanceof Error ? err.message : undefined)
      return null
    } finally {
      setDetailLoading(false)
    }
  }, [t])

  const pollWorkflow = useCallback((workflowId: string, attempt = 0) => {
    stopPolling()
    pollRef.current = window.setTimeout(async () => {
      const next = await loadWorkflow(workflowId)
      if (!next) return
      if (next.status.code === "executing" && attempt < 12) {
        pollWorkflow(workflowId, attempt + 1)
      }
    }, attempt === 0 ? 0 : 800)
  }, [loadWorkflow])

  useEffect(() => {
    void loadWorkflows(false)
    return () => stopPolling()
  }, [loadWorkflows])

  useEffect(() => {
    if (!activeId) return
    void loadWorkflow(activeId)
  }, [activeId, loadWorkflow])

  useEffect(() => {
    if (activeWorkflow?.status.code === "executing" && activeWorkflow.id) {
      pollWorkflow(activeWorkflow.id)
    }
  }, [activeWorkflow?.status.code, activeWorkflow?.id, pollWorkflow])

  const handleSelect = (workflow: AIWorkflowListItem) => {
    setActiveId(workflow.id)
  }

  const handleApprove = async (status: "approved" | "rejected") => {
    if (!activeWorkflow) return
    try {
      setMutating(true)
      const resp = await api.aiWorkflows.approve(activeWorkflow.id, { status, note: note.trim() || undefined })
      setActiveWorkflow(resp.workflow)
      setNote("")
      await loadWorkflows()
      toast.success(t("ai_workflows.toast.approval_saved"))
    } catch (err) {
      toast.error(t("ai_workflows.toast.approval_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setMutating(false)
    }
  }

  const handleExecute = async () => {
    if (!activeWorkflow) return
    try {
      setMutating(true)
      const resp = await api.aiWorkflows.execute(activeWorkflow.id)
      setActiveWorkflow(resp.workflow)
      await loadWorkflows()
      toast.success(t("ai_workflows.toast.execution_started"))
      if (resp.workflow.status.code === "executing") {
        pollWorkflow(resp.workflow.id)
      }
    } catch (err) {
      toast.error(t("ai_workflows.toast.execution_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setMutating(false)
    }
  }

  const listItems = workflows?.workflows ?? []
  const statusTone = activeWorkflow?.status.tone ?? "muted"

  const workflowStatusLabel = useMemo(() => {
    if (!activeWorkflow) return t("ai_workflows.detail.status_placeholder")
    return activeWorkflow.status.label
  }, [activeWorkflow, t])

  const statusHint = useMemo(() => {
    if (!activeWorkflow) return t("ai_workflows.detail.hint_idle")
    switch (activeWorkflow.status.code) {
      case "pending_approval":
        return t("ai_workflows.detail.hint_pending")
      case "approved":
        return t("ai_workflows.detail.hint_approved")
      case "executing":
        return t("ai_workflows.detail.hint_executing")
      case "completed":
        return t("ai_workflows.detail.hint_completed")
      case "failed":
        return t("ai_workflows.detail.hint_failed")
      default:
        return t("ai_workflows.detail.hint_idle")
    }
  }, [activeWorkflow, t])

  const navigateToAssistant = () => {
    navigate(orgPath("/ai-assistant"))
  }

  const actionStatusLabel = (status: ActionStatus) => {
    const map: Record<ActionStatus, string> = {
      pending: t("ai_workflows.action_status.pending"),
      running: t("ai_workflows.action_status.running"),
      done: t("ai_workflows.action_status.done"),
      failed: t("ai_workflows.action_status.failed")
    }
    return map[status]
  }

  return (
    <div className="page-content ai-workflow-shell">
      <PageHeader
        icon={<IconWorkflow />}
        title={t("ai_workflows.header.title")}
        description={t("ai_workflows.header.description")}
        actions={(
          <div className="page-header-actions">
            <Button variant="outline" size="sm" onClick={() => loadWorkflows()}>{t("ai_workflows.actions.refresh")}</Button>
            <Button size="sm" onClick={navigateToAssistant}>{t("ai_workflows.actions.open_assistant")}</Button>
          </div>
        )}
      />

      {error ? (
        <div className="panel">
          <div className="panel-body" style={{ color: "var(--status-danger-text)" }}>{error}</div>
        </div>
      ) : null}

      <div className="ai-workflow-grid">
        <div className="panel">
          <div className="panel-header">
            <div>
              <p className="panel-title">{t("ai_workflows.list.title")}</p>
              <p className="panel-desc">{t("ai_workflows.list.description")}</p>
            </div>
            <span className="badge badge-muted">{listItems.length}</span>
          </div>
          <div className="panel-body" style={{ padding: 0 }}>
            {loading ? (
              <div className="panel-body" style={{ padding: 20, color: "var(--ink-2)" }}>{t("ai_workflows.loading")}</div>
            ) : null}
            {!loading && listItems.length === 0 ? (
              <div className="panel-body">
                <div className="text-sm" style={{ fontWeight: 600, color: "var(--ink)" }}>{t("ai_workflows.list.empty_title")}</div>
                <div className="text-xs muted">{t("ai_workflows.list.empty_desc")}</div>
                <Button size="sm" style={{ marginTop: 12 }} onClick={navigateToAssistant}>
                  {t("ai_workflows.actions.open_assistant")}
                </Button>
              </div>
            ) : null}
            {listItems.length > 0 ? (
              <div className="list ai-workflow-list">
                {listItems.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    className="list-item ai-workflow-list-item"
                    onClick={() => handleSelect(item)}
                  >
                    <div>
                      <div style={{ fontWeight: 600, color: "var(--ink)" }}>{item.title}</div>
                      <div className="ai-recent-meta">{item.summary || t("ai_workflows.list.summary_placeholder")}</div>
                    </div>
                    <span className={`badge badge-${item.status.tone}`}>{item.status.label}</span>
                  </button>
                ))}
              </div>
            ) : null}
          </div>
        </div>

        <div className="panel">
          <div className="panel-header">
            <div>
              <p className="panel-title">{t("ai_workflows.detail.title")}</p>
              <p className="panel-desc">{t("ai_workflows.detail.description")}</p>
            </div>
            <span className={`badge badge-${statusTone}`}>{workflowStatusLabel}</span>
          </div>
          <div className="panel-body">
            {detailLoading || !activeWorkflow ? (
              <div className="text-sm muted">{detailLoading ? t("ai_workflows.loading_detail") : t("ai_workflows.detail.empty")}</div>
            ) : (
              <div className="ai-workflow-detail">
                <div className="ai-workflow-hint">{statusHint}</div>
                <div>
                  <div className="text-sm" style={{ fontWeight: 600, color: "var(--ink)" }}>{activeWorkflow.title}</div>
                  <div className="text-xs muted">{activeWorkflow.summary || t("ai_workflows.detail.summary_placeholder")}</div>
                </div>

                <div className="ai-workflow-meta">
                  <div className="ai-workflow-meta-item">
                    <span className="ai-workflow-meta-label">{t("ai_workflows.detail.intent")}</span>
                    <span>{activeWorkflow.intent || t("common.empty_dash")}</span>
                  </div>
                  <div className="ai-workflow-meta-item">
                    <span className="ai-workflow-meta-label">{t("ai_workflows.detail.created")}</span>
                    <span>{formatDateTime(activeWorkflow.created_at)}</span>
                  </div>
                  <div className="ai-workflow-meta-item">
                    <span className="ai-workflow-meta-label">{t("ai_workflows.detail.updated")}</span>
                    <span>{formatDateTime(activeWorkflow.updated_at)}</span>
                  </div>
                  {activeWorkflow.source_run_id ? (
                    <div className="ai-workflow-meta-item">
                      <span className="ai-workflow-meta-label">{t("ai_workflows.detail.source_run")}</span>
                      <span>{activeWorkflow.source_run_id.slice(0, 8)}…</span>
                    </div>
                  ) : null}
                </div>

                <div className="ai-workflow-section">
                  <div className="ai-workflow-section-title">{t("ai_workflows.detail.actions_title")}</div>
                  <div className="ai-workflow-action-list">
                    {activeWorkflow.actions.map((action) => (
                      <div key={action.id} className="ai-workflow-action">
                        <div>
                          <div className="ai-workflow-action-title">{action.label}</div>
                          <div className="ai-workflow-action-meta">
                            {action.type}{resolveActionMeta(action) ? ` · ${resolveActionMeta(action)}` : ""}
                          </div>
                        </div>
                        <span className={`badge badge-${actionStatusTone[action.status]}`}>{actionStatusLabel(action.status)}</span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="ai-workflow-section">
                  <div className="ai-workflow-section-title">{t("ai_workflows.detail.approvals_title")}</div>
                  {activeWorkflow.approvals.length === 0 ? (
                    <div className="text-xs muted">{t("ai_workflows.detail.no_approvals")}</div>
                  ) : (
                    <div className="ai-workflow-approvals">
                      {activeWorkflow.approvals.map((approval) => (
                        <div key={approval.id} className="ai-workflow-approval">
                          <div>
                            <div className="ai-workflow-action-title">{approval.actor_id}</div>
                            <div className="ai-workflow-action-meta">{approval.note || t("ai_workflows.detail.no_note")}</div>
                          </div>
                          <span className={`badge badge-${approval.status === "approved" ? "success" : "danger"}`}>
                            {approval.status.toUpperCase()}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {activeWorkflow.status.code === "pending_approval" ? (
                  <div className="ai-workflow-approval-form">
                    <Input
                      value={note}
                      onChange={(event) => setNote(event.target.value)}
                      placeholder={t("ai_workflows.detail.note_placeholder")}
                    />
                    <div className="ai-workflow-actions">
                      <Button size="sm" variant="outline" onClick={() => handleApprove("rejected")} disabled={mutating}>
                        {t("ai_workflows.actions.reject")}
                      </Button>
                      <Button size="sm" onClick={() => handleApprove("approved")} disabled={mutating}>
                        {t("ai_workflows.actions.approve")}
                      </Button>
                    </div>
                  </div>
                ) : null}

                {activeWorkflow.status.code === "approved" ? (
                  <div className="ai-workflow-execute">
                    <div className="text-xs muted">{t("ai_workflows.detail.execute_ready")}</div>
                    <Button size="sm" onClick={handleExecute} disabled={mutating}>
                      {t("ai_workflows.actions.execute")}
                    </Button>
                  </div>
                ) : null}

                {activeWorkflow.status.code === "executing" ? (
                  <div className="ai-workflow-executing">{t("ai_workflows.detail.executing")}</div>
                ) : null}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
