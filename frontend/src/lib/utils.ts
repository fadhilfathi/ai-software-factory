import { type ClassValue, clsx } from "clsx";

/**
 * Merge Tailwind classes with precedence (rightmost wins).
 */
export function cn(...inputs: ClassValue[]) {
  return clsx(inputs);
}

/**
 * Format a relative time string (e.g. "2h ago"). Returns "—" for
 * null/undefined so call sites can pass optional timestamps without
 * a null check.
 */
export function timeAgo(date: Date | string | number | null | undefined): string {
  if (!date) return "—";
  const now = new Date();
  const then = typeof date === "string" || typeof date === "number" ? new Date(date) : date;
  const seconds = Math.floor((now.getTime() - then.getTime()) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

/**
 * Format a number with commas.
 */
export function formatNumber(n: number): string {
  return n.toLocaleString("en-US");
}

/**
 * Format currency.
 */
export function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(amount);
}

/**
 * Truncate a string to N chars with ellipsis.
 */
export function truncate(str: string, max: number): string {
  if (str.length <= max) return str;
  return str.slice(0, max - 1) + "…";
}

/**
 * Generate a random optimistic ID.
 */
export function optimisticId(): string {
  return `opt_${crypto.randomUUID().slice(0, 8)}`;
}

/**
 * Format a fraction (0-1) as a percent string, e.g. 0.73 → "73%".
 */
export function formatPercent(value: number, fractionDigits = 0): string {
  if (!Number.isFinite(value)) return "—";
  return `${(value * 100).toFixed(fractionDigits)}%`;
}

/**
 * Format an uptime duration (seconds) as a compact human label,
 * e.g. 90061 → "1d 1h", 3661 → "1h 1m", 60 → "1m".
 */
export function formatUptime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  const s = Math.floor(seconds);
  const days = Math.floor(s / 86400);
  const hours = Math.floor((s % 86400) / 3600);
  const minutes = Math.floor((s % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m`;
  return `${s}s`;
}
