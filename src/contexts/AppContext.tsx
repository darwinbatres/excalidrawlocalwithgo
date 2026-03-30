/**
 * App Context — Global application state
 *
 * Delegates auth to AuthContext. Provides org state and CRUD
 * using the Go backend via orgApi.
 */

import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
} from "react";
import { useAuth } from "@/contexts/AuthContext";
import { orgApi } from "@/services/api.client";
import type { Organization } from "@/types";

interface OrgWithRole extends Organization {
  role?: string;
  memberCount?: number;
  boardCount?: number;
}

interface AppContextValue {
  // Auth (delegated from AuthContext)
  user: ReturnType<typeof useAuth>["user"];
  isLoading: boolean;
  isAuthenticated: boolean;

  // Organization state
  currentOrg: OrgWithRole | null;
  userOrgs: OrgWithRole[];

  // Actions
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  switchOrg: (orgId: string) => void;
  createOrg: (name: string, slug: string) => Promise<OrgWithRole>;
  renameOrg: (orgId: string, name: string) => Promise<void>;
  deleteOrg: (orgId: string) => Promise<void>;
  refreshOrgs: () => Promise<void>;
}

const AppContext = createContext<AppContextValue | null>(null);

const CURRENT_ORG_KEY = "excalidraw_current_org_id";

export function AppProvider({ children }: { children: React.ReactNode }) {
  const auth = useAuth();
  const [currentOrg, setCurrentOrg] = useState<OrgWithRole | null>(null);
  const [userOrgs, setUserOrgs] = useState<OrgWithRole[]>([]);
  const [orgsLoading, setOrgsLoading] = useState(false);

  const isLoading = auth.isLoading || orgsLoading;

  // Persist current org to localStorage
  useEffect(() => {
    if (currentOrg?.id) {
      localStorage.setItem(CURRENT_ORG_KEY, currentOrg.id);
    }
  }, [currentOrg?.id]);

  // Fetch user's organizations from Go backend
  const fetchOrgs = useCallback(async () => {
    if (!auth.isAuthenticated) {
      setUserOrgs([]);
      setCurrentOrg(null);
      return;
    }

    setOrgsLoading(true);
    try {
      const result = await orgApi.list();
      const orgs = (result.items || []) as OrgWithRole[];
      setUserOrgs(orgs);

      if (orgs.length > 0) {
        setCurrentOrg((prev) => {
          if (prev) return prev;

          const savedOrgId = localStorage.getItem(CURRENT_ORG_KEY);
          const savedOrg = savedOrgId
            ? orgs.find((o) => o.id === savedOrgId)
            : null;
          return savedOrg || orgs[0];
        });
      }
    } catch (error) {
      console.error("[AppContext] Error fetching organizations:", error);
    } finally {
      setOrgsLoading(false);
    }
  }, [auth.isAuthenticated]);

  // Load orgs when auth state changes
  useEffect(() => {
    if (!auth.isLoading) {
      fetchOrgs();
    }
  }, [auth.isLoading, auth.isAuthenticated, fetchOrgs]);

  const login = useCallback(
    async (email: string, password: string) => {
      await auth.login(email, password);
    },
    [auth]
  );

  const logout = useCallback(async () => {
    localStorage.removeItem(CURRENT_ORG_KEY);
    setCurrentOrg(null);
    setUserOrgs([]);
    await auth.logout();
  }, [auth]);

  const switchOrg = useCallback(
    (orgId: string) => {
      const org = userOrgs.find((o) => o.id === orgId);
      if (org) {
        setCurrentOrg(org);
      }
    },
    [userOrgs]
  );

  const createOrg = useCallback(
    async (name: string, slug: string): Promise<OrgWithRole> => {
      if (!auth.isAuthenticated) {
        throw new Error("Must be logged in to create an organization");
      }

      const newOrg = await orgApi.create(name, slug);
      await fetchOrgs();
      setCurrentOrg(newOrg as OrgWithRole);
      return newOrg as OrgWithRole;
    },
    [auth.isAuthenticated, fetchOrgs]
  );

  const renameOrg = useCallback(
    async (orgId: string, name: string): Promise<void> => {
      if (!auth.isAuthenticated) {
        throw new Error("Must be logged in to rename an organization");
      }

      await orgApi.update(orgId, { name });
      await fetchOrgs();

      if (currentOrg?.id === orgId) {
        setCurrentOrg((prev) => (prev ? { ...prev, name } : null));
      }
    },
    [auth.isAuthenticated, fetchOrgs, currentOrg?.id]
  );

  const deleteOrg = useCallback(
    async (orgId: string): Promise<void> => {
      if (!auth.isAuthenticated) {
        throw new Error("Must be logged in to delete an organization");
      }

      const wasCurrentOrg = currentOrg?.id === orgId;
      await orgApi.delete(orgId);

      if (wasCurrentOrg) {
        setCurrentOrg(null);
      }

      await fetchOrgs();
    },
    [auth.isAuthenticated, fetchOrgs, currentOrg?.id]
  );

  const value: AppContextValue = {
    user: auth.user,
    isLoading,
    isAuthenticated: auth.isAuthenticated,
    currentOrg,
    userOrgs,
    login,
    logout,
    switchOrg,
    createOrg,
    renameOrg,
    deleteOrg,
    refreshOrgs: fetchOrgs,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp(): AppContextValue {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error("useApp must be used within an AppProvider");
  }
  return context;
}

export function useUser() {
  const { user, isLoading, isAuthenticated } = useApp();
  return { user, isLoading, isAuthenticated };
}

export function useOrg() {
  const { currentOrg, userOrgs, switchOrg, createOrg, renameOrg, deleteOrg } =
    useApp();
  return { currentOrg, userOrgs, switchOrg, createOrg, renameOrg, deleteOrg };
}
