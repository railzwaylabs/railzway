import type { ReactNode } from "react"

interface StatCardProps {
  label: string
  value: ReactNode
  icon?: ReactNode
  sub?: string
  accentColor?: string
}

export default function StatCard({ label, value, icon, sub, accentColor = "hsl(var(--accent-primary))" }: StatCardProps) {
  return (
    <div className="stat-card">
      <div className="stat-card-header">
        <span className="stat-card-label">{label}</span>
        {icon ? (
          <div className="stat-card-icon" style={{ background: `${accentColor}14`, color: accentColor }}>
            {icon}
          </div>
        ) : null}
      </div>
      <div className="stat-card-value">{value}</div>
      {sub ? <div className="stat-card-sub">{sub}</div> : null}
    </div>
  )
}
