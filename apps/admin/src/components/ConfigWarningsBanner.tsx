import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { api } from "../lib/api"
import { ConfigWarningsResponse, ConfigWarning } from "../lib/types"

export default function ConfigWarningsBanner() {
  const [warnings, setWarnings] = useState<ConfigWarning[]>([])
  const { t } = useTranslation()

  useEffect(() => {
    let active = true
    api.settings
      .warnings()
      .then((resp: ConfigWarningsResponse) => {
        if (!active) return
        setWarnings(resp.warnings || [])
      })
      .catch(() => undefined)
    return () => {
      active = false
    }
  }, [])

  if (!warnings.length) {
    return null
  }

  return (
    <div className="config-warnings" role="status">
      {warnings.map((warning) => {
        const key = `warnings.${warning.code}`
        const message = t(key, {
          ...(warning.metadata as Record<string, string | number> | undefined),
          defaultValue: warning.code,
        })
        return (
          <div
            key={`${warning.module ?? "system"}:${warning.code}`}
            className="config-warning"
            data-testid="config-warning"
          >
            <span className="config-warning-badge">{warning.module ?? "system"}</span>
            <span className="config-warning-message">{message}</span>
            {warning.link ? (
              <a className="config-warning-link" href={warning.link}>
                {t("common.review")}
              </a>
            ) : null}
          </div>
        )
      })}
    </div>
  )
}
