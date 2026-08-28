"use client"

import * as React from "react"
import { Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"

/**
 * 统一加载状态。替代散落的"加载中..."纯文字。
 * - inline：行内 spinner + 文案（默认，用于卡片/区块内）
 * - block：整块居中（用于整页加载）
 */
function LoadingState({
  text = "加载中…",
  inline = false,
  className,
}: {
  text?: string
  inline?: boolean
  className?: string
}) {
  if (inline) {
    return (
      <span className={cn("inline-flex items-center gap-2 text-sm text-muted-foreground", className)}>
        <Loader2 className="h-4 w-4 animate-spin" />
        {text}
      </span>
    )
  }
  return (
    <div className={cn("flex flex-col items-center justify-center py-12 text-muted-foreground", className)}>
      <Loader2 className="h-6 w-6 animate-spin mb-2" />
      <p className="text-sm">{text}</p>
    </div>
  )
}

export { LoadingState }
