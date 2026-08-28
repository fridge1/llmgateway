import { cn } from "@/lib/utils"

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("animate-shimmer rounded-md", className)}
      style={{ background: "var(--muted)" }}
      {...props}
    />
  )
}

export { Skeleton }
