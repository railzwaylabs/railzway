import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useNavigate } from "react-router-dom"
import { api } from "../lib/api"
import { setOrgId } from "../lib/auth"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"
import type { ReferenceCountry, ReferenceTimezone } from "../lib/types"

function IconOrg() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="10" cy="6" r="3"/>
      <path d="M3 17c0-3.866 3.134-6 7-6s7 2.134 7 6" strokeLinecap="round"/>
    </svg>
  )
}

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function OrganizationsCreate() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [actionLoading, setActionLoading] = useState(false)
  const [createForm, setCreateForm] = useState({ name: "", countryCode: "", timezoneName: "" })
  const [countries, setCountries] = useState<ReferenceCountry[]>([])
  const [timezones, setTimezones] = useState<ReferenceTimezone[]>([])
  const [countriesLoading, setCountriesLoading] = useState(false)
  const [timezonesLoading, setTimezonesLoading] = useState(false)

  const createValidation = useMemo(() => {
    const e: string[] = []
    if (!createForm.name.trim()) e.push(t("organizations_create.validation.name_required"))
    return e
  }, [createForm, t])

  useEffect(() => {
    let mounted = true
    setCountriesLoading(true)
    api.reference
      .countries()
      .then((items) => {
        if (!mounted) return
        setCountries(items)
      })
      .catch((err) => {
        toast.error(t("organizations_create.toast.countries_failed"), err instanceof Error ? err.message : undefined)
      })
      .finally(() => {
        if (mounted) setCountriesLoading(false)
      })
    return () => {
      mounted = false
    }
  }, [])

  useEffect(() => {
    let mounted = true
    const code = createForm.countryCode.trim()
    if (!code) {
      setTimezones([])
      return () => {
        mounted = false
      }
    }
    setTimezonesLoading(true)
    api.reference
      .timezones(code)
      .then((items) => {
        if (!mounted) return
        setTimezones(items)
        if (!items.length) return
        const hasSelected = items.some((tz) => tz.name === createForm.timezoneName)
        if (!hasSelected) {
          setCreateForm((p) => ({ ...p, timezoneName: items[0].name }))
        }
      })
      .catch((err) => {
        toast.error(t("organizations_create.toast.timezones_failed"), err instanceof Error ? err.message : undefined)
      })
      .finally(() => {
        if (mounted) setTimezonesLoading(false)
      })
    return () => {
      mounted = false
    }
  }, [createForm.countryCode])

  const handleCreate = useCallback(async () => {
    try {
      setActionLoading(true)
      const resp = await api.organizations.create({
        name: createForm.name.trim(),
        country_code: createForm.countryCode.trim() || undefined,
        timezone_name: createForm.timezoneName.trim() || undefined,
      })
      toast.success(t("organizations_create.toast.created"), resp.id)
      setOrgId(resp.id)
      window.location.assign(`/organizations/${resp.id}/edit`)
    } catch (err) {
      toast.error(t("organizations_create.toast.create_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [createForm, navigate, t])

  return (
    <div className="org-create-shell">
      <div className="org-create-card">
        <div className="org-create-header">
          <button className="org-create-back" type="button" onClick={() => navigate("/organizations")}>
            <IconBack />
          </button>
          <div>
            <div className="org-create-title">{t("organizations_create.header.title")}</div>
            <div className="org-create-subtitle">{t("organizations_create.header.subtitle")}</div>
          </div>
        </div>

        <div className="org-create-body">
          <div className="org-create-field">
            <label className="org-create-label">{t("organizations_create.fields.name_label")}</label>
            <Input
              className="org-create-input"
              value={createForm.name}
              autoFocus
              placeholder={t("organizations_create.fields.name_placeholder")}
              onChange={(e) => setCreateForm((p) => ({ ...p, name: e.target.value }))}
            />
          </div>

          <div className="org-create-grid">
            <div className="org-create-field">
              <label className="org-create-label">{t("organizations_create.fields.country_label")}</label>
              {countries.length > 0 ? (
                <Select
                  value={createForm.countryCode}
                  onValueChange={(value) =>
                    setCreateForm((p) => ({ ...p, countryCode: value, timezoneName: "" }))
                  }
                >
                  <SelectTrigger className="org-create-input" disabled={countriesLoading}>
                    <SelectValue placeholder={countriesLoading ? t("common.loading") : t("organizations_create.fields.country_placeholder")} />
                  </SelectTrigger>
                  <SelectContent>
                    {countries.map((country) => (
                      <SelectItem key={country.code} value={country.code}>
                        {country.name} ({country.code})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  className="org-create-input"
                  value={createForm.countryCode}
                  placeholder={t("organizations_create.fields.country_input_placeholder")}
                  onChange={(e) => setCreateForm((p) => ({ ...p, countryCode: e.target.value }))}
                />
              )}
              <div className="org-create-hint">{t("organizations_create.fields.country_hint")}</div>
            </div>
            <div className="org-create-field">
              <label className="org-create-label">{t("organizations_create.fields.timezone_label")}</label>
              {timezones.length > 0 ? (
                <Select
                  value={createForm.timezoneName}
                  onValueChange={(value) => setCreateForm((p) => ({ ...p, timezoneName: value }))}
                >
                  <SelectTrigger className="org-create-input" disabled={timezonesLoading}>
                    <SelectValue placeholder={timezonesLoading ? t("common.loading") : t("organizations_create.fields.timezone_placeholder")} />
                  </SelectTrigger>
                  <SelectContent>
                    {timezones.map((tz) => (
                      <SelectItem key={tz.name} value={tz.name}>
                        {tz.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  className="org-create-input"
                  value={createForm.timezoneName}
                  placeholder={t("organizations_create.fields.timezone_input_placeholder")}
                  onChange={(e) => setCreateForm((p) => ({ ...p, timezoneName: e.target.value }))}
                />
              )}
              <div className="org-create-hint">{t("organizations_create.fields.timezone_hint")}</div>
            </div>
          </div>

          {createValidation.length > 0 ? <div className="inline-error">{createValidation.join(" ")}</div> : null}

          <div className="org-create-actions">
            <Button variant="outline" onClick={() => navigate("/organizations")}>{t("common.cancel")}</Button>
            <Button variant="default" disabled={actionLoading || createValidation.length > 0} onClick={handleCreate}>
              {actionLoading ? t("common.creating") : t("organizations_create.actions.create")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
