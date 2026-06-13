"use client"

import { useSearchParams, useRouter, usePathname } from "next/navigation";
import { useCallback, useMemo } from "react";

type FilterState = Record<string, string | undefined>;

/**
 * Hook that syncs page filters with URL search params.
 *
 * URL is the canonical source — this hook both reads and writes it.
 *
 * Two consumer shapes:
 *
 *   1. Free-form filter list (`/projects/page.tsx`):
 *        const { filters, setFilter } = useProjectFilters({ status: "all" })
 *
 *   2. Project-scoped content (the ProjectPickerGate and all hooks in
 *      `lib/hooks.ts` that send `X-Project-ID`):
 *        const { projectId, setProjectId } = useProjectFilters()
 *
 *      `projectId` is sourced from the `?projectId=…` search param. The
 *      gate shows a project picker whenever this is empty/undefined;
 *      every hook that calls `useProjectFilters().projectId` will be
 *      disabled until the gate has set a value.
 */
export function useProjectFilters(defaultFilters: FilterState = {}) {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();

  const filters: FilterState = useMemo(() => {
    const result: FilterState = { ...defaultFilters };
    // In test environments `useSearchParams` can return null. We treat
    // null as "no params" rather than crashing.
    if (searchParams) {
      for (const [key, value] of searchParams.entries()) {
        result[key] = value;
      }
    }
    return result;
  }, [searchParams, defaultFilters]);

  const setFilter = useCallback(
    (key: string, value: string | undefined) => {
      const params = new URLSearchParams(searchParams?.toString() ?? "");
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

  // Convenience accessors for the project-scoped contract. The gate
  // and the data hooks in `lib/hooks.ts` read/write `projectId` via
  // these helpers instead of going through `filters`/`setFilter` directly
  // so the call sites stay readable.
  const projectId = filters.projectId;
  const setProjectId = useCallback(
    (value: string | undefined) => setFilter("projectId", value),
    [setFilter],
  );

  return { filters, setFilter, clearFilters, projectId, setProjectId };
}
