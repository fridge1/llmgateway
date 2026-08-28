import Card from "./Card";
import Skeleton from "./Skeleton";

const accentStyles = {
  amber: { bg: "bg-amber-950/30", icon: "text-amber-400" },
  emerald: { bg: "bg-emerald-950/30", icon: "text-emerald-400" },
  blue: { bg: "bg-blue-950/30", icon: "text-blue-400" },
} as const;

interface StatCardProps {
  icon: React.ComponentType<{ className?: string; size?: number }>;
  title: string;
  value: string;
  subtitle?: string;
  loading?: boolean;
  accentColor?: "amber" | "emerald" | "blue";
}

export default function StatCard({
  icon: Icon,
  title,
  value,
  subtitle,
  loading = false,
  accentColor = "amber",
}: StatCardProps) {
  const styles = accentStyles[accentColor];

  return (
    <Card hover={true}>
      <div className={`w-9 h-9 rounded-lg ${styles.bg} flex items-center justify-center`}>
        <Icon size={18} className={styles.icon} />
      </div>
      <p className="text-xs uppercase tracking-wider text-obsidian-400 font-medium mt-3">
        {title}
      </p>
      {loading ? (
        <>
          <Skeleton.Text className="w-20 mt-2" />
          <Skeleton.Text className="w-16 mt-1" />
        </>
      ) : (
        <>
          <p className="text-2xl font-bold text-obsidian-50 font-mono mt-1">{value}</p>
          {subtitle && <p className="text-xs text-obsidian-500 mt-1">{subtitle}</p>}
        </>
      )}
    </Card>
  );
}
