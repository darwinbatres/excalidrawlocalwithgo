import { useState, useCallback, useEffect } from "react";
import { ApiError } from "@/services/api.client";

/**
 * useAsyncAction — Wraps an async function with loading, error, and reset state.
 * Eliminates the repetitive try/catch/setLoading/setError pattern.
 */
export function useAsyncAction<TArgs extends unknown[], TResult>(
  fn: (...args: TArgs) => Promise<TResult>
) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(
    async (...args: TArgs): Promise<TResult | undefined> => {
      setLoading(true);
      setError(null);
      try {
        const result = await fn(...args);
        return result;
      } catch (err) {
        const message =
          err instanceof ApiError
            ? err.message
            : err instanceof Error
              ? err.message
              : "An unexpected error occurred";
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [fn]
  );

  const clearError = useCallback(() => setError(null), []);

  return { run, loading, error, clearError };
}

/**
 * useModal — Manages open/close state and an optional associated item.
 * Replaces repeated [showModal, setShowModal] + [item, setItem] pairs.
 */
export function useModal<T = undefined>() {
  const [isOpen, setIsOpen] = useState(false);
  const [data, setData] = useState<T | null>(null);

  const open = useCallback((item?: T) => {
    setData(item ?? null);
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
    setData(null);
  }, []);

  return { isOpen, data, open, close };
}

/**
 * useOutsideClick — Calls handler when a click occurs outside the ref element.
 * Replaces manual addEventListener/removeEventListener patterns.
 */
export function useOutsideClick<T extends HTMLElement>(
  ref: React.RefObject<T | null>,
  handler: () => void,
  active = true
) {
  useEffect(() => {
    if (!active) return;

    function handleMouseDown(event: MouseEvent) {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        handler();
      }
    }

    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        handler();
      }
    }

    document.addEventListener("mousedown", handleMouseDown);
    document.addEventListener("keydown", handleEscape);
    return () => {
      document.removeEventListener("mousedown", handleMouseDown);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [ref, handler, active]);
}

/**
 * formatApiError — Extracts a user-friendly message from any caught error.
 */
export function formatApiError(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return err.message;
  if (err instanceof Error) return err.message;
  return fallback;
}
