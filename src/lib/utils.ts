import { v4 as uuidv4 } from "uuid";
import type { PersistedAppState } from "@/types";

/**
 * Generate a new unique ID (CUID-like)
 */
export function generateId(): string {
  return uuidv4();
}

/**
 * Generate a new ETag for optimistic concurrency
 */
export function generateEtag(): string {
  return uuidv4();
}

/**
 * Get current ISO timestamp
 */
export function nowISO(): string {
  return new Date().toISOString();
}

/**
 * Strip volatile fields from Excalidraw appState
 * We only want to persist non-volatile settings
 */
export function stripVolatileAppState(appState: Record<string, unknown>): PersistedAppState {
  return {
    viewBackgroundColor: appState.viewBackgroundColor as string | undefined,
    gridSize: appState.gridSize as number | null | undefined,
    gridModeEnabled: appState.gridModeEnabled as boolean | undefined,
    theme: appState.theme as "light" | "dark" | undefined,
    zenModeEnabled: appState.zenModeEnabled as boolean | undefined,
    viewModeEnabled: appState.viewModeEnabled as boolean | undefined,
  };
}

/**
 * Create a URL-friendly slug from a string
 */
export function slugify(str: string): string {
  return str
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, "")
    .replace(/[\s_-]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

/**
 * Debounce function with proper typing
 */
export function debounce<TArgs extends unknown[]>(
  fn: (...args: TArgs) => void,
  delay: number
): (...args: TArgs) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null;
  
  return (...args: TArgs) => {
    if (timeoutId) {
      clearTimeout(timeoutId);
    }
    timeoutId = setTimeout(() => {
      fn(...args);
    }, delay);
  };
}

/**
 * Format relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSecs < 60) return "just now";
  if (diffMins < 60) return `${diffMins} minute${diffMins > 1 ? "s" : ""} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? "s" : ""} ago`;
  if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? "s" : ""} ago`;
  
  return date.toLocaleDateString();
}

/**
 * Truncate text with ellipsis
 */
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - 3) + "...";
}

/**
 * Class name helper (like clsx but simple)
 */
export function cn(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(" ");
}
