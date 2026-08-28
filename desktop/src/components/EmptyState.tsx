interface EmptyStateProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
}

export default function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16">
      <div className="w-16 h-16 rounded-2xl bg-obsidian-850 flex items-center justify-center mb-4">
        <Icon className="w-8 h-8 text-obsidian-600 animate-pulse-subtle" />
      </div>
      <p className="text-sm font-medium text-obsidian-200">{title}</p>
      {description && <p className="text-xs text-obsidian-400 mt-1">{description}</p>}
      {action && (
        <button
          onClick={action.onClick}
          className="mt-4 px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-colors"
        >
          {action.label}
        </button>
      )}
    </div>
  );
}
