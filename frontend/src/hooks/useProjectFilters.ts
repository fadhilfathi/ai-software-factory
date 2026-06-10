"use client";

import { useSearchParams, useRouter, usePathname } from "next/navigation";
import { useCallback, useMemo } from "react";

type FilterState = Record<string, string | undefined>;

/**
 * Hook that syncs page filters with URL search params.
 * URL is the canonical source — this hook both reads and writes it.
 */
export function useProjectFilters(defaultFilters: FilterState = {}) {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();

  const filters: FilterState = useMemo(() => {
    const result: FilterState = { ...defaultFilters };
    for (const [key, value] of searchParams.entries()) {
      result[key] = value;
    }
    return result;
  }, [searchParams, defaultFilters]);

  const setFilter = useCallback(
    (key: string, value: string | undefined) => {
      const params = new URLSearchParams(searchParams.toString());
      if (value === undefined || value === "") {
        params.delete(key);
      } else {
        params.set(key, value);
      }
      router.push(`${pathname}?${params.toString()}`);
    },
    [searchParams, router, pathname],
  );

  const clearFilters = useCallback(() => {
    router.push(pathname);
  }, [router, pathname]);

  return { filters, setFilter, clearFilters };
}
