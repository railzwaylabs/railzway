
import { useRouteError, isRouteErrorResponse, useNavigate } from "react-router-dom"
import { ShieldAlert, RefreshCw, ChevronLeft, WifiOff, FileWarning, ServerCrash } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"

// --- Components ---

function ErrorActionButtons() {
  const navigate = useNavigate()
  return (
    <div className="flex flex-col sm:flex-row gap-3 pt-2">
      <Button
        onClick={() => window.location.reload()}
        className="gap-2"
      >
        <RefreshCw className="h-4 w-4" />
        Reload page
      </Button>
      <Button
        variant="outline"
        onClick={() => navigate("/orgs")}
        className="gap-2"
      >
        <ChevronLeft className="h-4 w-4" />
        Go back to dashboard
      </Button>
    </div>
  )
}

function ErrorLayout({
  icon: Icon,
  title,
  message,
  footer
}: {
  icon: any,
  title: string,
  message: string,
  footer?: string
}) {
  return (
    <div className="min-h-[60vh] w-full flex items-center justify-center p-4">
      <Card className="max-w-md w-full shadow-lg border-border-subtle bg-bg-surface">
        <CardHeader className="text-center pb-2">
          <div className="mx-auto bg-bg-subtle/50 p-3 rounded-full w-fit mb-4">
            <Icon className="h-8 w-8 text-text-secondary" />
          </div>
          <CardTitle className="text-xl font-semibold text-text-primary">
            {title}
          </CardTitle>
        </CardHeader>
        <CardContent className="text-center space-y-6">
          <p className="text-text-muted text-sm leading-relaxed">
            {message}
          </p>
          <ErrorActionButtons />
        </CardContent>
        {footer && (
          <CardFooter className="bg-bg-subtle/20 border-t border-border-subtle py-3 justify-center">
            <p className="text-xs text-text-muted text-center max-w-xs">{footer}</p>
          </CardFooter>
        )}
      </Card>
    </div>
  )
}

// --- Logic ---

export function EnterpriseErrorBoundary() {
  const error = useRouteError() as any

  // 1. Chunk / Dynamic Import Error
  // "Failed to fetch dynamically imported module" usually means a deployment happened
  if (error?.message?.includes("Failed to fetch dynamically imported module") || error?.name === "ChunkLoadError") {
    return (
      <ErrorLayout
        icon={RefreshCw}
        title="Update Available"
        message="A new version of Railzway has been deployed. Please reload the page to apply the update and continue working."
        footer="Your session and data are preserved."
      />
    )
  }

  // 2. Network Error
  if (error?.message === "Failed to fetch" || error?.code === "ERR_NETWORK") {
    return (
      <ErrorLayout
        icon={WifiOff}
        title="Connection Issue"
        message="We couldn't connect to the server. Please check your internet connection and try again."
      />
    )
  }

  // 3. Permission Error (403)
  if (isRouteErrorResponse(error) && error.status === 403) {
    return (
      <ErrorLayout
        icon={ShieldAlert}
        title="Access Restricted"
        message="You do not have permission to view this page. If you believe this is a mistake, please contact your organization administrator."
        footer="Reference: ERR_PERM_DENIED"
      />
    )
  }

  // 4. Server Error (5xx)
  if (isRouteErrorResponse(error) && error.status >= 500) {
    return (
      <ErrorLayout
        icon={ServerCrash}
        title="Service Unavailable"
        message="We encountered a temporary issue on our end. Our engineering team has been notified. Please try again in a few moments."
        footer={`Reference: ${error.status} ${error.statusText}`}
      />
    )
  }

  // 5. Generic / Unhandled / React Render Error
  return (
    <ErrorLayout
      icon={FileWarning}
      title="We couldn't load this page"
      message="This may be caused by a temporary loading issue or a recent update. Your billing data and configuration are safe."
      footer="If the problem persists, contact your administrator or Railzway support."
    />
  )
}
