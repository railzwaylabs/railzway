import { Badge } from "@/components/ui/badge"
import { Clock, AlertTriangle } from "lucide-react"
import { cn } from "@/lib/utils"
import type { SLAStatus } from "../types/ia-types"

interface SLABadgeProps {
  status: SLAStatus
  minutesRemaining?: number
  className?: string
}

export function SLABadge({ status, minutesRemaining, className }: SLABadgeProps) {
  const config = getSLAConfig(status)

  return (
    <Badge
      variant="outline"
      className={cn(
        "flex items-center gap-1.5",
        config.className,
        className
      )}
    >
      {config.icon}
      <span>{config.label}</span>
      {minutesRemaining !== undefined && minutesRemaining > 0 && (
        <span className="text-[10px] opacity-70">({minutesRemaining}m)</span>
      )}
    </Badge>
  )
}

function getSLAConfig(status: SLAStatus) {
  switch (status) {
    case "fresh":
      return {
        label: "Fresh",
        icon: <Clock className="h-3 w-3" />,
        className: "border-emerald-500/30 bg-emerald-500/10 text-emerald-600",
      }
    case "active":
      return {
        label: "Active",
        icon: <Clock className="h-3 w-3" />,
        className: "border-blue-500/30 bg-blue-500/10 text-blue-600",
      }
    case "aging":
      return {
        label: "Aging",
        icon: <Clock className="h-3 w-3" />,
        className: "border-amber-500/30 bg-amber-500/10 text-amber-600",
      }
    case "stale":
      return {
        label: "Stale",
        icon: <AlertTriangle className="h-3 w-3" />,
        className: "border-red-500/30 bg-red-500/10 text-red-600 font-semibold",
      }
    case "breached":
    case "resolved":
      return {
        label: status === "breached" ? "Escalated" : "Resolved",
        icon: <AlertTriangle className="h-3 w-3" />,
        className: "border-red-500/30 bg-red-500/10 text-red-600 font-bold animate-pulse",
      }
    default:
      return {
        label: "Unknown",
        icon: <Clock className="h-3 w-3" />,
        className: "border-muted-foreground/30 bg-muted text-muted-foreground",
      }
  }
}
