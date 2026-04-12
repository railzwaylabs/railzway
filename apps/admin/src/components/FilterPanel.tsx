import { type ReactNode, useState } from "react"

interface FilterPanelProps {
  children: ReactNode
  actions?: ReactNode
  defaultOpen?: boolean
  title?: string
  count?: number
}

function ChevronIcon({ open }: { open: boolean }) {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      className={`filter-panel-chevron${open ? " open" : ""}`}
    >
      <path d="M4 6l4 4 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function FilterIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 4h12M5 8h6M8 12h0" strokeLinecap="round" />
    </svg>
  )
}

export default function FilterPanel({
  children,
  actions,
  defaultOpen = true,
  title = "Filters",
  count,
}: FilterPanelProps) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="filter-panel">
      <div className="filter-panel-header" onClick={() => setOpen((o) => !o)} role="button" tabIndex={0}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") setOpen((o) => !o) }}>
        <div className="filter-panel-title">
          <FilterIcon />
          {title}
          {count !== undefined && count > 0 ? (
            <span
              style={{
                background: "hsl(var(--accent-primary))",
                color: "hsl(var(--text-inverse))",
                borderRadius: "999px",
                fontSize: "10px",
                fontWeight: 700,
                padding: "1px 7px",
              }}
            >
              {count}
            </span>
          ) : null}
        </div>
        <ChevronIcon open={open} />
      </div>
      {open ? (
        <div className="filter-panel-body">
          {children}
          {actions ? <div className="filter-actions">{actions}</div> : null}
        </div>
      ) : null}
    </div>
  )
}
