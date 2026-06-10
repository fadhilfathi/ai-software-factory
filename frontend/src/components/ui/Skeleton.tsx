import { cn } from "@/lib/utils";

type SkeletonProps = {
  className?: string;
};

export function Skeleton({ className }: SkeletonProps) {
  return (
    <div
      className={cn("animate-pulse rounded bg-gray-800", className)}
    />
  );
}

Skeleton.Card = function SkeletonCard({ className }: SkeletonProps) {
  return (
    <div
      className={cn(
        "animate-pulse rounded-lg border border-gray-800 bg-gray-950 p-4",
        className,
      )}
    >
      <div className="h-5 w-2/3 rounded bg-gray-800" />
      <div className="mt-3 h-2 rounded-full bg-gray-800" />
      <div className="mt-4 flex justify-between">
        <div className="h-3 w-16 rounded bg-gray-800" />
        <div className="h-3 w-12 rounded bg-gray-800" />
      </div>
    </div>
  );
};

Skeleton.CardGrid = function SkeletonCardGrid({
  count = 6,
  className,
}: SkeletonProps & { count?: number }) {
  return (
    <div
      className={cn("grid gap-4 sm:grid-cols-2 lg:grid-cols-3", className)}
    >
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton.Card key={i} />
      ))}
    </div>
  );
};

Skeleton.List = function SkeletonList({
  count = 3,
  className,
}: SkeletonProps & { count?: number }) {
  return (
    <div className={cn("space-y-3", className)}>
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className="animate-pulse rounded-lg border border-gray-800 bg-gray-950 p-3"
        >
          <div className="h-4 w-3/4 rounded bg-gray-800" />
          <div className="mt-2 h-3 w-1/4 rounded bg-gray-800" />
        </div>
      ))}
    </div>
  );
};

Skeleton.Activity = function SkeletonActivity({
  count = 3,
  className,
}: SkeletonProps & { count?: number }) {
  return (
    <div className={cn("space-y-3", className)}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="animate-pulse flex items-start gap-3">
          <div className="h-5 w-5 rounded-full bg-gray-800" />
          <div className="flex-1 space-y-1.5">
            <div className="h-3 w-3/4 rounded bg-gray-800" />
            <div className="h-2 w-1/4 rounded bg-gray-800" />
          </div>
        </div>
      ))}
    </div>
  );
};

Skeleton.Metrics = function SkeletonMetrics({
  count = 4,
  className,
}: SkeletonProps & { count?: number }) {
  return (
    <div className={cn("grid grid-cols-2 gap-4 md:grid-cols-4", className)}>
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className="animate-pulse rounded-lg border border-gray-800 bg-gray-950 p-4"
        >
          <div className="h-3 w-16 rounded bg-gray-800" />
          <div className="mt-2 h-6 w-20 rounded bg-gray-800" />
        </div>
      ))}
    </div>
  );
};
