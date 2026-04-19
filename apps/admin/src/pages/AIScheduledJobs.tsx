import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../components/ui/button"
import { api } from "../lib/api"
import type { AIScheduledJob } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { formatDate, shortID } from "../lib/display"

function IconScheduler() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
      <line x1="16" y1="2" x2="16" y2="6" />
      <line x1="8" y1="2" x2="8" y2="6" />
      <line x1="3" y1="10" x2="21" y2="10" />
      <path d="M8 14h.01" />
      <path d="M12 14h.01" />
      <path d="M16 14h.01" />
      <path d="M8 18h.01" />
      <path d="M12 18h.01" />
      <path d="M16 18h.01" />
    </svg>
  )
}

export default function AIScheduledJobs() {
  const { t } = useTranslation()
  const [jobs, setJobs] = useState<AIScheduledJob[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)

  const loadJobs = useCallback(async () => {
    try {
      setLoading(true)
      const resp = await api.ai.listJobs()
      setJobs(resp.jobs)
      setTotal(resp.total)
    } catch (err) {
      toast.error(t("ai_scheduled_jobs.notifications.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadJobs()
  }, [loadJobs])

  const handleRetry = async (id: string) => {
    try {
      await api.ai.retryJob(id)
      toast.success(t("ai_scheduled_jobs.notifications.retry_success"))
      void loadJobs()
    } catch (err) {
      toast.error(t("ai_scheduled_jobs.notifications.retry_failed"), err instanceof Error ? err.message : undefined)
    }
  }

  const handleCancel = async (id: string) => {
    if (!confirm(t("ai_scheduled_jobs.confirmations.cancel"))) return
    try {
      await api.ai.cancelJob(id)
      toast.success(t("ai_scheduled_jobs.notifications.cancel_success"))
      void loadJobs()
    } catch (err) {
      toast.error(t("ai_scheduled_jobs.notifications.cancel_failed"), err instanceof Error ? err.message : undefined)
    }
  }

  const columns = [
    {
      key: "task_type",
      label: t("ai_scheduled_jobs.columns.task_type"),
      render: (r: AIScheduledJob) => (
        <div>
          <div className="font-semibold">{r.task_type}</div>
          <div className="text-xs text-slate-500 font-mono">{shortID(r.id)}</div>
        </div>
      ),
    },
    {
      key: "schedule",
      label: t("ai_scheduled_jobs.columns.schedule"),
      render: (r: AIScheduledJob) => (
        <div className="text-sm">
          {r.schedule_cron ? (
            <code className="bg-slate-100 px-1 rounded text-xs">{r.schedule_cron}</code>
          ) : (
            <span className="text-slate-400 italic">{t("ai_scheduled_jobs.schedule.one_off")}</span>
          )}
        </div>
      ),
    },
    {
      key: "status",
      label: t("ai_scheduled_jobs.columns.status"),
      render: (r: AIScheduledJob) => {
        let variant = "badge-secondary"
        if (r.status === "pending") variant = "badge-info"
        if (r.status === "running") variant = "badge-warning"
        if (r.status === "completed") variant = "badge-success"
        if (r.status === "failed") variant = "badge-danger"
        if (r.status === "cancelled") variant = "badge-secondary"
        return <span className={`badge ${variant}`}>{t(`ai_scheduled_jobs.status.${r.status}`)}</span>
      },
    },
    {
      key: "next_run",
      label: t("ai_scheduled_jobs.columns.next_run"),
      render: (r: AIScheduledJob) => (
        <div className="text-sm text-slate-600">
          {formatDate(r.next_run_at)}
        </div>
      ),
    },
    {
      key: "errors",
      label: t("ai_scheduled_jobs.columns.health"),
      render: (r: AIScheduledJob) => (
        <div className="text-sm">
          {r.error_count > 0 ? (
            <div className="text-red-600 flex flex-col">
              <span>{t("ai_scheduled_jobs.health.errors", { count: r.error_count })}</span>
              <span className="text-[10px] truncate max-w-[200px]" title={r.last_error}>
                {r.last_error}
              </span>
            </div>
          ) : (
            <span className="text-green-600">{t("ai_scheduled_jobs.health.healthy")}</span>
          )}
        </div>
      ),
    },
    {
      key: "actions",
      label: "",
      render: (r: AIScheduledJob) => (
        <div className="flex gap-2 justify-end">
          {(r.status === "failed" || r.status === "cancelled") && (
            <Button size="sm" variant="secondary" onClick={() => handleRetry(r.id)}>
              {t("common.retry")}
            </Button>
          )}
          {r.status !== "cancelled" && r.status !== "completed" && (
            <Button size="sm" variant="secondary" className="text-red-600 hover:text-red-700 hover:bg-red-50" onClick={() => handleCancel(r.id)}>
              {t("common.cancel")}
            </Button>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconScheduler />}
        title={t("ai_scheduled_jobs.title")}
        description={t("ai_scheduled_jobs.description")}
      />

      <DataTable
        columns={columns}
        data={jobs}
        loading={loading}
        emptyTitle={t("ai_scheduled_jobs.empty.title")}
        emptyDesc={t("ai_scheduled_jobs.empty.description")}
      />
    </div>
  )
}
