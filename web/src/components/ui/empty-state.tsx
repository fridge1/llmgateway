"use client"

import * as React from "react"
import { cn } from "@/lib/utils"

/**
 * 统一空状态组件。替代各页面散落的"暂无数据"纯文字。
 * 提供图标 + 标题 + 可选描述/操作，遵循空状态引导原则。
 */
function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: {
  icon?: React.ComponentType<{ className?: string }>
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center py-14 px-4",
        className,
      )}
    >
      {Icon && (
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted mb-4">
          <Icon className="h-7 w-7 text-muted-foreground/50" />
        </div>
      )}
      <p className="text-sm font-medium text-foreground">{title}</p>
      {description && (
        <p className="text-xs text-muted-foreground mt-1 max-w-xs leading-relaxed">{description}</p>
      )}
      {action && <div className="mt-5">{action}</div>}
    </div>
  )
}

export { EmptyState }
