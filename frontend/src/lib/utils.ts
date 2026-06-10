import { type ClassValue, clsx } from "clsx";

/**
 * Merge Tailwind classes with precedence (rightmost wins).
 */
export function cn(...inputs: ClassValue[]) {
  return clsx(inputs);
}

/**
 * Format a relative time string (e.g. "2h ago").
 */
export function timeAgo(date: Date | string | number): string {
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
