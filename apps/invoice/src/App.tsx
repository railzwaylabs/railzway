import { InvoiceDetail, type InvoiceDetailData } from "@railzway/invoice-ui"

const sample: InvoiceDetailData = {
  invoice: {
    number: "INV-90823",
    status: "paid",
    currency: "USD",
    issueDate: "2026-03-15T00:00:00Z",
    dueDate: "2026-04-15T00:00:00Z",
    periodStart: "2026-03-01T00:00:00Z",
    periodEnd: "2026-03-31T23:59:59Z",
    subtotalCents: 3930000,
    taxCents: 393000,
    totalCents: 4126500,
    amountDueCents: 0,
    amountPaidCents: 4126500,
  },
  billedFrom: {
    name: "Acme Solutions Inc.",
    addressLine1: "123 Innovation Drive, Suite 400",
    city: "San Francisco",
    region: "CA",
    postalCode: "94105",
    country: "US",
    taxId: "VAT-US-123456789",
    email: "billing@acmesolutions.com",
  },
  billedTo: {
    name: "TechStart Ventures",
    contactName: "Sarah Johnson",
    addressLine1: "456 Startup Lane",
    city: "Austin",
    region: "TX",
    postalCode: "78701",
    country: "US",
    email: "accounts@techstart.com",
  },
  lineItems: [
    {
      id: "line-1",
      title: "Professional Services - Q1 2026",
      description: "Custom software development and consulting services",
      quantity: 160,
      unitAmountCents: 15000,
      amountCents: 2400000,
      currency: "USD",
    },
    {
      id: "line-2",
      title: "Cloud Infrastructure Management",
      description: "AWS infrastructure setup, monitoring, and optimization",
      quantity: 3,
      unitAmountCents: 250000,
      amountCents: 750000,
      currency: "USD",
    },
    {
      id: "line-3",
      title: "Technical Support & Maintenance",
      description: "24/7 support package with SLA guarantees",
      quantity: 1,
      unitAmountCents: 300000,
      amountCents: 300000,
      currency: "USD",
    },
    {
      id: "line-4",
      title: "Code Review & Quality Assurance",
      description: "Comprehensive code audits and testing services",
      quantity: 40,
      unitAmountCents: 12000,
      amountCents: 480000,
      currency: "USD",
    },
  ],
  paymentInfo: [
    "Bank: First National Bank",
    "Account: ****7890",
    "Routing: 021000021",
    "Wire transfers typically process within 1-2 business days.",
  ],
  terms: [
    "Payment is due within 30 days of invoice date.",
    "Late payments may incur a 1.5% monthly interest charge.",
    "Please include invoice number in payment reference.",
  ],
  footerNote: "Thank you for your business! For questions, contact billing@acmesolutions.com.",
}

export default function App() {
  return <InvoiceDetail data={sample} />
}
