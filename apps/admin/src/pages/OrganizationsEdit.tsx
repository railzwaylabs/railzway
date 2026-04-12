import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useLocation, useNavigate, useParams } from "react-router-dom"
import HelpHint from "../components/HelpHint"
import AutoCompleteInput from "../components/AutoCompleteInput"
import { formatDate, rfc3339Hint } from "../lib/display"
import { api } from "../lib/api"
import { isCurrencyCode, isEmail } from "../lib/validation"
import { currencyHint } from "../lib/hints"
import { useCurrencies } from "../lib/reference"
import type { InvoiceNumberFormat, Organization, OrganizationMemberInfo, ReferenceCountry, ReferenceTimezone } from "../lib/types"
import DataTable from "../components/DataTable"
import PageHeader from "../components/PageHeader"
import { toast } from "../components/Toast"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select"

function IconBack() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M15 10H5M5 10L10 5M5 10L10 15" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

export default function OrganizationsEdit() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { options: currencyOptions, loading: currenciesLoading } = useCurrencies()
  
  const [org, setOrg] = useState<Organization | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [formats, setFormats] = useState<InvoiceNumberFormat[]>([])
  const [formatsLoading, setFormatsLoading] = useState(false)
  const [members, setMembers] = useState<OrganizationMemberInfo[]>([])
  const [membersLoading, setMembersLoading] = useState(false)
  const [countries, setCountries] = useState<ReferenceCountry[]>([])
  const [timezones, setTimezones] = useState<ReferenceTimezone[]>([])
  const [countriesLoading, setCountriesLoading] = useState(false)
  const [timezonesLoading, setTimezonesLoading] = useState(false)

  const [updateForm, setUpdateForm] = useState({ name: "" })
  const [localeForm, setLocaleForm] = useState({ countryCode: "", timezoneName: "" })
  const [billingForm, setBillingForm] = useState({
    currency: "", timezone: "", invoicePrefix: "", invoiceNumberFormat: "", invoiceSequenceScope: ""
  })
  const [formatForm, setFormatForm] = useState({
    format: "", sequenceScope: "org_month", effectiveFrom: "", effectiveTo: ""
  })
  const [closeForm, setCloseForm] = useState({ formatId: "", effectiveTo: "" })
  const [inviteForm, setInviteForm] = useState({ email: "", role: "member" })

  const updateValidation = useMemo(() => {
    const e: string[] = []
    if (!updateForm.name.trim()) e.push(t("organizations_edit.validation.name_required"))
    return e
  }, [updateForm, t])
  const billingValidation = useMemo(() => {
    const e: string[] = []
    if (billingForm.currency.trim() && !isCurrencyCode(billingForm.currency)) e.push(t("organizations_edit.validation.invalid_currency"))
    return e
  }, [billingForm, t])
  const localeValidation = useMemo(() => {
    const e: string[] = []
    if (!localeForm.countryCode.trim()) e.push(t("organizations_edit.validation.country_required"))
    if (!localeForm.timezoneName.trim()) e.push(t("organizations_edit.validation.timezone_required"))
    return e
  }, [localeForm, t])
  const formatValidation = useMemo(() => {
    const e: string[] = []
    if (!formatForm.format.trim()) e.push(t("organizations_edit.validation.format_required"))
    if (!formatForm.effectiveFrom.trim()) e.push(t("organizations_edit.validation.effective_from_required"))
    return e
  }, [formatForm, t])
  const closeValidation = useMemo(() => {
    const e: string[] = []
    if (!closeForm.formatId.trim()) e.push(t("organizations_edit.validation.format_id_required"))
    if (!closeForm.effectiveTo.trim()) e.push(t("organizations_edit.validation.effective_to_required"))
    return e
  }, [closeForm, t])
  const inviteValidation = useMemo(() => {
    const e: string[] = []
    if (!inviteForm.email.trim() || !isEmail(inviteForm.email)) e.push(t("organizations_edit.validation.invite_email_required"))
    return e
  }, [inviteForm, t])

  const invoiceFormatPreview = useMemo(() => {
    const raw = billingForm.invoiceNumberFormat.trim()
    if (!raw) return ""
    const now = new Date()
    const yyyy = String(now.getFullYear())
    const yy = yyyy.slice(-2)
    const mm = String(now.getMonth() + 1).padStart(2, "0")
    const dd = String(now.getDate()).padStart(2, "0")
    const prefix = billingForm.invoicePrefix.trim() || "INV"
    const seqValue = 123
    return raw
      .replace(/\{PREFIX\}/g, prefix)
      .replace(/\{prefix\}/g, prefix)
      .replace(/\{YYYY\}/g, yyyy)
      .replace(/\{YY\}/g, yy)
      .replace(/\{MM\}/g, mm)
      .replace(/\{DD\}/g, dd)
      .replace(/\{SEQ:(\d+)\}/gi, (_, len) => String(seqValue).padStart(Number(len), "0"))
      .replace(/\{SEQ\}/g, String(seqValue))
      .replace(/\{seq\}/g, String(seqValue))
  }, [billingForm.invoiceNumberFormat, billingForm.invoicePrefix])

  const loadData = useCallback(async () => {
    if (!id) return
    try {
      setLoading(true)
      const [orgResp, formatsResp] = await Promise.all([
        api.organizations.get(id),
        api.organizations.listInvoiceFormats(id).catch(() => [] as InvoiceNumberFormat[])
      ])
      setOrg(orgResp)
      setUpdateForm({ name: orgResp.name })
      setLocaleForm({
        countryCode: orgResp.country_code ?? "",
        timezoneName: orgResp.timezone_name ?? "",
      })
      setFormats(formatsResp)
    } catch (err) {
      toast.error(t("organizations_edit.toast.load_failed"), err instanceof Error ? err.message : undefined)
    } finally { setLoading(false) }
  }, [id, t])

  useEffect(() => { void loadData() }, [loadData])

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
        toast.error(t("organizations_edit.toast.countries_failed"), err instanceof Error ? err.message : undefined)
      })
      .finally(() => {
        if (mounted) setCountriesLoading(false)
      })
    return () => { mounted = false }
  }, [t])

  useEffect(() => {
    let mounted = true
    const countryCode = localeForm.countryCode.trim()
    if (!countryCode) {
      setTimezones([])
      return () => { mounted = false }
    }
    setTimezonesLoading(true)
    api.reference
      .timezones(countryCode)
      .then((items) => {
        if (!mounted) return
        setTimezones(items)
        if (!items.length) return
        const hasSelected = items.some((tz) => tz.name === localeForm.timezoneName)
        if (!hasSelected) {
          setLocaleForm((p) => ({ ...p, timezoneName: items[0].name }))
        }
      })
      .catch((err) => {
        toast.error(t("organizations_edit.toast.timezones_failed"), err instanceof Error ? err.message : undefined)
        setTimezones([])
      })
      .finally(() => {
        if (mounted) setTimezonesLoading(false)
      })
    return () => { mounted = false }
  }, [localeForm.countryCode, t])

  const loadMembers = useCallback(async () => {
    if (!id) return
    try {
      setMembersLoading(true)
      const resp = await api.organizations.listMembers(id)
      setMembers(resp)
    } catch {
      setMembers([])
    } finally { setMembersLoading(false) }
  }, [id])

  useEffect(() => { void loadMembers() }, [loadMembers])

  const loadFormats = useCallback(async () => {
    if (!id) return
    try {
      setFormatsLoading(true)
      const resp = await api.organizations.listInvoiceFormats(id)
      setFormats(resp)
    } catch { setFormats([]) } finally { setFormatsLoading(false) }
  }, [id])

  const handleUpdate = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      const resp = await api.organizations.update(id, { name: updateForm.name.trim() })
      toast.success(t("organizations_edit.toast.updated"), resp.id)
      void loadData()
    } catch (err) {
      toast.error(t("organizations_edit.toast.update_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, updateForm, loadData, t])

  const handleUpdateLocale = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      const resp = await api.organizations.update(id, {
        country_code: localeForm.countryCode.trim(),
        timezone_name: localeForm.timezoneName.trim(),
      })
      toast.success(t("organizations_edit.toast.locale_updated"), resp.id)
      void loadData()
    } catch (err) {
      toast.error(t("organizations_edit.toast.locale_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, localeForm, loadData, t])

  const handleBillingPreferences = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      await api.organizations.setBillingPreferences(id, {
        currency: billingForm.currency.trim() || undefined,
        timezone: billingForm.timezone.trim() || undefined,
        invoice_prefix: billingForm.invoicePrefix.trim() || undefined,
        invoice_number_format: billingForm.invoiceNumberFormat.trim() || undefined,
        invoice_sequence_scope: billingForm.invoiceSequenceScope.trim() || undefined,
      })
      toast.success(t("organizations_edit.toast.billing_saved"))
      setBillingForm({ currency: "", timezone: "", invoicePrefix: "", invoiceNumberFormat: "", invoiceSequenceScope: "" })
    } catch (err) {
      toast.error(t("organizations_edit.toast.billing_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, billingForm, t])

  const handleCreateFormat = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      const resp = await api.organizations.createInvoiceFormat(id, {
        format: formatForm.format.trim(),
        sequence_scope: formatForm.sequenceScope.trim(),
        effective_from: formatForm.effectiveFrom.trim(),
        effective_to: formatForm.effectiveTo.trim() || undefined,
      })
      toast.success(t("organizations_edit.toast.format_created"), resp.id)
      setFormatForm((p) => ({ ...p, format: "", effectiveFrom: "", effectiveTo: "" }))
      void loadFormats()
    } catch (err) {
      toast.error(t("organizations_edit.toast.format_create_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, formatForm, loadFormats, t])

  const handleCloseFormat = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      const resp = await api.organizations.closeInvoiceFormat(id, closeForm.formatId.trim(), {
        effective_to: closeForm.effectiveTo.trim(),
      })
      toast.success(t("organizations_edit.toast.format_closed"), resp.id)
      setCloseForm({ formatId: "", effectiveTo: "" })
      void loadFormats()
    } catch (err) {
      toast.error(t("organizations_edit.toast.format_close_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, closeForm, loadFormats, t])

  const handleInvite = useCallback(async () => {
    if (!id) return
    try {
      setActionLoading(true)
      await api.organizations.inviteMembers(id, {
        invites: [{ email: inviteForm.email.trim(), role: inviteForm.role.trim() }],
      })
      toast.success(t("organizations_edit.toast.invite_sent", { email: inviteForm.email }))
      setInviteForm((p) => ({ ...p, email: "" }))
    } catch (err) {
      toast.error(t("organizations_edit.toast.invite_failed"), err instanceof Error ? err.message : undefined)
    } finally { setActionLoading(false) }
  }, [id, inviteForm, t])

  const fmtColumns = [
    { key: "format", label: t("organizations_edit.formats.table.columns.format"), render: (r: InvoiceNumberFormat) => (
      <div><div style={{ fontWeight: 600 }}>{r.format}</div><span className="muted">{r.sequence_scope}</span></div>
    ) },
    { key: "effective_from", label: t("organizations_edit.formats.table.columns.effective_from"), width: "150px", render: (r: InvoiceNumberFormat) => (
      <span className="muted">{formatDate(r.effective_from)}</span>
    ) },
    { key: "status", label: t("organizations_edit.formats.table.columns.status"), width: "100px", render: (r: InvoiceNumberFormat) => (
      <span className={`badge ${r.effective_to ? "badge-muted" : "badge-success"}`}>
        {r.effective_to ? t("organizations_edit.formats.status.closed") : t("organizations_edit.formats.status.active")}
      </span>
    ) },
  ]

  const memberId = useMemo(() => {
    const params = new URLSearchParams(location.search)
    return params.get("member_id") ?? ""
  }, [location.search])

  const memberColumns = [
    { key: "display_name", label: t("organizations_edit.members.table.columns.member"), render: (r: OrganizationMemberInfo) => (
      <div>
        <div style={{ fontWeight: 600, display: "flex", gap: 8, alignItems: "center" }}>
          {r.display_name || r.email}
          {memberId && r.user_id === memberId ? <span className="badge badge-info">{t("organizations_edit.members.selected")}</span> : null}
        </div>
        <span className="muted">{r.email}</span>
      </div>
    ) },
    { key: "role", label: t("organizations_edit.members.table.columns.role"), width: "140px", render: (r: OrganizationMemberInfo) => (
      <span className="badge badge-muted">{r.role}</span>
    ) },
    { key: "created_at", label: t("organizations_edit.members.table.columns.added"), width: "160px", render: (r: OrganizationMemberInfo) => (
      <span className="muted">{formatDate(r.created_at)}</span>
    ) },
  ]

  if (loading) {
    return <div className="page-content"><div className="loader" /></div>
  }

  return (
    <div className="page-content">
      <PageHeader 
        title={org ? t("organizations_edit.header.title_with_name", { name: org.name }) : t("organizations_edit.header.title")} 
        description={id} 
        icon={<IconBack />}
        // @ts-expect-error type
        onIconClick={() => navigate("/organizations")}
        style={{ cursor: "pointer" }}
      />
      <div className="action-panel" id="invoice-formats">
        <div className="action-section">
          <div className="action-section-title">{t("organizations_edit.rename.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.rename.name_label")}</label>
              <Input className="action-input" value={updateForm.name}
                onChange={(e) => setUpdateForm((p) => ({ ...p, name: e.target.value }))} />
            </div>
          </div>
          {updateValidation.length > 0 ? <div className="inline-error">{updateValidation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="default" disabled={actionLoading || updateValidation.length > 0} onClick={handleUpdate}>
              {actionLoading ? t("common.updating") : t("common.save_changes")}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("organizations_edit.locale.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.locale.country_label")}</label>
              {countries.length > 0 ? (
                <Select
                  value={localeForm.countryCode}
                  onValueChange={(value) => setLocaleForm((p) => ({ ...p, countryCode: value, timezoneName: "" }))}
                >
                  <SelectTrigger className="action-select" disabled={countriesLoading}>
                    <SelectValue placeholder={countriesLoading ? t("common.loading") : t("organizations_edit.locale.country_placeholder")} />
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
                <Input className="action-input" value={localeForm.countryCode} placeholder={t("organizations_edit.locale.country_input_placeholder")}
                  onChange={(e) => setLocaleForm((p) => ({ ...p, countryCode: e.target.value }))} />
              )}
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.locale.timezone_label")}</label>
              {timezones.length > 0 ? (
                <Select
                  value={localeForm.timezoneName}
                  onValueChange={(value) => setLocaleForm((p) => ({ ...p, timezoneName: value }))}
                >
                  <SelectTrigger className="action-select" disabled={timezonesLoading}>
                    <SelectValue placeholder={timezonesLoading ? t("common.loading") : t("organizations_edit.locale.timezone_placeholder")} />
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
                <Input className="action-input" value={localeForm.timezoneName} placeholder={t("organizations_edit.locale.timezone_input_placeholder")}
                  onChange={(e) => setLocaleForm((p) => ({ ...p, timezoneName: e.target.value }))} />
              )}
            </div>
          </div>
          {localeValidation.length > 0 ? <div className="inline-error">{localeValidation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="default" disabled={actionLoading || localeValidation.length > 0} onClick={handleUpdateLocale}>
              {actionLoading ? t("common.saving") : t("organizations_edit.locale.save")}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("organizations_edit.billing.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <AutoCompleteInput
                id="organizations-edit-currency"
                label={<>{t("organizations_edit.billing.currency_label")} <HelpHint text={currencyHint} /></>}
                value={billingForm.currency}
                options={currencyOptions}
                placeholder={currenciesLoading ? t("common.loading") : t("organizations_edit.billing.currency_placeholder")}
                onChange={(value) => setBillingForm((p) => ({ ...p, currency: value }))}
              />
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.billing.timezone_label")}</label>
              {timezones.length > 0 ? (
                <Select
                  value={billingForm.timezone}
                  onValueChange={(value) => setBillingForm((p) => ({ ...p, timezone: value }))}
                >
                  <SelectTrigger className="action-select" disabled={timezonesLoading}>
                    <SelectValue placeholder={timezonesLoading ? t("common.loading") : t("organizations_edit.billing.timezone_placeholder")} />
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
                <Input className="action-input" value={billingForm.timezone} placeholder={t("organizations_edit.billing.timezone_input_placeholder")}
                  onChange={(e) => setBillingForm((p) => ({ ...p, timezone: e.target.value }))} />
              )}
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.billing.invoice_prefix_label")}</label>
              <Input className="action-input" value={billingForm.invoicePrefix} placeholder={t("organizations_edit.billing.invoice_prefix_placeholder")}
                onChange={(e) => setBillingForm((p) => ({ ...p, invoicePrefix: e.target.value }))} />
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.billing.invoice_format_label")}</label>
              <Input className="action-input" value={billingForm.invoiceNumberFormat} placeholder={t("organizations_edit.billing.invoice_format_placeholder")}
                onChange={(e) => setBillingForm((p) => ({ ...p, invoiceNumberFormat: e.target.value }))} />
              <div className="muted" style={{ marginTop: 6 }}>
                {t("organizations_edit.billing.invoice_format_preview_label")}{" "}
                <span className="cell-mono">{invoiceFormatPreview || t("common.empty_dash")}</span>
              </div>
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.billing.sequence_scope_label")}</label>
              <Input className="action-input" value={billingForm.invoiceSequenceScope} placeholder={t("organizations_edit.billing.sequence_scope_placeholder")}
                onChange={(e) => setBillingForm((p) => ({ ...p, invoiceSequenceScope: e.target.value }))} />
            </div>
          </div>
          {billingValidation.length > 0 ? <div className="inline-error">{billingValidation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="default" disabled={actionLoading || billingValidation.length > 0} onClick={handleBillingPreferences}>
              {actionLoading ? t("common.saving") : t("organizations_edit.billing.save")}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("organizations_edit.invite.title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.invite.email_label")}</label>
              <Input className="action-input" type="email" value={inviteForm.email}
                onChange={(e) => setInviteForm((p) => ({ ...p, email: e.target.value }))} />
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.invite.role_label")}</label>
              <Select value={inviteForm.role} onValueChange={(value) => setInviteForm((p) => ({ ...p, role: value }))}>
                <SelectTrigger className="action-select">
                  <SelectValue placeholder={t("organizations_edit.invite.role_placeholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="owner">{t("organizations_edit.invite.roles.owner")}</SelectItem>
                  <SelectItem value="admin">{t("organizations_edit.invite.roles.admin")}</SelectItem>
                  <SelectItem value="member">{t("organizations_edit.invite.roles.member")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          {inviteValidation.length > 0 ? <div className="inline-error">{inviteValidation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="default" disabled={actionLoading || inviteValidation.length > 0} onClick={handleInvite}>
              {actionLoading ? t("common.sending") : t("organizations_edit.invite.send")}
            </Button>
          </div>
        </div>
      </div>

      <div className="action-panel" id="members">
        <div className="action-panel-header">
          <span className="panel-title">{t("organizations_edit.members.title")}</span>
        </div>
        <DataTable
          columns={memberColumns as Parameters<typeof DataTable>[0]["columns"]}
          data={members}
          loading={membersLoading}
          emptyTitle={t("organizations_edit.members.empty_title")}
          emptyDesc={t("organizations_edit.members.empty_desc")}
          keyExtractor={(r) => r.user_id}
        />
      </div>

      <div className="action-panel">
        <div className="action-panel-header">
          <span className="panel-title">{t("organizations_edit.formats.title")}</span>
        </div>
        
        {formats.length > 0 && (
          <div className="action-section" style={{ borderBottom: "1px solid var(--border-color)", paddingBottom: 20 }}>
            <DataTable
              columns={fmtColumns as Parameters<typeof DataTable>[0]["columns"]}
              data={formats}
              loading={formatsLoading}
            />
          </div>
        )}

        <div className="action-section">
          <div className="action-section-title">{t("organizations_edit.formats.create_title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.formats.fields.format_label")}</label>
              <Input className="action-input" value={formatForm.format} placeholder={t("organizations_edit.formats.fields.format_placeholder")}
                onChange={(e) => setFormatForm((p) => ({ ...p, format: e.target.value }))} />
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.formats.fields.sequence_scope_label")}</label>
              <Input className="action-input" value={formatForm.sequenceScope} placeholder={t("organizations_edit.formats.fields.sequence_scope_placeholder")}
                onChange={(e) => setFormatForm((p) => ({ ...p, sequenceScope: e.target.value }))} />
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.formats.fields.effective_from_label")} <HelpHint text={rfc3339Hint} /></label>
              <Input className="action-input" type="datetime-local" value={formatForm.effectiveFrom}
                onChange={(e) => setFormatForm((p) => ({ ...p, effectiveFrom: e.target.value }))} />
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.formats.fields.effective_to_label")} <HelpHint text={rfc3339Hint} /></label>
              <Input className="action-input" type="datetime-local" min={formatForm.effectiveFrom || undefined} value={formatForm.effectiveTo}
                onChange={(e) => setFormatForm((p) => ({ ...p, effectiveTo: e.target.value }))} />
            </div>
          </div>
          {formatValidation.length > 0 ? <div className="inline-error">{formatValidation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="default" disabled={actionLoading || formatValidation.length > 0} onClick={handleCreateFormat}>
              {actionLoading ? t("common.creating") : t("organizations_edit.formats.create")}
            </Button>
          </div>
        </div>

        <div className="action-section">
          <div className="action-section-title">{t("organizations_edit.formats.close_title")}</div>
          <div className="action-fields">
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.formats.fields.format_id_label")}</label>
              <Input className="action-input" value={closeForm.formatId}
                onChange={(e) => setCloseForm((p) => ({ ...p, formatId: e.target.value }))} />
            </div>
            <div className="action-field">
              <label className="action-label">{t("organizations_edit.formats.fields.close_effective_to_label")} <HelpHint text={rfc3339Hint} /></label>
              <Input className="action-input" type="datetime-local" value={closeForm.effectiveTo}
                onChange={(e) => setCloseForm((p) => ({ ...p, effectiveTo: e.target.value }))} />
            </div>
          </div>
          {closeValidation.length > 0 ? <div className="inline-error">{closeValidation.join(" ")}</div> : null}
          <div className="action-buttons">
            <Button variant="outline" disabled={actionLoading || closeValidation.length > 0} onClick={handleCloseFormat}>
              {actionLoading ? t("common.closing") : t("organizations_edit.formats.close")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
