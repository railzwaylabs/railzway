import type { ReactNode } from "react"

interface Column<T> {
  key: string
  label: string
  width?: string
  render?: (row: T) => ReactNode
  className?: string
}

interface DataTableProps<T> {
  columns: Column<T>[]
  data: T[]
  loading?: boolean
  emptyTitle?: string
  emptyDesc?: string
  emptyIcon?: ReactNode
  keyExtractor?: (row: T, index: number) => string
  footer?: ReactNode
}

function DefaultEmptyIcon() {
  return (
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="3" y="3" width="18" height="18" rx="3" />
      <path d="M8 12h8M8 8h8M8 16h4" strokeLinecap="round" />
    </svg>
  )
}

function SkeletonRow({ cols }: { cols: number }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} style={{ padding: "14px 16px" }}>
          <div
            className="skeleton skeleton-text"
            style={{ width: i === 0 ? "60%" : i === cols - 1 ? "40%" : "75%" }}
          />
        </td>
      ))}
    </tr>
  )
}

export default function DataTable<T>({
  columns,
  data,
  loading = false,
  emptyTitle = "No results",
  emptyDesc = "Try adjusting your filters.",
  emptyIcon,
  keyExtractor,
  footer,
}: DataTableProps<T>) {
  const getKey = (row: T, i: number): string => {
    if (keyExtractor) return keyExtractor(row, i)
    if ((row as Record<string, unknown>).id) return String((row as Record<string, unknown>).id)
    return String(i)
  }

  const isEmpty = !loading && data.length === 0

  return (
    <div className="panel">
      <div className="table-wrapper">
        <table className="data-table">
          <thead>
            <tr>
              {columns.map((col) => (
                <th key={col.key} style={col.width ? { width: col.width } : undefined}>
                  {col.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading && data.length === 0
              ? Array.from({ length: 5 }).map((_, i) => <SkeletonRow key={i} cols={columns.length} />)
              : null}
            {!loading && isEmpty ? (
              <tr>
                <td colSpan={columns.length} style={{ padding: 0, border: "none" }}>
                  <div className="empty-state">
                    <div className="empty-state-icon">{emptyIcon ?? <DefaultEmptyIcon />}</div>
                    <p className="empty-state-title">{emptyTitle}</p>
                    <p className="empty-state-desc">{emptyDesc}</p>
                  </div>
                </td>
              </tr>
            ) : null}
            {data.map((row, i) => (
              <tr key={getKey(row, i)}>
                {columns.map((col) => (
                  <td key={col.key} className={col.className}>
                    {col.render ? col.render(row) : String((row as Record<string, unknown>)[col.key] ?? "—")}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {footer ? <div className="load-more-row">{footer}</div> : null}
    </div>
  )
}
