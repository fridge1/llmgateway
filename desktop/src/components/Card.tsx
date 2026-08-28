import { motion } from "motion/react";

interface CardProps {
  children: React.ReactNode;
  className?: string;
  hover?: boolean;
  glow?: boolean;
  padding?: "sm" | "md" | "lg";
}

const paddingMap = {
  sm: "p-3",
  md: "p-4",
  lg: "p-6",
} as const;

export default function Card({
  children,
  className = "",
  hover = true,
  glow = false,
  padding = "md",
}: CardProps) {
  const base = `bg-obsidian-900 border border-obsidian-700 rounded-xl shadow-card ${paddingMap[padding]}`;
  const hoverClasses = hover
    ? `hover:shadow-card-hover ${glow ? "hover:shadow-glow-amber" : ""}`
    : "";
  const classes = `${base} ${hoverClasses} ${className}`.trim();

  if (!hover) {
    return <div className={classes}>{children}</div>;
  }

  return (
    <motion.div
      className={classes}
      whileHover={{ y: -1 }}
      transition={{ duration: 0.2 }}
    >
      {children}
    </motion.div>
  );
}
