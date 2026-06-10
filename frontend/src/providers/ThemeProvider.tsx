"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from "react";

type ThemeMode = "light" | "dark";
type FontScale = 1 | 1.25 | 1.5;

type ThemeContextValue = {
  mode: ThemeMode;
  toggleMode: () => void;
  prefersReducedMotion: boolean;
  fontScale: FontScale;
  setFontScale: (scale: FontScale) => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

const STORAGE_KEY_MODE = "theme-mode";
const STORAGE_KEY_FONT = "theme-font-scale";

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(() => {
    if (typeof window !== "undefined") {
      return (localStorage.getItem(STORAGE_KEY_MODE) as ThemeMode) || "dark";
    }
    return "dark";
  });

  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false);
  const [fontScale, setFontScale] = useState<FontScale>(() => {
    if (typeof window !== "undefined") {
      const stored = Number(localStorage.getItem(STORAGE_KEY_FONT));
      return ([1, 1.25, 1.5].includes(stored) ? stored : 1) as FontScale;
    }
    return 1;
  });

  // Sync mode to <html> data-theme attribute
  useEffect(() => {
    document.documentElement.dataset.theme = mode;
    localStorage.setItem(STORAGE_KEY_MODE, mode);
  }, [mode]);

  // Apply font scale
  useEffect(() => {
    document.documentElement.style.fontSize = `${fontScale * 100}%`;
    localStorage.setItem(STORAGE_KEY_FONT, String(fontScale));
  }, [fontScale]);

  // Detect prefers-reduced-motion
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const handler = (e: MediaQueryListEvent) => setPrefersReducedMotion(e.matches);
    setPrefersReducedMotion(mq.matches);
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, []);

  const toggleMode = useCallback(() => {
    setMode((prev) => (prev === "dark" ? "light" : "dark"));
  }, []);

  return (
    <ThemeContext.Provider
      value={{ mode, toggleMode, prefersReducedMotion, fontScale, setFontScale }}
    >
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within <ThemeProvider>");
  return ctx;
}
