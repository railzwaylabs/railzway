import { ArrowLeft, Download, Send, CheckCircle2 } from "lucide-react"
import { Button } from "./ui/button"
import { Card } from "./ui/card"
import { Badge } from "./ui/badge"
import { Separator } from "./ui/separator"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table"

export default function InvoiceDetail() {
  const invoiceData = {
    invoiceNumber: "INV-90823",
    status: "paid",
    issueDate: "March 15, 2026",
    dueDate: "April 15, 2026",
    billedFrom: {
      companyName: "Acme Solutions Inc.",
      address: "123 Innovation Drive, Suite 400",
      city: "San Francisco, CA 94105",
      vatId: "VAT-US-123456789",
      email: "billing@acmesolutions.com",
    },
    billedTo: {
      companyName: "TechStart Ventures",
      contactName: "Sarah Johnson",
      address: "456 Startup Lane",
      city: "Austin, TX 78701",
      email: "accounts@techstart.com",
    },
    lineItems: [
      {
        id: 1,
        title: "Professional Services - Q1 2026",
        description: "Custom software development and consulting services",
        quantity: 160,
        rate: 150.0,
        amount: 24000.0,
      },
      {
        id: 2,
        title: "Cloud Infrastructure Management",
        description: "AWS infrastructure setup, monitoring, and optimization",
        quantity: 3,
        rate: 2500.0,
        amount: 7500.0,
      },
      {
        id: 3,
        title: "Technical Support & Maintenance",
        description: "24/7 support package with SLA guarantees",
        quantity: 1,
        rate: 3000.0,
        amount: 3000.0,
      },
      {
        id: 4,
        title: "Code Review & Quality Assurance",
        description: "Comprehensive code audits and testing services",
        quantity: 40,
        rate: 120.0,
        amount: 4800.0,
      },
    ],
    subtotal: 39300.0,
    tax: 3930.0,
    discount: 1965.0,
    total: 41265.0,
  }

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
    }).format(amount)
  }

  const getStatusBadge = (status: string) => {
    const statusConfig = {
      paid: { label: "Paid", variant: "success" as const },
      pending: { label: "Pending", variant: "warning" as const },
      overdue: { label: "Overdue", variant: "destructive" as const },
    }
    const config = statusConfig[status as keyof typeof statusConfig]
    return (
      <Badge variant={config.variant} className="text-xs px-3 py-1">
        {config.label}
      </Badge>
    )
  }

  return (
    <div className="min-h-screen bg-slate-50/50">
      {/* Action Bar */}
      <div className="border-b border-border bg-white sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <Button variant="ghost" className="gap-2">
              <ArrowLeft className="h-4 w-4" />
              Back to Invoices
            </Button>
            <div className="flex items-center gap-3">
              <Button variant="outline" className="gap-2">
                <Download className="h-4 w-4" />
                Download PDF
              </Button>
              <Button variant="outline" className="gap-2">
                <Send className="h-4 w-4" />
                Send Reminder
              </Button>
              <Button variant="default" className="gap-2">
                <CheckCircle2 className="h-4 w-4" />
                Record Payment
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Main Invoice Content */}
      <div className="max-w-5xl mx-auto px-6 py-12">
        <Card className="shadow-lg">
          {/* Invoice Header */}
          <div className="p-12">
            <div className="flex items-start justify-between mb-12">
              {/* Company Logo & Name */}
              <div className="flex items-center gap-4">
                <div className="w-14 h-14 rounded-lg bg-gradient-to-br from-blue-600 to-blue-700 flex items-center justify-center shadow-md">
                  <span className="text-white font-bold text-xl">AS</span>
                </div>
                <div>
                  <div className="font-semibold tracking-tight">
                    {invoiceData.billedFrom.companyName}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {invoiceData.billedFrom.email}
                  </div>
                </div>
              </div>

              {/* Invoice Title & Number */}
              <div className="text-right">
                <div className="text-sm tracking-[0.2em] text-muted-foreground mb-2">
                  INVOICE
                </div>
                <div className="font-bold tracking-tight mb-2">
                  {invoiceData.invoiceNumber}
                </div>
                {getStatusBadge(invoiceData.status)}
              </div>
            </div>

            {/* Billing Information Grid */}
            <div className="grid grid-cols-3 gap-8 mb-12">
              {/* Billed From */}
              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">
                  BILLED FROM
                </div>
                <div className="space-y-1">
                  <div className="font-medium">
                    {invoiceData.billedFrom.companyName}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {invoiceData.billedFrom.address}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {invoiceData.billedFrom.city}
                  </div>
                  <div className="text-sm text-muted-foreground mt-2">
                    {invoiceData.billedFrom.vatId}
                  </div>
                </div>
              </div>

              {/* Billed To */}
              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">
                  BILLED TO
                </div>
                <div className="space-y-1">
                  <div className="font-semibold">
                    {invoiceData.billedTo.companyName}
                  </div>
                  <div className="text-sm">
                    {invoiceData.billedTo.contactName}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {invoiceData.billedTo.address}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {invoiceData.billedTo.city}
                  </div>
                  <div className="text-sm text-muted-foreground mt-2">
                    {invoiceData.billedTo.email}
                  </div>
                </div>
              </div>

              {/* Invoice Details */}
              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">
                  INVOICE DETAILS
                </div>
                <div className="space-y-3">
                  <div>
                    <div className="text-xs text-muted-foreground">
                      Issue Date
                    </div>
                    <div className="text-sm font-medium">
                      {invoiceData.issueDate}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">
                      Due Date
                    </div>
                    <div className="text-sm font-medium">
                      {invoiceData.dueDate}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <Separator className="mb-8" />

            {/* Line Items Table */}
            <div className="mb-8">
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/50 hover:bg-muted/50">
                    <TableHead className="w-[50%]">
                      Item Description
                    </TableHead>
                    <TableHead className="text-center w-[10%]">Qty</TableHead>
                    <TableHead className="text-right w-[20%]">Rate</TableHead>
                    <TableHead className="text-right w-[20%]">
                      Amount
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invoiceData.lineItems.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>
                        <div>
                          <div className="font-medium mb-1">{item.title}</div>
                          <div className="text-sm text-muted-foreground">
                            {item.description}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="text-center">
                        {item.quantity}
                      </TableCell>
                      <TableCell className="text-right">
                        {formatCurrency(item.rate)}
                      </TableCell>
                      <TableCell className="text-right font-medium">
                        {formatCurrency(item.amount)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <Separator className="mb-8" />

            {/* Totals Section */}
            <div className="flex justify-end">
              <div className="w-80 space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Subtotal</span>
                  <span>{formatCurrency(invoiceData.subtotal)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Tax (10%)</span>
                  <span>{formatCurrency(invoiceData.tax)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Discount (5%)</span>
                  <span className="text-destructive">
                    -{formatCurrency(invoiceData.discount)}
                  </span>
                </div>
                <Separator />
                <div className="flex justify-between items-center pt-2">
                  <span className="font-semibold">Total Amount</span>
                  <span className="font-bold">
                    {formatCurrency(invoiceData.total)}
                  </span>
                </div>
              </div>
            </div>

            <Separator className="my-10" />

            {/* Payment Information & Notes */}
            <div className="grid grid-cols-2 gap-8">
              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">
                  PAYMENT INFORMATION
                </div>
                <div className="text-sm space-y-1 text-muted-foreground">
                  <div>Bank: First National Bank</div>
                  <div>Account: ****7890</div>
                  <div>Routing: 021000021</div>
                  <div className="mt-3">
                    Wire transfers typically process within 1-2 business days.
                  </div>
                </div>
              </div>

              <div>
                <div className="text-xs tracking-wide text-muted-foreground mb-3">
                  TERMS & CONDITIONS
                </div>
                <div className="text-sm text-muted-foreground space-y-1">
                  <div>Payment is due within 30 days of invoice date.</div>
                  <div>
                    Late payments may incur a 1.5% monthly interest charge.
                  </div>
                  <div className="mt-3">
                    Please include invoice number in payment reference.
                  </div>
                </div>
              </div>
            </div>

            {/* Footer Message */}
            <div className="mt-12 pt-8 border-t border-border">
              <div className="text-center text-sm text-muted-foreground">
                Thank you for your business! For questions regarding this
                invoice, please contact{" "}
                <span className="text-foreground font-medium">
                  {invoiceData.billedFrom.email}
                </span>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>
  )
}
