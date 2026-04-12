import type { ReactNode } from "react"

export type InvoiceParty = {
  name: string
  contactName?: string
  email?: string
  addressLine1?: string
  addressLine2?: string
  city?: string
  region?: string
  postalCode?: string
  country?: string
  taxId?: string
}

export type InvoiceLineItem = {
  id: string
  title: string
  description?: string
  quantity: number
  unitAmountCents: number
  amountCents: number
  currency: string
  periodStart?: string
  periodEnd?: string
  lineType?: string
}

export type InvoiceSummary = {
  number: string
  status: string
  currency: string
  issueDate?: string
  dueDate?: string
  periodStart?: string
  periodEnd?: string
  subtotalCents: number
  taxCents: number
  discountCents?: number
  totalCents: number
  amountDueCents: number
  amountPaidCents?: number
}

export type InvoiceDetailData = {
  invoice: InvoiceSummary
  billedFrom?: InvoiceParty
  billedTo?: InvoiceParty
  lineItems: InvoiceLineItem[]
  paymentInfo?: string[]
  terms?: string[]
  footerNote?: string
}

export type InvoiceDetailProps = {
  data: InvoiceDetailData
  actions?: ReactNode
  onBack?: () => void
  backLabel?: string
  variant?: "full" | "embedded"
  locale?: string
  className?: string
}
