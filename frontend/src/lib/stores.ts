/**
 * Persistent client-side stores using localStorage.
 *
 * These stores manage non-server state that should survive page reloads:
 *   - Board view preferences (collapsed columns, grouping)
 *   - Dismissed hints / onboarding completions
 *   - Recently viewed items
 *   - Global search history
 *
 * Uses a simple subscribe/notify pattern with localStorage persistence.
 */

// ─── Generic store factory ───────────────────────────────────────────────────

type Listener = () => void;

interface Store<T> {
  getState(): T;
  setState(updater: Partial<T> | ((prev: T) => Partial<T>)): void;
  subscribe(listener: Listener): () => void;
  reset(): void;
}

function createStore<T>(key: string, defaults: T): Store<T> {
  const listeners = new Set<Listener>();

  function read(): T {
    if (typeof window === "undefined") return defaults;
    try {
      const raw = localStorage.getItem(key);
      return raw ? (JSON.parse(raw) as T) : defaults;
    } catch {
      return defaults;
    }
  }

  function write(state: T) {
    try {
      localStorage.setItem(key, JSON.stringify(state));
    } catch {
      // localStorage full or unavailable — silently skip
    }
  }

  let state = read();

  return {
    getState: () => state,
    setState: (updater) => {
      const patch = typeof updater === "function" ? updater(state) : updater;
      state = { ...state, ...patch };
      write(state);
      listeners.forEach((fn) => fn());
    },
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    reset: () => {
      state = { ...defaults };
      write(state);
      listeners.forEach((fn) => fn());
    },
  };
}

// ─── Store definitions ───────────────────────────────────────────────────────

export interface BoardPreferences {
  collapsedColumns: string[];
  groupBy: "status" | "priority" | "agent" | "none";
  compactMode: boolean;
}

export interface AppPreferences {
  sidebarCollapsed: boolean;
  dismissedHints: string[];
  recentProjectIds: string[];
  searchHistory: string[];
  tutorialCompleted: boolean;
}

const BOARD_DEFAULTS: BoardPreferences = {
  collapsedColumns: [],
  groupBy: "status",
  compactMode: false,
};

const APP_DEFAULTS: AppPreferences = {
  sidebarCollapsed: false,
  dismissedHints: [],
  recentProjectIds: [],
  searchHistory: [],
  tutorialCompleted: false,
};

export const boardStore = createStore("board-preferences", BOARD_DEFAULTS);
export const appStore = createStore("app-preferences", APP_DEFAULTS);

// ─── React Hook ──────────────────────────────────────────────────────────────

import { useSyncExternalStore, useCallback } from "react";

/**
 * Subscribe to a persistent store from React components.
 *
 * Usage:
 * ```ts
 * const prefs = useStore(boardStore);
 * // prefs.collapsedColumns, prefs.groupBy, etc.
 * ```
 */
export function useStore<T>(store: Store<T>): T {
  const getSnapshot = useCallback(() => store.getState(), [store]);
  return useSyncExternalStore(store.subscribe, getSnapshot, getSnapshot);
}

/**
 * Get a setter for a specific store.
 *
 * Usage:
 * ```ts
 * const update = useStoreSet(boardStore);
 * update({ collapsedColumns: ["done"] });
 * // or with function:
 * update((prev) => ({ collapsedColumns: [...prev.collapsedColumns, "review"] }));
 * ```
 */
export function useStoreSet<T>(store: Store<T>) {
  return store.setState.bind(store);
}

/**
 * Hook that returns a single key from a store + a setter for that key.
 *
 * Usage:
 * ```ts
 * const [groupBy, setGroupBy] = useStoreKey(boardStore, "groupBy");
 * ```
 */
export function useStoreKey<T, K extends keyof T>(
  store: Store<T>,
  key: K,
): [T[K], (value: T[K]) => void] {
  const state = useStore(store);
  const setter = useCallback(
    (value: T[K]) => store.setState({ [key]: value } as unknown as Partial<T>),
    [store, key],
  );
  return [state[key], setter];
}
