import type { ReactNode } from "react"

interface PageHeaderProps {
  icon?: ReactNode
  title: string
  description?: string
  actions?: ReactNode
}

export default function PageHeader({ icon, title, description, actions }: PageHeaderProps) {
  return (
    <div className="page-header">
      <div className="page-header-left">
        {icon ? <div className="page-header-icon">{icon}</div> : null}
        <div className="page-header-text">
          <h1 className="page-header-title" data-testid="page-header-title">{title}</h1>
          {description ? <p className="page-header-desc" data-testid="page-header-description">{description}</p> : null}
        </div>
      </div>
      {actions ? <div className="page-header-actions">{actions}</div> : null}
    </div>
  )
}
