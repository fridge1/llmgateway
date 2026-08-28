interface BadgeProps {
  variant: "success" | "warning" | "error" | "info" | "neutral";
  children: React.ReactNode;
  dot?: boolean;
}

const variantStyles = {
  success: { badge: "bg-emerald-950/50 text-emerald-400", dot: "bg-emerald-500" },
  warning: { badge: "bg-amber-950/50 text-amber-400", dot: "bg-amber-500" },
  error: { badge: "bg-red-950/50 text-red-400", dot: "bg-red-500" },
  info: { badge: "bg-blue-950/50 text-blue-400", dot: "bg-blue-500" },
  neutral: { badge: "bg-obsidian-800 text-obsidian-400", dot: "bg-obsidian-400" },
} as const;

export default function Badge({ variant, children, dot = false }: BadgeProps) {
  const styles = variantStyles[variant];

  return (
    <span
      className={`px-2.5 py-0.5 text-xs rounded-full font-medium inline-flex items-center gap-1.5 ${styles.badge}`}
    >
      {dot && <span className={`w-1.5 h-1.5 rounded-full ${styles.dot}`} />}
      {children}
    </span>
  );
}
