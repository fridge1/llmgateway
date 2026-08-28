import * as React from "react"
import { Moon, Sun } from "lucide-react"
import { useTheme } from "next-themes"
import { cn } from "@/lib/utils"

/**
 * 共享导航原子组件。统一三套布局（用户/子用户/管理后台）的导航视觉，
 * 消除重复手写的主题切换、Logo、图标按钮，保证尺寸/交互一致。
 */

/** 主题切换按钮——统一尺寸(w-9 h-9)、圆角、hover 态。 */
function ThemeToggle({ size = 16, className }: { size?: number; className?: string }) {
  const { theme, setTheme } = useTheme()
  return (
    <button
      onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      className={cn(
        "w-9 h-9 flex items-center justify-center rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors duration-150 cursor-pointer",
        className,
      )}
      title={theme === "dark" ? "切换到浅色模式" : "切换到深色模式"}
      aria-label="切换主题"
    >
      {theme === "dark" ? <Sun size={size} /> : <Moon size={size} />}
    </button>
  )
}

/** 品牌渐变方块 Logo——统一尺寸、圆角、阴影。 */
function BrandLogo({
  icon: Icon,
  label,
  gradient,
  size = "md",
  onClick,
}: {
  icon: React.ComponentType<{ className?: string; size?: number }>
  label?: string
  gradient?: string
  size?: "sm" | "md"
  onClick?: () => void
}) {
  const box = size === "sm" ? "w-6 h-6 rounded-md" : "w-8 h-8 rounded-xl"
  const iconSize = size === "sm" ? 12 : 15
  return (
    <div
      className={cn("flex items-center gap-2.5 cursor-pointer select-none shrink-0", onClick && "cursor-pointer")}
      onClick={onClick}
    >
      <div
        className={cn(box, "flex items-center justify-center shadow-button")}
        style={gradient ? { background: gradient } : undefined}
      >
        {gradient ? (
          <Icon size={iconSize} className="text-white" />
        ) : (
          <div className="brand-gradient w-full h-full rounded-xl flex items-center justify-center">
            <Icon size={iconSize} className="text-white" />
          </div>
        )}
      </div>
      {label && <span className="font-bold text-sm text-foreground tracking-tight hidden sm:inline">{label}</span>}
    </div>
  )
}

/** 导航图标按钮——统一尺寸、圆角、hover 态（客服/退出/通知等）。 */
function NavIconButton({
  icon: Icon,
  onClick,
  title,
  label,
  hover = "default",
  size = 16,
  className,
}: {
  icon: React.ComponentType<{ className?: string; size?: number }>
  onClick?: () => void
  title: string
  label?: string
  hover?: "default" | "destructive"
  size?: number
  className?: string
}) {
  return (
    <button
      onClick={onClick}
      title={title}
      aria-label={label ?? title}
      className={cn(
        "w-9 h-9 flex items-center justify-center rounded-lg text-muted-foreground transition-colors duration-150 cursor-pointer",
        hover === "destructive"
          ? "hover:text-destructive hover:bg-destructive/10"
          : "hover:text-foreground hover:bg-muted/60",
        className,
      )}
    >
      <Icon size={size} />
    </button>
  )
}

export { ThemeToggle, BrandLogo, NavIconButton }
