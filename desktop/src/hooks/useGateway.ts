import { useState, useEffect, useCallback } from "react";
import { api } from "../lib/tauri";

export function useGateway() {
  const [balance, setBalance] = useState<number | null>(null);
  const [giftBalance, setGiftBalance] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [authExpired, setAuthExpired] = useState(false);
  const [offline, setOffline] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getBalance();
      setBalance(data.balance);
      setGiftBalance(data.gift_balance);
      setOffline(false);
    } catch (err) {
      const msg = String(err);
      if (msg.includes("登录已过期") || msg.includes("Unauthorized")) {
        setAuthExpired(true);
      } else if (msg.includes("网络错误") || msg.includes("Network")) {
        setOffline(true);
      }
      console.error("获取余额失败:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  return { balance, giftBalance, loading, refresh, authExpired, offline };
}
