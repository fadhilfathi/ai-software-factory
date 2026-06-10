import { cn } from "@/lib/utils";

type PaginationInfoProps = {
  total: number;
  page?: number;
  pages?: number;
  showing: number;
  className?: string;
};

export function PaginationInfo({
  total,
  page,
  pages,
  showing,
  className,
}: PaginationInfoProps) {
  if (total === 0) return null;

  return (
    <p className={cn("mt-4 text-center text-xs text-gray-600", className)}>
      Showing {showing} of {total}
      {pages && pages > 1 && ` (page ${page ?? 1} of ${pages})`}
    </p>
  );
}
