import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { api } from "../lib/api"
import { Badge } from "../components/ui/badge"
import { formatCurrency, formatNumber } from "../lib/format"
import { formatDateTime, shortID } from "../lib/display"
import type { DashboardSummary, ReconciliationMismatch, ReconciliationSummaryResponse } from "../lib/types"
import StatCard from "../components/StatCard"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"

function IconTrending() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 11L6 7l3 2 5-5" strokeLinecap="round" strokeLinejoin="round"/>
      <path d="M11 4h3v3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

function IconUsage() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 12L5 8l3 2 3-5 3 3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

function IconInvoice() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="1" width="12" height="14" rx="1.5"/>
      <path d="M5 5h6M5 8h6M5 11h3" strokeLinecap="round"/>
    </svg>
  )
}

function IconAlert() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M8 2L14.5 13H1.5L8 2z" strokeLinejoin="round"/>
      <path d="M8 6v3.5M8 11v.5" strokeLinecap="round"/>
    </svg>
  )
}

function IconDashboard() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="1" width="7" height="7" rx="2"/>
      <rect x="12" y="1" width="7" height="7" rx="2"/>
      <rect x="1" y="12" width="7" height="7" rx="2"/>
      <rect x="12" y="12" width="7" height="7" rx="2"/>
    </svg>
  )
}

export default function Dashboard() {
  const { t } = useTranslation()
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [reconciliation, setReconciliation] = useState<ReconciliationSummaryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const alertColumns = useMemo(
    () => [
      { key: "title", label: t("dashboard.alerts.columns.signal"), render: (row: { title: string; subtitle: string }) => (
        <div>
          <div style={{ fontWeight: 600, color: "var(--ink)" }}>{row.title}</div>
          <div className="muted">{row.subtitle}</div>
        </div>
      )},
      { key: "tag", label: t("dashboard.alerts.columns.tag"), width: "120px", render: (row: { tag: string }) => (
        <Badge variant="secondary">{row.tag}</Badge>
      )},
    ],
    [t]
  )

  const reconciliationColumns = useMemo(
    () => [
      { key: "action", label: t("dashboard.reconciliation.columns.type"), width: "180px", render: (row: ReconciliationMismatch) => {
        const label = row.action.includes("usage")
          ? t("dashboard.reconciliation.badges.usage")
          : row.action.includes("ledger")
            ? t("dashboard.reconciliation.badges.ledger")
            : row.action
        const badgeClass = row.action.includes("usage") ? "badge badge-danger" : row.action.includes("ledger") ? "badge badge-warning" : "badge badge-muted"
        return <span className={badgeClass}>{label}</span>
      }},
      { key: "invoice_id", label: t("dashboard.reconciliation.columns.invoice"), width: "160px", render: (row: ReconciliationMismatch) => (
        <span className="mono">{shortID(row.invoice_id)}</span>
      )},
      { key: "created_at", label: t("dashboard.reconciliation.columns.detected"), width: "180px", render: (row: ReconciliationMismatch) => (
        <span className="muted">{formatDateTime(row.created_at)}</span>
      )},
    ],
    [t]
  )

  useEffect(() => {
    const load = async () => {
      try {
        setLoading(true)
        setError(null)
        const data = await api.dashboard.summary()
        setSummary(data)
        try {
          const recon = await api.reconciliation.summary()
          setReconciliation(recon)
        } catch {
          setReconciliation(null)
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "failed_to_load")
      } finally {
        setLoading(false)
      }
    }
    void load()
  }, [])

  const alerts = summary?.alerts ?? []
  const reconTotal = reconciliation?.totalMismatches ?? 0
  const reconBadgeClass = reconTotal > 0 ? "badge badge-danger" : "badge badge-success"

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconDashboard />}
        title={t("dashboard.header.title")}
        description={t("dashboard.header.description")}
      />

      {error ? (
        <div className="panel">
          <div className="panel-body" style={{ display: "flex", alignItems: "center", gap: 10, color: "var(--status-danger-text)" }}>
            <IconAlert />
            <div>
              <strong>{t("dashboard.error.title")}</strong>
              <div className="muted" style={{ marginTop: 2 }}>{error}</div>
            </div>
          </div>
        </div>
      ) : null}

      {/* KPI Cards */}
      <div className="stat-grid">
        <StatCard
          label={t("dashboard.kpis.mrr.label")}
          value={loading ? <span className="skeleton skeleton-title" style={{ width: 100 }} /> : formatCurrency(summary?.mrr_cents ?? 0)}
          icon={<IconTrending />}
          sub={t("dashboard.kpis.mrr.sub")}
          accentColor="hsl(var(--status-success))"
        />
        <StatCard
          label={t("dashboard.kpis.usage.label")}
          value={loading ? <span className="skeleton skeleton-title" style={{ width: 80 }} /> : formatCurrency(summary?.usage_cents ?? 0)}
          icon={<IconUsage />}
          sub={t("dashboard.kpis.usage.sub")}
          accentColor="hsl(var(--accent-primary))"
        />
        <StatCard
          label={t("dashboard.kpis.open_invoices.label")}
          value={loading ? <span className="skeleton skeleton-title" style={{ width: 60 }} /> : formatNumber(summary?.open_invoices ?? 0)}
          icon={<IconInvoice />}
          sub={t("dashboard.kpis.open_invoices.sub")}
          accentColor="hsl(var(--status-warning))"
        />
        <StatCard
          label={t("dashboard.kpis.late_events.label")}
          value={loading ? <span className="skeleton skeleton-title" style={{ width: 60 }} /> : `${formatNumber(summary?.late_events ?? 0)}`}
          icon={<IconAlert />}
          sub={t("dashboard.kpis.late_events.sub")}
          accentColor="hsl(var(--status-error))"
        />
      </div>

      {reconciliation ? (
        <div className="panel" id="reconciliation">
          <div className="panel-header">
            <div>
              <p className="panel-title">{t("dashboard.reconciliation.title")}</p>
              <p className="panel-desc">{t("dashboard.reconciliation.description", { days: reconciliation.windowDays })}</p>
            </div>
            <span className={reconBadgeClass}>
              {reconTotal === 0 ? t("dashboard.reconciliation.badge_all_matched") : t("dashboard.reconciliation.badge_mismatch", { count: reconTotal })}
            </span>
          </div>
          <div className="panel-body">
            <div className="stat-grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))" }}>
              <StatCard
                label={t("dashboard.reconciliation.kpis.usage.label")}
                value={formatNumber(reconciliation.usageMismatches)}
                sub={t("dashboard.reconciliation.kpis.usage.sub")}
                accentColor="hsl(var(--status-danger))"
              />
              <StatCard
                label={t("dashboard.reconciliation.kpis.ledger.label")}
                value={formatNumber(reconciliation.ledgerMismatches)}
                sub={t("dashboard.reconciliation.kpis.ledger.sub")}
                accentColor="hsl(var(--status-warning))"
              />
              <StatCard
                label={t("dashboard.reconciliation.kpis.total.label")}
                value={formatNumber(reconciliation.totalMismatches)}
                sub={t("dashboard.reconciliation.kpis.total.sub")}
                accentColor={reconciliation.totalMismatches > 0 ? "hsl(var(--status-danger))" : "hsl(var(--status-success))"}
              />
            </div>
            <div style={{ marginTop: 16 }}>
              <DataTable
                columns={reconciliationColumns as Parameters<typeof DataTable>[0]["columns"]}
                data={reconciliation.latest as ReconciliationMismatch[]}
                loading={loading}
                emptyTitle={t("dashboard.reconciliation.empty_title")}
                emptyDesc={t("dashboard.reconciliation.empty_desc")}
                keyExtractor={(row) => `${row.invoice_id}-${row.created_at}`}
              />
            </div>
          </div>
        </div>
      ) : null}

      {/* Alerts Table */}
      <DataTable
        columns={alertColumns as Parameters<typeof DataTable>[0]["columns"]}
        data={alerts as Array<{ id?: string; title: string; subtitle: string; tag: string }>}
        loading={loading}
        emptyTitle={t("dashboard.alerts.empty_title")}
        emptyDesc={t("dashboard.alerts.empty_desc")}
        keyExtractor={(row) => `${row.title}-${row.subtitle}`}
      />
    </div>
  )
}
