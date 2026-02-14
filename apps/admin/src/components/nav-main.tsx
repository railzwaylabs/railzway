import type { LucideIcon } from "lucide-react"
import { Link, useLocation } from "react-router-dom"

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

export function NavMain({
  items,
}: {
  items: {
    title: string
    url: string
    icon?: LucideIcon
    isActive?: (pathname: string) => boolean
  }[]
}) {
  const { pathname } = useLocation()

  const normalizePath = (value: string) => {
    const trimmed = value.replace(/\/+$/, "")
    return trimmed.length ? trimmed : "/"
  }
  const currentPath = normalizePath(pathname)

  const resolveActive = (item: (typeof items)[number]) => {
    if (item.isActive) return item.isActive(currentPath)
    const target = normalizePath(item.url)
    if (currentPath === target) return true
    return currentPath.startsWith(`${target}/`)
  }

  return (
    <SidebarGroup>
      <SidebarGroupContent className="flex flex-col gap-2">
        <SidebarMenu>
          {items.map((item) => {
            const isActive = resolveActive(item)
            return (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton asChild tooltip={item.title} isActive={isActive}>
                  <Link to={item.url} aria-current={isActive ? "page" : undefined}>
                    {item.icon && <item.icon />}
                    <span>{item.title}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
