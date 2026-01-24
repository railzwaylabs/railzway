import { useEffect } from "react"
import { useCapabilityStore } from "@/stores/capabilityStore"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Card, CardContent } from "@/components/ui/card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { IconCheck, IconX, IconAlertCircle } from "@tabler/icons-react"
import { cn } from "@/lib/utils"

export default function LicensePage() {
  const { capabilities, fetchCapabilities, isLoading, error } = useCapabilityStore()

  useEffect(() => {
    void fetchCapabilities()
  }, [fetchCapabilities])

  if (isLoading) {
    return <div className="p-8 text-text-muted text-sm">Loading license information...</div>
  }

  if (error) {
    return (
      <Alert variant="destructive" className="max-w-2xl mt-4">
        <IconAlertCircle className="h-4 w-4" />
        <AlertTitle>Error</AlertTitle>
        <AlertDescription>
          Unable to retrieve license information. Please check your connection or contact your administrator.
        </AlertDescription>
      </Alert>
    )
  }

  if (!capabilities) {
    return null
  }

  const features = [
    { key: "sso", label: "Enterprise SSO" },
    { key: "rbac", label: "Role-Based Access Control (RBAC)" },
    { key: "audit_export", label: "Audit Log Export" },
    { key: "forecasting", label: "Revenue Forecasting" },
  ]

  const featureList = features.map((feat) => ({
    ...feat,
    enabled: capabilities.features[feat.key] ?? false,
  }))

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    if (date.getFullYear() <= 1 || isNaN(date.getTime())) return "Never"
    return new Intl.DateTimeFormat("en-GB", {
      day: "numeric",
      month: "long",
      year: "numeric",
    }).format(date)
  }

  const isPlus = capabilities.plan === "plus"
  const isExpired = isPlus && new Date(capabilities.expires_at).getTime() < Date.now()

  return (
    <div className="space-y-8 max-w-3xl pb-10">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold">License</h1>
        <p className="text-text-muted text-sm">
          View the current license state and enabled capabilities for this instance.
        </p>
      </div>

      {/* Section 1: License Summary */}
      <Card>
        <CardContent className="p-6">
          <div className="flex flex-col gap-1">
            <h3 className="font-medium text-text-primary">
              This instance is running Railzway {isPlus ? "Plus" : "OSS"}.
            </h3>
            {isPlus ? (
              <p className={cn("text-sm", isExpired ? "text-destructive" : "text-text-muted")}>
                {isExpired
                  ? `License expired on ${formatDate(capabilities.expires_at)}.`
                  : `License valid until ${formatDate(capabilities.expires_at)}.`}
              </p>
            ) : (
              <p className="text-sm text-text-muted">
                Open source edition.
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Section 2: Enabled Capabilities */}
      <div className="space-y-4">
        <h3 className="text-sm font-medium text-text-primary px-1">Enabled Capabilities</h3>
        <div className="rounded-lg border border-border-subtle bg-bg-surface overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent bg-bg-subtle/50">
                <TableHead className="w-[300px]">Feature</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {featureList.map((feature) => (
                <TableRow key={feature.key}>
                  <TableCell className="font-medium text-sm text-text-primary">
                    {feature.label}
                  </TableCell>
                  <TableCell>
                    {feature.enabled ? (
                      <div className="flex items-center gap-2 text-text-primary">
                        <IconCheck className="w-4 h-4 text-success-600" />
                        <span className="text-sm">Enabled</span>
                      </div>
                    ) : (
                      <div className="flex flex-col gap-1 py-1">
                        <div className="flex items-center gap-2 text-text-muted">
                          <IconX className="w-4 h-4" />
                          <span className="text-sm">Not enabled</span>
                        </div>
                        <span className="text-xs text-text-muted/75 pl-6">
                          This feature is available in Railzway Plus.
                        </span>
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* Section 3: Operational Notes */}
      <div className="rounded-md bg-bg-subtle p-4 border border-border-subtle">
        <p className="text-xs text-text-muted leading-relaxed">
          This page reflects the current license state of this Railzway instance.
          License validation is performed locally and does not require an internet connection.
        </p>
      </div>
    </div>
  )
}
