/**
 * Auth Context — Cookie-based JWT auth against Go backend
 *
 * Replaces NextAuth SessionProvider. The Go backend sets httpOnly cookies
 * on login/register, so we only need to call /auth/me to check session.
 */

import React, {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  useRef,
} from "react";
import { authApi, ApiError } from "@/services/api.client";
import type { User } from "@/types";

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/** How often to silently refresh the session (4 minutes) */
const REFRESH_INTERVAL_MS = 4 * 60 * 1000;

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const isAuthenticated = !!user;

  const clearRefreshTimer = useCallback(() => {
    if (refreshTimerRef.current) {
      clearInterval(refreshTimerRef.current);
      refreshTimerRef.current = null;
    }
  }, []);

  const startRefreshTimer = useCallback(() => {
    clearRefreshTimer();
    refreshTimerRef.current = setInterval(async () => {
      try {
        await authApi.refresh();
      } catch {
        // If refresh fails, user will be logged out on next API call
      }
    }, REFRESH_INTERVAL_MS);
  }, [clearRefreshTimer]);

  // Check current session on mount
  useEffect(() => {
    let cancelled = false;

    async function checkSession() {
      try {
        const me = await authApi.me();
        if (!cancelled) {
          setUser(me);
          startRefreshTimer();
        }
      } catch {
        if (!cancelled) {
          setUser(null);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    checkSession();

    return () => {
      cancelled = true;
      clearRefreshTimer();
    };
  }, [startRefreshTimer, clearRefreshTimer]);

  const login = useCallback(
    async (email: string, password: string) => {
      const response = await authApi.login(email, password);
      setUser(response.user);
      startRefreshTimer();
    },
    [startRefreshTimer]
  );

  const register = useCallback(
    async (email: string, password: string, name: string) => {
      const response = await authApi.register(email, password, name);
      setUser(response.user);
      startRefreshTimer();
    },
    [startRefreshTimer]
  );

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // Ignore errors during logout
    }
    clearRefreshTimer();
    setUser(null);
  }, [clearRefreshTimer]);

  const refreshSession = useCallback(async () => {
    try {
      const me = await authApi.me();
      setUser(me);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setUser(null);
        clearRefreshTimer();
      }
    }
  }, [clearRefreshTimer]);

  const value: AuthContextValue = {
    user,
    isLoading,
    isAuthenticated,
    login,
    register,
    logout,
    refreshSession,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
