import { useState, useEffect, useCallback } from "react";
import type { UserInfo } from "../lib/types";
import { api } from "../lib/tauri";

export function useAuth() {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.checkToken()
      .then(u => setUser(u ?? null))
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  const login = useCallback(async (phone: string, password: string, remember: boolean) => {
    const u = await api.login(phone, password, remember);
    setUser(u);
    return u;
  }, []);

  const register = useCallback(async (phone: string, code: string, password: string, adminToken?: string) => {
    const u = await api.register(phone, code, password, adminToken);
    setUser(u);
    return u;
  }, []);

  const sendSmsCode = useCallback(async (phone: string, purpose: string) => {
    await api.sendSmsCode(phone, purpose);
  }, []);

  const resetPassword = useCallback(async (phone: string, code: string, newPassword: string) => {
    await api.resetPassword(phone, code, newPassword);
  }, []);

  const logout = useCallback(async () => {
    await api.logout();
    setUser(null);
  }, []);

  return { user, loading, login, register, sendSmsCode, resetPassword, logout };
}
