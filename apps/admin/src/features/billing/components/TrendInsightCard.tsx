
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

interface TrendInsightCardProps {
  title: string
  previous?: number | null
  growthRate?: number | null
  growthAmount?: number | null
  currency?: string
  type?: "currency" | "number"
  className?: string
}

export function TrendInsightCard({
  title,
  previous,
  growthRate,
  growthAmount,
  currency = "USD",
  type = "number",
  className,
}: TrendInsightCardProps) {
  const getNarrative = () => {
    const hasPrevious = previous !== null && previous !== undefined
    if (!hasPrevious && (growthRate === null || growthRate === undefined)) {
      if (growthAmount === null || growthAmount === undefined) {
        return { text: "No prior period", color: "text-text-muted" }
      }
    }
    const delta = growthRate ?? growthAmount ?? 0
    const threshold = growthRate !== null && growthRate !== undefined ? 2 : 0
    if (delta > threshold) {
      return { text: "Up vs last period", color: "text-status-success" }
    }
    if (delta < -threshold) {
      return { text: "Down vs last period", color: "text-status-error" }
    }
    return { text: "Mostly flat", color: "text-text-muted" }
  }

  const formatValue = (val: number) => {
    if (type === "currency") {
      return new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: currency,
        maximumFractionDigits: 0,
      }).format(val)
    }
    return new Intl.NumberFormat("en-US").format(val)
  }

  const formatGrowthValue = () => {
    if (growthAmount === null || growthAmount === undefined) return null
    const val = formatValue(Math.abs(growthAmount))
    const prefix = growthAmount >= 0 ? "+" : "-"
    return `${prefix}${val}`
  }

  const normalizeRate = (rate: number) => {
    if (!Number.isFinite(rate)) return 0
    return Math.abs(rate) < 0.05 ? 0 : rate
  }

  const narrative = getNarrative()
  const normalizedGrowthRate =
    growthRate === null || growthRate === undefined ? null : normalizeRate(growthRate)

  return (
    <Card className={cn("flex flex-col transition-all duration-300 hover:shadow-md hover:border-border-strong group", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="text-[10px] font-bold text-text-muted uppercase tracking-widest opacity-70 group-hover:opacity-100 transition-opacity">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-1">
          <div className={cn("text-xl font-bold tracking-tight", narrative.color)}>
            {narrative.text}
          </div>
          <div className="flex items-center gap-2 text-sm text-text-muted mt-0.5">
            {(growthAmount !== null && growthAmount !== undefined) && (
              <span className={cn("font-bold", growthAmount >= 0 ? "text-status-success" : "text-status-error")}>
                {formatGrowthValue()}
              </span>
            )}
            {normalizedGrowthRate !== null && (
              <span className={cn("font-bold px-2 py-0.5 rounded-full text-[10px] bg-bg-primary border border-border-subtle shadow-sm", normalizedGrowthRate >= 0 ? "text-status-success" : "text-status-error")}>
                {normalizedGrowthRate > 0 ? "+" : ""}{normalizedGrowthRate.toFixed(1)}%
              </span>
            )}
          </div>
          <div className="text-[10px] font-medium text-text-muted mt-2 opacity-60">
            Compared with previous period
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
