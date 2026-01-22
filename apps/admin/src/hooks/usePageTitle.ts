import { useEffect } from "react"

type PageTitleOptions = {
  page?: string
  org?: string
}

const BASE_TITLE = "Railzway"
const DEFAULT_TITLE = "Railzway Dashboard"

const buildTitle = ({ page, org }: PageTitleOptions = {}): string => {
  if (page) {
    return org ? `${page} · ${org} · ${BASE_TITLE}` : `${page} · ${BASE_TITLE}`
  }
  return DEFAULT_TITLE
}

export function usePageTitle(options?: PageTitleOptions) {
  const page = options?.page
  const org = options?.org

  useEffect(() => {
    const title = buildTitle({ page, org })
    if (document.title !== title) {
      document.title = title
    }
  }, [page, org])
}
