interface SkeletonProps {
  className?: string;
}

function Skeleton({ className = "" }: SkeletonProps) {
  return <div className={`bg-obsidian-800 rounded-md animate-shimmer ${className}`} />;
}

Skeleton.Text = function SkeletonText({ className = "w-full" }: SkeletonProps) {
  return <Skeleton className={`h-3.5 rounded-sm ${className}`} />;
};

Skeleton.Title = function SkeletonTitle({ className = "w-48" }: SkeletonProps) {
  return <Skeleton className={`h-6 rounded-md ${className}`} />;
};

Skeleton.Card = function SkeletonCard({ className = "" }: SkeletonProps) {
  return <Skeleton className={`h-[120px] rounded-xl ${className}`} />;
};

Skeleton.Circle = function SkeletonCircle({ className = "w-10 h-10" }: SkeletonProps) {
  return <Skeleton className={`rounded-full ${className}`} />;
};

export default Skeleton;
