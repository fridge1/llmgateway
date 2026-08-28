import * as React from "react"
import { ArrowLeft } from "lucide-react"
import { cn } from "@/lib/utils"

/**
 * 统一页面标题区。替代 50+ 页面散落手写的 <div className="mb-6"><h1/><p/></div>。
 * 固定标题字号、副标题、间距与右侧操作区，保证全站页眉视觉一致。
 */
function PageHeader({
  title,
  description,
  actions,
  eyebrow,
  backAction,
  backLabel = "返回上一级",
  className,
}: {
  title: React.ReactNode
  description?: React.ReactNode
  actions?: React.ReactNode
  eyebrow?: React.ReactNode
  backAction?: () => void
  backLabel?: string
  className?: string
}) {
  return (
    <div
      className={cn(
        "relative flex flex-col gap-4 overflow-hidden rounded-2xl border border-border/70 bg-card/70 p-5 mb-6 glass sm:flex-row sm:items-start sm:justify-between",
        className,
      )}
    >
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-12 -top-16 h-36 w-36 rounded-full bg-primary/10 blur-2xl"
      />
      <div className="flex min-w-0 items-start gap-3">
        {backAction && (
          <button
            onClick={backAction}
            className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-border/70 text-muted-foreground transition-colors duration-150 hover:bg-muted hover:text-foreground cursor-pointer"
            aria-label={backLabel}
            title={backLabel}
          >
            <ArrowLeft size={16} />
          </button>
        )}
        <div className="min-w-0">
          {eyebrow && (
            <div className="mb-1 text-xs font-semibold uppercase tracking-[0.12em] text-primary">
              {eyebrow}
            </div>
          )}
          <h1 className="text-2xl font-bold leading-tight text-foreground">
            {title}
          </h1>
          {description && (
            <p className="max-w-3xl text-sm leading-relaxed text-muted-foreground mt-1">
              {description}
            </p>
          )}
        </div>
      </div>
      {actions && (
        <div className="flex flex-wrap items-center justify-end gap-2 shrink-0">
          {actions}
        </div>
      )}
    </div>
  )
}

export { PageHeader }
