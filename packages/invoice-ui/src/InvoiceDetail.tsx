import type { InvoiceDetailProps, InvoiceLineItem, InvoiceParty } from "./types"

const statusLabel: Record<string, string> = {
  draft: "Draft",
  open: "Open",
  paid: "Paid",
  void: "Void",
  uncollectible: "Uncollectible",
  overdue: "Overdue",
}

const statusClasses: Record<string, string> = {
  draft: "bg-slate-100 text-slate-700 border-slate-200",
  open: "bg-amber-50 text-amber-700 border-amber-200",
  paid: "bg-emerald-50 text-emerald-700 border-emerald-200",
  void: "bg-rose-50 text-rose-700 border-rose-200",
  uncollectible: "bg-rose-50 text-rose-700 border-rose-200",
  overdue: "bg-rose-50 text-rose-700 border-rose-200",
}

const formatCurrency = (amountCents: number, currency: string, locale: string) => {
  const value = amountCents / 100
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
  }).format(value)
}

const formatQuantity = (value: number, locale: string) => {
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: value % 1 === 0 ? 0 : 2,
    maximumFractionDigits: 6,
  }).format(value)
}

const formatDate = (value?: string, locale?: string) => {
  if (!value) return "—"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale ?? "en-US", {
    year: "numeric",
    month: "short",
    day: "2-digit",
  }).format(date)
}

const renderParty = (party?: InvoiceParty) => {
  if (!party) return <span className="text-sm text-muted-foreground">—</span>
  return (
    <div className="space-y-1">
      <div className="font-medium text-slate-900">{party.name}</div>
      {party.contactName ? <div className="text-sm text-slate-700">{party.contactName}</div> : null}
      {party.addressLine1 ? <div className="text-sm text-muted-foreground">{party.addressLine1}</div> : null}
      {party.addressLine2 ? <div className="text-sm text-muted-foreground">{party.addressLine2}</div> : null}
      {(party.city || party.region || party.postalCode) ? (
        <div className="text-sm text-muted-foreground">
          {[party.city, party.region, party.postalCode].filter(Boolean).join(", ")}
        </div>
      ) : null}
      {party.country ? <div className="text-sm text-muted-foreground">{party.country}</div> : null}
      {party.taxId ? <div className="text-sm text-muted-foreground mt-2">{party.taxId}</div> : null}
      {party.email ? <div className="text-sm text-muted-foreground mt-2">{party.email}</div> : null}
    </div>
  )
}

const getLineItemLabel = (item: InvoiceLineItem) => {
  if (item.title) return item.title
  if (item.lineType) return item.lineType.replace(/_/g, " ")
  return "Line item"
}

export default function InvoiceDetail({
  data,
  actions,
  onBack,
  backLabel = "Back",
  variant = "full",
  locale = "en-US",
  className,
}: InvoiceDetailProps) {
  const { invoice, billedFrom, billedTo, lineItems, paymentInfo, terms, footerNote } = data
  const status = invoice.status || "draft"
  const badgeClass = statusClasses[status] ?? statusClasses.draft
  const label = statusLabel[status] ?? status

  const containerClass = variant === "full"
    ? "min-h-screen bg-slate-50/50"
    : "bg-transparent"

  const actionBar = (onBack || actions) ? (
    <div className="border-b border-border bg-white">
      <div className="max-w-6xl mx-auto px-6 py-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        {onBack ? (
          <button
            type="button"
            onClick={onBack}
            className="inline-flex items-center gap-2 text-sm font-medium text-slate-700 hover:text-slate-900"
          >
            <span aria-hidden>←</span>
            {backLabel}
          </button>
        ) : <div />}
        {actions ? <div className="flex flex-wrap items-center gap-2 justify-start sm:justify-end">{actions}</div> : null}
      </div>
    </div>
  ) : null

  return (
    <div className={`${containerClass} ${className ?? ""}`.trim()}>
      {variant === "full" ? actionBar : null}
      <div className="max-w-5xl mx-auto px-6 py-10">
        {variant === "embedded" ? actionBar : null}
        <div className="bg-white border border-slate-200 rounded-2xl shadow-sm">
          <div className="p-8 sm:p-12">
            <div className="flex flex-col gap-8 sm:flex-row sm:items-start sm:justify-between mb-10">
              <div className="flex items-center gap-4">
                <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-600 to-blue-700 flex items-center justify-center shadow-md">
                  <span className="text-white font-semibold text-lg">RW</span>
                </div>
                <div>
                  <div className="font-semibold text-slate-900">{billedFrom?.name ?? "Railzway"}</div>
                  {billedFrom?.email ? (
                    <div className="text-sm text-muted-foreground">{billedFrom.email}</div>
                  ) : null}
                </div>
              </div>
              <div className="text-left sm:text-right">
                <div className="text-xs tracking-[0.2em] text-muted-foreground mb-2">INVOICE</div>
                <div className="text-xl font-semibold text-slate-900 mb-2">{invoice.number}</div>
                <span className={`inline-flex items-center border px-3 py-1 rounded-full text-xs font-medium ${badgeClass}`}>
                  {label}
                </span>
              </div>
            </div>

            <div className="grid gap-8 md:grid-cols-3 mb-10">
              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">BILLED FROM</div>
                {renderParty(billedFrom)}
              </div>
              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">BILLED TO</div>
                {renderParty(billedTo)}
              </div>
              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">INVOICE DETAILS</div>
                <div className="space-y-3">
                  <div>
                    <div className="text-xs text-muted-foreground">Issue Date</div>
                    <div className="text-sm font-medium">{formatDate(invoice.issueDate, locale)}</div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">Due Date</div>
                    <div className="text-sm font-medium">{formatDate(invoice.dueDate, locale)}</div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">Period</div>
                    <div className="text-sm font-medium">
                      {formatDate(invoice.periodStart, locale)} — {formatDate(invoice.periodEnd, locale)}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="border-t border-slate-200 mb-6" />

            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-slate-50 text-slate-500">
                  <tr>
                    <th className="text-left font-medium px-4 py-3 w-[50%]">Item Description</th>
                    <th className="text-center font-medium px-4 py-3 w-[12%]">Qty</th>
                    <th className="text-right font-medium px-4 py-3 w-[18%]">Rate</th>
                    <th className="text-right font-medium px-4 py-3 w-[20%]">Amount</th>
                  </tr>
                </thead>
                <tbody>
                  {lineItems.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">No line items</td>
                    </tr>
                  ) : lineItems.map((item) => (
                    <tr key={item.id} className="border-b border-slate-100">
                      <td className="px-4 py-4">
                        <div className="font-medium text-slate-900 mb-1">{getLineItemLabel(item)}</div>
                        {item.description ? (
                          <div className="text-xs text-muted-foreground">{item.description}</div>
                        ) : null}
                        {(item.periodStart || item.periodEnd) ? (
                          <div className="text-xs text-muted-foreground mt-1">
                            {formatDate(item.periodStart, locale)} — {formatDate(item.periodEnd, locale)}
                          </div>
                        ) : null}
                      </td>
                      <td className="px-4 py-4 text-center">{formatQuantity(item.quantity, locale)}</td>
                      <td className="px-4 py-4 text-right">{formatCurrency(item.unitAmountCents, item.currency, locale)}</td>
                      <td className="px-4 py-4 text-right font-medium">
                        {formatCurrency(item.amountCents, item.currency, locale)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="border-t border-slate-200 my-6" />

            <div className="flex justify-end">
              <div className="w-full max-w-sm space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Subtotal</span>
                  <span>{formatCurrency(invoice.subtotalCents, invoice.currency, locale)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Tax</span>
                  <span>{formatCurrency(invoice.taxCents, invoice.currency, locale)}</span>
                </div>
                {typeof invoice.discountCents === "number" ? (
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Discount</span>
                    <span className="text-rose-600">-{formatCurrency(invoice.discountCents, invoice.currency, locale)}</span>
                  </div>
                ) : null}
                <div className="border-t border-slate-200 pt-3 flex justify-between items-center">
                  <span className="font-semibold">Total Amount</span>
                  <span className="font-semibold">{formatCurrency(invoice.totalCents, invoice.currency, locale)}</span>
                </div>
                <div className="flex justify-between text-sm text-muted-foreground">
                  <span>Amount Due</span>
                  <span>{formatCurrency(invoice.amountDueCents, invoice.currency, locale)}</span>
                </div>
              </div>
            </div>

            {(paymentInfo && paymentInfo.length > 0) || (terms && terms.length > 0) ? (
              <>
                <div className="border-t border-slate-200 my-8" />
                <div className="grid gap-8 md:grid-cols-2">
                  {paymentInfo && paymentInfo.length > 0 ? (
                    <div>
                      <div className="text-xs tracking-wide text-muted-foreground mb-3">PAYMENT INFORMATION</div>
                      <div className="text-sm text-muted-foreground space-y-1">
                        {paymentInfo.map((line) => (
                          <div key={line}>{line}</div>
                        ))}
                      </div>
                    </div>
                  ) : null}
                  {terms && terms.length > 0 ? (
                    <div>
                      <div className="text-xs tracking-wide text-muted-foreground mb-3">TERMS & CONDITIONS</div>
                      <div className="text-sm text-muted-foreground space-y-1">
                        {terms.map((line) => (
                          <div key={line}>{line}</div>
                        ))}
                      </div>
                    </div>
                  ) : null}
                </div>
              </>
            ) : null}

            {footerNote ? (
              <div className="mt-10 pt-6 border-t border-slate-200 text-center text-sm text-muted-foreground">
                {footerNote}
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  )
}
