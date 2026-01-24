import { Diamond, ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Link } from "react-router-dom"

interface FeatureLockedStateProps {
  title?: string
  description?: string
}

export function FeatureLockedState({
  title = "This feature is available in Railzway Plus",
  description = "Upgrade your license to unlock advanced analytics, team workflows, and more."
}: FeatureLockedStateProps) {
  return (
    <div className="h-full w-full min-h-[400px] flex items-center justify-center p-6">
      <Card className="max-w-md w-full border-dashed shadow-sm">
        <CardContent className="flex flex-col items-center text-center p-10 space-y-6">
          <div className="h-16 w-16 bg-gradient-to-br from-indigo-500/20 to-purple-500/20 rounded-full flex items-center justify-center mb-2">
            <Diamond className="h-8 w-8 text-indigo-600" />
          </div>

          <div className="space-y-2">
            <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
            <p className="text-sm text-muted-foreground leading-relaxed">
              {description}
            </p>
          </div>

          <Button asChild className="gap-2" variant="default">
            <Link to="/orgs/current/settings/license">
              View License Options
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
