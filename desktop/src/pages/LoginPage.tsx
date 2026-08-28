import { useState, useEffect, useCallback } from "react";
import { motion } from "motion/react";
import { Gem, Loader2, AlertCircle } from "../components/icons";

type Tab = "login" | "register" | "reset";

interface Props {
  onLogin: (phone: string, password: string, remember: boolean) => Promise<void>;
  onRegister: (phone: string, code: string, password: string, adminToken?: string) => Promise<void>;
  onSendSmsCode: (phone: string, purpose: string) => Promise<void>;
  onResetPassword: (phone: string, code: string, newPassword: string) => Promise<void>;
}

export default function LoginPage({ onLogin, onRegister, onSendSmsCode, onResetPassword }: Props) {
  const [tab, setTab] = useState<Tab>("login");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [code, setCode] = useState("");
  const [remember, setRemember] = useState(true);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [resetDone, setResetDone] = useState(false);

  useEffect(() => {
    if (countdown <= 0) return;
    const timer = setTimeout(() => setCountdown(c => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [countdown]);

  const resetForm = useCallback(() => {
    setPhone("");
    setPassword("");
    setConfirmPassword("");
    setCode("");
    setError("");
    setResetDone(false);
  }, []);

  const switchTab = (t: Tab) => {
    resetForm();
    setTab(t);
  };

  const handleSendCode = async () => {
    if (!phone.trim() || countdown > 0) return;
    setError("");
    try {
      await onSendSmsCode(phone.trim(), tab === "register" ? "register" : "reset");
      setCountdown(60);
    } catch (err) {
      setError(String(err));
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await onLogin(phone, password, remember);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirmPassword) {
      setError("两次密码不一致");
      return;
    }
    setError("");
    setLoading(true);
    try {
      await onRegister(phone.trim(), code.trim(), password);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  };

  const handleReset = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirmPassword) {
      setError("两次密码不一致");
      return;
    }
    setError("");
    setLoading(true);
    try {
      await onResetPassword(phone.trim(), code.trim(), password);
      setResetDone(true);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  };

  const inputCls = "w-full px-3 py-2.5 bg-obsidian-800 border border-obsidian-700 rounded-lg text-obsidian-100 placeholder-obsidian-600 focus:outline-none focus:border-amber-500/50 focus:ring-1 focus:ring-amber-500/20 transition-all duration-200";

  return (
    <div className="min-h-screen flex items-center justify-center bg-obsidian-950 bg-noise">
      <div className="absolute inset-0" style={{ background: "radial-gradient(ellipse at center, rgba(245,158,11,0.03) 0%, transparent 70%)" }} />

      <motion.div
        className="relative z-10 w-full max-w-sm"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: "easeOut" }}
      >
        <div className="rounded-xl p-px bg-gradient-to-b from-amber-500/20 to-obsidian-700">
          <div className="rounded-xl bg-obsidian-900 p-8">
            {/* Logo */}
            <div className="flex flex-col items-center mb-6">
              <Gem size={24} className="text-amber-400" />
              <h1 className="text-xl font-bold text-obsidian-50 mt-3">LLM Gateway</h1>
              <p className="text-[10px] uppercase tracking-[0.2em] text-obsidian-500 mt-1">Developer Control Panel</p>
            </div>

            {/* Tabs */}
            <div className="flex gap-1 bg-obsidian-800 rounded-lg p-0.5 mb-6">
              {([["login", "登录"], ["register", "注册"], ["reset", "找回密码"]] as const).map(([key, label]) => (
                <button key={key} onClick={() => switchTab(key)} className={`flex-1 px-2 py-1.5 rounded-md text-xs font-medium transition-all ${tab === key ? "bg-amber-500 text-obsidian-950" : "text-obsidian-400 hover:text-obsidian-200"}`}>{label}</button>
              ))}
            </div>

            {/* Login form */}
            {tab === "login" && (
              <form onSubmit={handleLogin} className="space-y-4">
                <div>
                  <label className="block text-xs text-obsidian-400 font-medium mb-1.5">手机号</label>
                  <div className="flex gap-2">
                    <span className="flex items-center bg-obsidian-800 border border-obsidian-700 rounded-lg text-obsidian-400 text-sm px-3">+86</span>
                    <input className={`flex-1 ${inputCls.replace("w-full ", "")}`} placeholder="请输入手机号" value={phone} onChange={(e) => setPhone(e.target.value)} />
                  </div>
                </div>
                <div>
                  <label className="block text-xs text-obsidian-400 font-medium mb-1.5">密码</label>
                  <input type="password" className={inputCls} placeholder="请输入密码" value={password} onChange={(e) => setPassword(e.target.value)} />
                </div>
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={remember} onChange={(e) => setRemember(e.target.checked)} className="accent-amber-500 rounded" />
                  <span className="text-sm text-obsidian-400">记住我</span>
                </label>
                {error && <div className="flex items-center gap-1.5 text-red-400 text-sm"><AlertCircle size={14} /><span>{error}</span></div>}
                <button type="submit" disabled={loading || !phone || !password} className="w-full py-2.5 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 rounded-lg font-semibold transition-all duration-200">
                  {loading ? <span className="inline-flex items-center gap-2"><Loader2 size={16} className="animate-spin" />登录中</span> : "登录"}
                </button>
              </form>
            )}

            {/* Register form */}
            {tab === "register" && (
              <form onSubmit={handleRegister} className="space-y-4">
                <div>
                  <label className="block text-xs text-obsidian-400 font-medium mb-1.5">手机号</label>
                  <div className="flex gap-2">
                    <span className="flex items-center bg-obsidian-800 border border-obsidian-700 rounded-lg text-obsidian-400 text-sm px-3">+86</span>
                    <input className={`flex-1 ${inputCls.replace("w-full ", "")}`} placeholder="请输入手机号" value={phone} onChange={(e) => setPhone(e.target.value)} />
                  </div>
                </div>
                <div>
                  <label className="block text-xs text-obsidian-400 font-medium mb-1.5">验证码</label>
                  <div className="flex gap-2">
                    <input className={`flex-1 ${inputCls.replace("w-full ", "")}`} placeholder="短信验证码" value={code} onChange={(e) => setCode(e.target.value)} />
                    <button type="button" onClick={handleSendCode} disabled={countdown > 0 || !phone.trim()} className="px-4 py-2.5 bg-obsidian-800 border border-obsidian-700 rounded-lg text-xs text-obsidian-300 hover:text-obsidian-100 disabled:opacity-40 whitespace-nowrap transition-colors">
                      {countdown > 0 ? `${countdown}s` : "发送验证码"}
                    </button>
                  </div>
                </div>
                <div>
                  <label className="block text-xs text-obsidian-400 font-medium mb-1.5">密码</label>
                  <input type="password" className={inputCls} placeholder="设置密码" value={password} onChange={(e) => setPassword(e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs text-obsidian-400 font-medium mb-1.5">确认密码</label>
                  <input type="password" className={inputCls} placeholder="再次输入密码" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} />
                </div>
                {error && <div className="flex items-center gap-1.5 text-red-400 text-sm"><AlertCircle size={14} /><span>{error}</span></div>}
                <button type="submit" disabled={loading || !phone || !code || !password || !confirmPassword} className="w-full py-2.5 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 rounded-lg font-semibold transition-all duration-200">
                  {loading ? <span className="inline-flex items-center gap-2"><Loader2 size={16} className="animate-spin" />注册中</span> : "注册"}
                </button>
              </form>
            )}

            {/* Reset password form */}
            {tab === "reset" && (
              resetDone ? (
                <div className="text-center py-6">
                  <div className="text-emerald-400 text-sm font-medium mb-2">密码重置成功</div>
                  <p className="text-xs text-obsidian-400 mb-4">请使用新密码登录</p>
                  <button onClick={() => switchTab("login")} className="px-6 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-all">返回登录</button>
                </div>
              ) : (
                <form onSubmit={handleReset} className="space-y-4">
                  <div>
                    <label className="block text-xs text-obsidian-400 font-medium mb-1.5">手机号</label>
                    <div className="flex gap-2">
                      <span className="flex items-center bg-obsidian-800 border border-obsidian-700 rounded-lg text-obsidian-400 text-sm px-3">+86</span>
                      <input className={`flex-1 ${inputCls.replace("w-full ", "")}`} placeholder="请输入手机号" value={phone} onChange={(e) => setPhone(e.target.value)} />
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs text-obsidian-400 font-medium mb-1.5">验证码</label>
                    <div className="flex gap-2">
                      <input className={`flex-1 ${inputCls.replace("w-full ", "")}`} placeholder="短信验证码" value={code} onChange={(e) => setCode(e.target.value)} />
                      <button type="button" onClick={handleSendCode} disabled={countdown > 0 || !phone.trim()} className="px-4 py-2.5 bg-obsidian-800 border border-obsidian-700 rounded-lg text-xs text-obsidian-300 hover:text-obsidian-100 disabled:opacity-40 whitespace-nowrap transition-colors">
                        {countdown > 0 ? `${countdown}s` : "发送验证码"}
                      </button>
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs text-obsidian-400 font-medium mb-1.5">新密码</label>
                    <input type="password" className={inputCls} placeholder="设置新密码" value={password} onChange={(e) => setPassword(e.target.value)} />
                  </div>
                  <div>
                    <label className="block text-xs text-obsidian-400 font-medium mb-1.5">确认密码</label>
                    <input type="password" className={inputCls} placeholder="再次输入新密码" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} />
                  </div>
                  {error && <div className="flex items-center gap-1.5 text-red-400 text-sm"><AlertCircle size={14} /><span>{error}</span></div>}
                  <button type="submit" disabled={loading || !phone || !code || !password || !confirmPassword} className="w-full py-2.5 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 rounded-lg font-semibold transition-all duration-200">
                    {loading ? <span className="inline-flex items-center gap-2"><Loader2 size={16} className="animate-spin" />重置中</span> : "重置密码"}
                  </button>
                </form>
              )
            )}
          </div>
        </div>
      </motion.div>
    </div>
  );
}
