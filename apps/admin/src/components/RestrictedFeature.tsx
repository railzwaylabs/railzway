import { type ReactNode } from "react"
import { IconLock } from "@tabler/icons-react"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useCapability } from "@/stores/capabilityStore"
import type { SystemFeatures } from "@/api/capabilities"
import { cn } from "@/lib/utils"

interface RestrictedFeatureProps {
  feature: keyof SystemFeatures
  children: ReactNode
  fallback?: ReactNode
  className?: string
  description?: string // Override default "Available in Plus" message
}

export function RestrictedFeature({
  feature,
  children,
  fallback,
  className,
  description,
}: RestrictedFeatureProps) {
  const isEnabled = useCapability(feature)

  if (isEnabled) {
    return <>{children}</>
  }

  if (fallback) {
    return <>{fallback}</>
  }

  // Default behavior: Render children but disabled/locked appearance
  // We wrap in a div to capture events if necessary, or just style the children.
  // Ideally, the children should support a 'disabled' prop, but we can't enforce that here.
  // Instead, we render a lock overlay or tooltip wrapper.

  return (
    <TooltipProvider>
      <Tooltip delayDuration={200}>
        <TooltipTrigger asChild>
          <div className={cn("relative opacity-50 grayscale cursor-not-allowed select-none", className)}>
            {/* Overlay to block interactions */}
            <div className="absolute inset-0 z-10 bg-transparent" />
            {children}
          </div>
        </TooltipTrigger>
        <TooltipContent side="top" className="bg-bg-surface-strong border-border-subtle text-text-primary">
          <p className="flex items-center gap-2 text-sm">
            <IconLock className="h-3.5 w-3.5" />
            {description || "This feature is available in Railzway Plus."}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
