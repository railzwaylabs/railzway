import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { api } from "../lib/api"
import { formatDateTime, rfc3339Hint } from "../lib/display"
import type { TestClock } from "../lib/types"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import { useOrgPath } from "../lib/org"

function IconClock() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="10" cy="10" r="7" />
      <path d="M10 6v4l3 2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export default function TestClockPage() {
  const { t } = useTranslation()
  const orgPath = useOrgPath()
  const [clock, setClock] = useState<TestClock | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [advanceBy, setAdvanceBy] = useState("3600")
  const [form, setForm] = useState({ currentTime: "", status: "active" })

  const loadClock = useCallback(async () => {
    try {
      setLoading(true)
      const resp = await api.testClock.get()
      if (resp.clock) {
        setClock(resp.clock)
        setForm({
          currentTime: resp.clock.current_time,
          status: resp.clock.status
        })
      } else {
        setClock(null)
      }
    } catch (err) {
      toast.error(t("test_clock.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => { void loadClock() }, [loadClock])

  const validation = useMemo(() => {
    const errors: string[] = []
    if (!form.currentTime.trim()) {
      errors.push(t("test_clock.validation.current_time_required"))
    }
    return errors
  }, [form, t])

  const handleSave = useCallback(async () => {
    try {
      setSaving(true)
      const resp = await api.testClock.upsert({
        current_time: form.currentTime.trim(),
        status: form.status.trim() || undefined
      })
      setClock(resp)
      toast.success(t("test_clock.toast.saved"), resp.current_time)
    } catch (err) {
      toast.error(t("test_clock.toast.save_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [form, t])

  const handleAdvance = useCallback(async () => {
    const seconds = Number.parseInt(advanceBy, 10)
    if (!Number.isFinite(seconds) || seconds <= 0) {
      toast.error(t("test_clock.toast.advance_invalid"))
      return
    }
    try {
      setSaving(true)
      const resp = await api.testClock.advance({ advance_by_seconds: seconds })
      setClock(resp)
      setForm((prev) => ({ ...prev, currentTime: resp.current_time, status: resp.status }))
      toast.success(t("test_clock.toast.advanced"), `+${seconds}s`)
    } catch (err) {
      toast.error(t("test_clock.toast.advance_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [advanceBy, t])

  const handlePause = useCallback(async () => {
    try {
      setSaving(true)
      const resp = await api.testClock.pause()
      setClock(resp)
      setForm((prev) => ({ ...prev, status: resp.status }))
      toast.success(t("test_clock.toast.paused"))
    } catch (err) {
      toast.error(t("test_clock.toast.pause_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [t])

  const handleResume = useCallback(async () => {
    try {
      setSaving(true)
      const resp = await api.testClock.resume()
      setClock(resp)
      setForm((prev) => ({ ...prev, status: resp.status }))
      toast.success(t("test_clock.toast.resumed"))
    } catch (err) {
      toast.error(t("test_clock.toast.resume_failed"), err instanceof Error ? err.message : undefined)
    } finally {
      setSaving(false)
    }
  }, [t])

  return (
    <div className="page-content">
      <PageHeader
        icon={<IconClock />}
        title={t("test_clock.header.title")}
        description={t("test_clock.header.description")}
        actions={(
          <Button variant="outline" asChild data-testid="testclock-audit-logs">
            <Link to={orgPath("/audit-logs")}>{t("test_clock.actions.view_audit")}</Link>
          </Button>
        )}
      />

      <div className="panel" style={{ maxWidth: 820 }}>
        <div className="action-section" style={{ border: "none" }}>
          <div className="action-section-title">{t("test_clock.current.title")}</div>
          {loading ? (
            <div className="muted">{t("test_clock.current.loading")}</div>
          ) : clock ? (
            <div className="flag-toolbar">
              <div>
                <div className="muted">{t("test_clock.current.status")}</div>
                <strong>{clock.status}</strong>
              </div>
              <div>
                <div className="muted">{t("test_clock.current.time")}</div>
                <strong>{formatDateTime(clock.current_time)}</strong>
              </div>
              <div>
                <div className="muted">{t("test_clock.current.updated")}</div>
                <strong>{formatDateTime(clock.updated_at)}</strong>
              </div>
            </div>
          ) : (
            <div className="muted">{t("test_clock.current.empty")}</div>
          )}
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("test_clock.update.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">
                {t("test_clock.update.current_time")} <HelpHint text={rfc3339Hint} />
              </label>
              <Input
                className="action-input"
                type="datetime-local"
                value={form.currentTime}
                onChange={(e) => setForm((prev) => ({ ...prev, currentTime: e.target.value }))}
                data-testid="testclock-current-time"
              />
            </div>
            <div className="action-field">
              <label className="action-label">{t("test_clock.update.status")}</label>
              <Select value={form.status} onValueChange={(value) => setForm((prev) => ({ ...prev, status: value }))}>
                <SelectTrigger className="action-select" data-testid="testclock-status">
                  <SelectValue placeholder={t("test_clock.update.status_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">{t("test_clock.status.active")}</SelectItem>
                  <SelectItem value="paused">{t("test_clock.status.paused")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          {validation.length > 0 ? <div className="inline-error">{validation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="default" disabled={saving || validation.length > 0} onClick={handleSave} data-testid="testclock-save">
              {saving ? t("common.saving") : t("test_clock.update.save")}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("test_clock.advance.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("test_clock.advance.label")}</label>
              <input
                className="action-input"
                value={advanceBy}
                onChange={(e) => setAdvanceBy(e.target.value)}
                data-testid="testclock-advance-seconds"
              />
            </div>
          </div>
          <div className="action-buttons">
            <Button variant="secondary" disabled={saving} onClick={handleAdvance} data-testid="testclock-advance">
              {saving ? t("test_clock.advance.advancing") : t("test_clock.advance.cta")}
            </Button>
            <Button variant="outline" disabled={saving} onClick={handlePause} data-testid="testclock-pause">
              {t("test_clock.actions.pause")}
            </Button>
            <Button variant="outline" disabled={saving} onClick={handleResume} data-testid="testclock-resume">
              {t("test_clock.actions.resume")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
