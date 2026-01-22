
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
    if (previous === null || previous === undefined) {
      return { text: "No comparison data", color: "text-text-muted" }
    }
    if (growthRate && growthRate > 5) {
      return { text: "Trending up", color: "text-text-brand" }
    }
    if (growthRate && growthRate < -5) {
      return { text: "Down this period", color: "text-text-danger" }
    }
    return { text: "Stable performance", color: "text-text-muted" }
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

  const narrative = getNarrative()

  return (
    <Card className={cn("flex flex-col", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-text-muted">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-1">
          <div className={cn("text-lg font-semibold", narrative.color)}>
            {narrative.text}
          </div>
          <div className="flex items-center gap-2 text-sm text-text-muted">
            {(growthAmount !== null && growthAmount !== undefined) && (
              <span className={cn("font-medium", growthAmount >= 0 ? "text-text-brand" : "text-text-danger")}>
                {formatGrowthValue()}
              </span>
            )}
            {(growthRate !== null && growthRate !== undefined) && (
              <span className={cn("font-medium", growthRate >= 0 ? "text-text-brand" : "text-text-danger")}>
                ({growthRate > 0 ? "+" : ""}{growthRate.toFixed(1)}%)
              </span>
            )}
          </div>
          <div className="text-xs text-text-muted mt-1">
            Compared to previous period
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
