import { createContext, useContext, useCallback, useState, useEffect, useMemo } from "react";
import type { ReactNode } from "react";
import { apiGet, apiPost, ApiError } from "@/lib/api-client";

interface SubUserInfo {
  sub_user_id: string;
  tenant_id: string;
  username: string;
  nickname: string;
  status: string;
  quota_limit: number | null;
  quota_used: number;
  quota_remaining?: number;
  role: "sub_user";
}

interface SubUserAuthState {
  user: SubUserInfo | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (tenantId: string, username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
}

const SubUserAuthContext = createContext<SubUserAuthState | undefined>(undefined);

export function SubUserAuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SubUserInfo | null>(null);
  // tri-state: undefined = first fetch in flight, true = succeeded, false = failed.
  const [hasFetched, setHasFetched] = useState(false);

  const fetchMe = useCallback(async () => {
    try {
      const data = await apiGet<SubUserInfo>("/api/sub-user/me");
      if (data.role === "sub_user") {
        setUser(data);
      } else {
        setUser(null);
      }
    } catch {
      setUser(null);
    } finally {
      setHasFetched(true);
    }
  }, []);

  useEffect(() => {
    fetchMe();
  }, [fetchMe]);

  const isLoading = !hasFetched;

  const login = useCallback(async (tenantId: string, username: string, password: string) => {
    await apiPost("/api/sub-user/login", { tenant_id: tenantId, username, password });
    const data = await apiGet<SubUserInfo>("/api/sub-user/me");
    setUser(data);
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiPost("/api/sub-user/logout");
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 401)) throw err;
    }
    setUser(null);
  }, []);

  const refresh = useCallback(async () => {
    await fetchMe();
  }, [fetchMe]);

  const value: SubUserAuthState = useMemo(
    () => ({
      user,
      isLoading,
      isAuthenticated: user !== null,
      login,
      logout,
      refresh,
    }),
    [user, isLoading, login, logout, refresh],
  );

  return <SubUserAuthContext.Provider value={value}>{children}</SubUserAuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useSubUserAuth(): SubUserAuthState {
  const ctx = useContext(SubUserAuthContext);
  if (ctx === undefined) {
    throw new Error("useSubUserAuth must be used within a SubUserAuthProvider");
  }
  return ctx;
}
