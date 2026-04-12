import { useCallback, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { api } from "../lib/api"
import { useOrgPath } from "../lib/org"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

const parseCSV = (raw: string) => {
  const lines = raw.split(/\r?\n/).filter((l) => l.trim())
  if (!lines.length) return []
  const header = lines[0].split(",").map((c) => c.trim().toLowerCase())
  return lines.slice(1).map((line) => {
    const cells = line.split(",").map((c) => c.trim())
    const row: Record<string, string> = {}
    header.forEach((k, i) => { row[k] = cells[i] ?? "" })
    return row
  })
}

export default function UsageCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const orgPath = useOrgPath()
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState({ total: 0, success: 0, failed: 0 })
  const [uploadFile, setUploadFile] = useState<File | null>(null)

  const handleUpload = useCallback(async () => {
    if (!uploadFile) { toast.error(t("usage_create.toast.select_file")); return }
    try {
      setUploading(true)
      const rows = parseCSV(await uploadFile.text())
      if (!rows.length) { toast.error(t("usage_create.toast.empty_csv")); return }
      setUploadProgress({ total: rows.length, success: 0, failed: 0 })
      let success = 0; let failed = 0
      for (const row of rows) {
        try {
          await api.usage.ingest({
            meter_code: row.meter_code || row.meterCode || "",
            customer_id: row.customer_id || row.customerId || "",
            value: Number.parseFloat(row.value ?? "0"),
            recorded_at: row.recorded_at || row.recordedAt || "",
            idempotency_key: row.idempotency_key || row.idempotencyKey || undefined,
          })
          success += 1
        } catch { failed += 1 }
        setUploadProgress({ total: rows.length, success, failed })
      }
      toast.success(t("usage_create.toast.complete_title"), t("usage_create.toast.complete_desc", { success, failed }))
      navigate(orgPath("/usage"))
    } catch (err) {
      toast.error(t("usage_create.toast.failed"), err instanceof Error ? err.message : undefined)
    } finally { setUploading(false) }
  }, [uploadFile, navigate, orgPath, t])

  return (
    <div className="page-content">
      <PageHeader
        title={t("usage_create.header.title")}
        description={t("usage_create.header.description")}
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate(orgPath("/usage"))}
        style={{ cursor: "pointer" }}
      />
      
      <div className="panel" style={{ maxWidth: 640 }}>
        <div className="action-section" style={{ border: "none" }}>
          <p className="muted" style={{ margin: 0 }}>
            {t("usage_create.csv_hint")} <code>meter_code, customer_id, value, recorded_at, idempotency_key</code>
          </p>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("usage_create.fields.file")}</label>
              <input type="file" accept=".csv" style={{ padding: "6px 0" }}
                onChange={(e) => setUploadFile(e.target.files?.[0] ?? null)} data-testid="usage-upload-file" />
            </div>
          </div>
          {uploadProgress.total > 0 ? (
            <div className="muted" style={{ marginTop: 12 }}>
              {t("usage_create.progress", {
                done: uploadProgress.success + uploadProgress.failed,
                total: uploadProgress.total,
                success: uploadProgress.success,
                failed: uploadProgress.failed
              })}
            </div>
          ) : null}
          <div className="action-buttons" style={{ marginTop: 24 }}>
            <Button variant="outline" onClick={() => navigate(orgPath("/usage"))} data-testid="usage-upload-cancel">{t("common.cancel")}</Button>
            <Button variant="default" disabled={uploading || !uploadFile} onClick={handleUpload} data-testid="usage-upload-submit">
              {uploading ? t("usage_create.actions.uploading") : t("usage_create.actions.upload")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
