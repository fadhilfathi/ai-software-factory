"use client";

import { useCallback, useState } from "react";

/**
 * Hook for debounced search input.
 * Returns raw value (for immediate display) and debounced value (for API calls).
 */
export function useDebouncedSearch(debounceMs = 300) {
  const [raw, setRaw] = useState("");
  const [debounced, setDebounced] = useState("");
  const [timer, setTimer] = useState<ReturnType<typeof setTimeout> | null>(null);

  const onChange = useCallback(
    (value: string) => {
      setRaw(value);
      if (timer) clearTimeout(timer);
      const t = setTimeout(() => setDebounced(value), debounceMs);
      setTimer(t);
    },
    [debounceMs, timer],
  );

  return { raw, debounced, onChange };
}

/**
 * Hook for auto-saving to localStorage.
 */
export function useAutoSave<T>(key: string, initialValue: T, debounceMs = 1000) {
  const [value, setValue] = useState<T>(() => {
    if (typeof window !== "undefined") {
      try {
        const stored = localStorage.getItem(key);
        return stored ? (JSON.parse(stored) as T) : initialValue;
      } catch {
        return initialValue;
      }
    }
    return initialValue;
  });

  const [timer, setTimer] = useState<ReturnType<typeof setTimeout> | null>(null);

  const save = useCallback(
    (newValue: T) => {
      setValue(newValue);
      if (timer) clearTimeout(timer);
      const t = setTimeout(() => {
        localStorage.setItem(key, JSON.stringify(newValue));
      }, debounceMs);
      setTimer(t);
    },
    [debounceMs, timer],
  );

  return { value, save };
}
