import { Badge } from "@/components/ui/badge"
import { useSystemPlan } from "@/stores/capabilityStore"
import { cn } from "@/lib/utils"

interface LicenseStatusBadgeProps {
  className?: string
}

export function LicenseStatusBadge({ className }: LicenseStatusBadgeProps) {
  const plan = useSystemPlan()

  if (plan === "plus") {
    return (
      <Badge
        variant="outline"
        className={cn(
          "bg-indigo-50 text-indigo-700 border-indigo-200 hover:bg-indigo-50 hover:text-indigo-700", // Neutral but distinct for Plus
          className
        )}
      >
        Plus
      </Badge>
    )
  }

  // OSS Badge - Minimalist/Neutral
  return (
    <Badge
      variant="outline"
      className={cn("bg-muted/50 text-muted-foreground border-border-subtle", className)}
    >
      OSS
    </Badge>
  )
}
