import { createContext, useContext, useCallback, useState, useEffect, useMemo } from "react";
import type { ReactNode } from "react";
import type { MeResponse, ImageShareMe } from "@/types/api";
import { apiGet, apiPost, ApiError } from "@/lib/api-client";
import { isImageHost } from "@/lib/host";

interface AuthState {
  user: MeResponse | null;
  imageShare: ImageShareMe | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isImageShare: boolean;
  login: (identifier: string, password: string, remember?: boolean) => Promise<void>;
  loginImageShare: (key: string) => Promise<void>;
  refreshImageShare: () => Promise<void>;
  register: (identifier: string, code: string, password: string, adminToken?: string, acceptAup?: boolean, referralCode?: string) => Promise<void>;
  logout: () => Promise<void>;
  sendSmsCode: (phone: string, purpose: string) => Promise<void>;
  sendVerificationCode: (identifier: string, purpose: string) => Promise<void>;
  resetPassword: (identifier: string, code: string, newPassword: string) => Promise<void>;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<MeResponse | null>(null);
  const [imageShare, setImageShare] = useState<ImageShareMe | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Bootstrap: try image-share session first (cheaper, isolated cookie),
  // then fall back to the regular /api/me. Only one of them will succeed
  // because the cookies are independent and JWTs are typed by role.
  // On the image-share subdomain we skip the /api/me probe — the main-platform
  // cookie is host-only and can't be present here.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const share = await apiGet<ImageShareMe>("/api/image-share/me");
        if (!cancelled) {
          setImageShare(share);
          setUser(null);
          return;
        }
      } catch {
        // not an image-share session — fall through
      }
      if (isImageHost()) {
        if (!cancelled) {
          setUser(null);
          setImageShare(null);
        }
        return;
      }
      try {
        const me = await apiGet<MeResponse>("/api/me");
        if (!cancelled) {
          setUser(me);
          setImageShare(null);
        }
      } catch {
        if (!cancelled) {
          setUser(null);
          setImageShare(null);
        }
      }
    })().finally(() => {
      if (!cancelled) setIsLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (identifier: string, password: string, remember?: boolean) => {
    await apiPost("/api/login", { identifier, password, remember });
    const me = await apiGet<MeResponse>("/api/me");
    setUser(me);
    setImageShare(null);
  }, []);

  const loginImageShare = useCallback(async (key: string) => {
    await apiPost("/api/image-share/login", { key });
    const share = await apiGet<ImageShareMe>("/api/image-share/me");
    setImageShare(share);
    setUser(null);
  }, []);

  const refreshImageShare = useCallback(async () => {
    try {
      const share = await apiGet<ImageShareMe>("/api/image-share/me");
      setImageShare(share);
    } catch {
      // session expired — leave state to be cleaned by caller
    }
  }, []);

  const register = useCallback(
    async (identifier: string, code: string, password: string, adminToken?: string, acceptAup?: boolean, referralCode?: string) => {
      const isEmail = identifier.includes("@");
      const body: Record<string, unknown> = { code, password, accept_aup: acceptAup ?? true };
      if (isEmail) {
        body.email = identifier;
      } else {
        body.phone = identifier;
      }
      if (adminToken) body.admin_token = adminToken;
      if (referralCode) body.referral_code = referralCode;
      await apiPost<{
        token: string;
        user: { id: string; phone?: string; email?: string; role: string };
      }>("/api/register", body);
      const me = await apiGet<MeResponse>("/api/me");
      setUser(me);
      setImageShare(null);
    },
    [],
  );

  const logout = useCallback(async () => {
    if (imageShare) {
      try {
        await apiPost("/api/image-share/logout");
      } catch (err) {
        if (!(err instanceof ApiError && err.status === 401)) throw err;
      }
      setImageShare(null);
      return;
    }
    try {
      await apiPost("/api/logout");
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 401)) throw err;
    }
    setUser(null);
  }, [imageShare]);

  const sendSmsCode = useCallback(
    async (phone: string, purpose: string) => {
      await apiPost("/api/sms/send", { phone, purpose });
    },
    [],
  );

  const sendVerificationCode = useCallback(
    async (identifier: string, purpose: string) => {
      const isEmail = identifier.includes("@");
      if (isEmail) {
        await apiPost("/api/sms/send", { email: identifier, purpose });
      } else {
        await apiPost("/api/sms/send", { phone: identifier, purpose });
      }
    },
    [],
  );

  const resetPassword = useCallback(
    async (identifier: string, code: string, newPassword: string) => {
      const isEmail = identifier.includes("@");
      if (isEmail) {
        await apiPost("/api/reset-password", { email: identifier, code, new_password: newPassword });
      } else {
        await apiPost("/api/reset-password", { phone: identifier, code, new_password: newPassword });
      }
    },
    [],
  );

  const value: AuthState = useMemo(
    () => ({
      user,
      imageShare,
      isLoading,
      isAuthenticated: user !== null || imageShare !== null,
      isAdmin: user?.role === "admin",
      isImageShare: imageShare !== null,
      login,
      loginImageShare,
      refreshImageShare,
      register,
      logout,
      sendSmsCode,
      sendVerificationCode,
      resetPassword,
    }),
    [user, imageShare, isLoading, login, loginImageShare, refreshImageShare, register, logout, sendSmsCode, sendVerificationCode, resetPassword],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (ctx === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
