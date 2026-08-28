import { useState, useEffect } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Zap, Eye, EyeOff, Phone, Lock, Mail, CheckCircle, ArrowLeft, Shield } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { apiGet } from "@/lib/api-client";
import BeianBar from "@/components/BeianBar";
import { Seo } from "@/components/Seo";

const AuthPage = () => {
  const navigate = useNavigate();
  const auth = useAuth();

  const [tab, setTab] = useState<"login" | "register" | "forgot">("login");
  const [showPwd, setShowPwd] = useState(false);
  const [remember, setRemember] = useState(() => localStorage.getItem("auth_remember") === "true");
  const [identifier, setIdentifier] = useState(() => localStorage.getItem("auth_identifier") ?? "");
  const [password, setPassword] = useState(() => {
    const saved = localStorage.getItem("auth_password");
    return saved ? atob(saved) : "";
  });
  const [confirmPassword, setConfirmPassword] = useState("");
  const [verifyCode, setVerifyCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [resetSuccess, setResetSuccess] = useState(false);
  const [needsAdminSetup, setNeedsAdminSetup] = useState(false);
  const [adminToken, setAdminToken] = useState("");
  const [acceptAup, setAcceptAup] = useState(false);
  const [referralCode, setReferralCode] = useState(() => new URLSearchParams(window.location.search).get("ref") ?? "");

  // 判断是否为邮箱
  const isEmail = (str: string) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(str);

  useEffect(() => {
    apiGet<{ needs_admin_setup: boolean }>("/api/system/setup-status")
      .then((data) => setNeedsAdminSetup(data.needs_admin_setup))
      .catch(() => {});
  }, []);

  if (auth.isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  if (auth.user) {
    return <Navigate to="/dashboard" replace />;
  }

  const handleSendCode = async (purpose: string = "register") => {
    if (countdown > 0 || loading) return;
    setError("");
    try {
      await auth.sendVerificationCode(identifier, purpose);
      setCodeSent(true);
      setCountdown(60);
      const timer = setInterval(() => {
        setCountdown((v) => {
          if (v <= 1) { clearInterval(timer); return 0; }
          return v - 1;
        });
      }, 1000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "发送验证码失败");
    }
  };

  const handleLogin = async () => {
    if (loading) return;
    setError("");
    setLoading(true);
    try {
      await auth.login(identifier, password, remember);
      if (remember) {
        localStorage.setItem("auth_remember", "true");
        localStorage.setItem("auth_identifier", identifier);
        localStorage.setItem("auth_password", btoa(password));
      } else {
        localStorage.removeItem("auth_remember");
        localStorage.removeItem("auth_identifier");
        localStorage.removeItem("auth_password");
      }
      navigate("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async () => {
    if (loading) return;
    setError("");
    if (!acceptAup) {
      setError("请阅读并同意使用条款");
      return;
    }
    setLoading(true);
    try {
      await auth.register(identifier, verifyCode, password, needsAdminSetup ? adminToken : undefined, acceptAup, referralCode || undefined);
      navigate(needsAdminSetup ? "/dashboard" : "/onboarding");
    } catch (err) {
      setError(err instanceof Error ? err.message : "注册失败");
    } finally {
      setLoading(false);
    }
  };

  const handleResetPassword = async () => {
    if (loading) return;
    setError("");
    if (password !== confirmPassword) {
      setError("两次输入的密码不一致");
      return;
    }
    setLoading(true);
    try {
      await auth.resetPassword(identifier, verifyCode, password);
      setResetSuccess(true);
      setTimeout(() => {
        setTab("login");
        setResetSuccess(false);
        setPassword("");
        setConfirmPassword("");
        setVerifyCode("");
      }, 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "密码重置失败");
    } finally {
      setLoading(false);
    }
  };

  const switchTab = (t: "login" | "register" | "forgot") => {
    setTab(t);
    setError("");
    setResetSuccess(false);
  };

  return (
    <div className="w-full h-screen flex relative">
      <Seo path="/login" title="登录 / 注册 · LLM Gateway" description="登录或注册 LLM Gateway 账户。" noindex />
      {/* Left panel — deep indigo gradient */}
      <div className="w-1/2 flex-shrink-0 relative overflow-hidden flex flex-col justify-center px-16" style={{ background: "var(--hero-gradient)" }}>
        {/* Mesh gradient overlays */}
        <div className="absolute top-0 right-0 w-96 h-96 rounded-full pointer-events-none"
          style={{ background: "radial-gradient(circle, rgba(129,140,248,0.2) 0%, transparent 70%)", transform: "translate(30%, -30%)" }} />
        <div className="absolute bottom-0 left-0 w-72 h-72 rounded-full pointer-events-none"
          style={{ background: "radial-gradient(circle, rgba(167,139,250,0.15) 0%, transparent 70%)", transform: "translate(-30%, 30%)" }} />
        <div className="absolute top-1/2 left-1/2 w-[600px] h-[600px] rounded-full pointer-events-none"
          style={{ background: "radial-gradient(circle, rgba(79,70,229,0.1) 0%, transparent 60%)", transform: "translate(-50%, -50%)" }} />

        {/* Floating geometric shapes */}
        <div className="floating-shape" style={{ width: 40, height: 40, top: "15%", left: "25%", background: "linear-gradient(135deg, rgba(129,140,248,0.3), rgba(167,139,250,0.1))", animationDelay: "0s", borderRadius: "8px", transform: "rotate(15deg)" }} />
        <div className="floating-shape" style={{ width: 24, height: 24, top: "35%", left: "70%", background: "rgba(196,181,253,0.2)", animationDelay: "2s", borderRadius: "50%" }} />
        <div className="floating-shape" style={{ width: 32, height: 32, top: "65%", left: "15%", background: "rgba(129,140,248,0.15)", animationDelay: "4s", borderRadius: "6px", transform: "rotate(45deg)" }} />
        <div className="floating-shape" style={{ width: 18, height: 18, top: "80%", left: "55%", background: "rgba(196,181,253,0.25)", animationDelay: "1s", borderRadius: "50%" }} />
        <div className="floating-shape" style={{ width: 28, height: 28, top: "50%", left: "80%", background: "linear-gradient(135deg, rgba(129,140,248,0.2), transparent)", animationDelay: "3s", borderRadius: "50%" }} />

        {/* Subtle grid pattern */}
        <div className="absolute inset-0 pointer-events-none" style={{
          backgroundImage: "linear-gradient(rgba(255,255,255,0.02) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.02) 1px, transparent 1px)",
          backgroundSize: "60px 60px",
        }} />

        {/* Logo */}
        <div className="relative z-10 flex items-center gap-2.5 mb-16">
          <div className="w-9 h-9 rounded-xl flex items-center justify-center brand-gradient shadow-button">
            <Zap size={18} className="text-white" />
          </div>
          <span className="text-lg font-bold text-white">LLM Gateway</span>
        </div>

        {/* Hero text */}
        <div className="relative z-10 mb-12">
          <h1 className="text-4xl font-extrabold text-white leading-tight mb-2">一个网关</h1>
          <h1 className="text-4xl font-extrabold leading-tight mb-6 brand-gradient-text">连通所有大模型</h1>
          <p className="text-sm leading-relaxed" style={{ color: "rgba(196,181,253,0.8)" }}>
            国内外主流模型统一接入，原生协议透传，<br />按量计费，故障自动切换。
          </p>
        </div>

        {/* Stats */}
        <div className="relative z-10 flex gap-8">
          {[
            { value: "多协议透传", label: "OpenAI / Anthropic / Gemini" },
            { value: "自动故障切换", label: "多上游负载均衡" },
            { value: "支付宝直充", label: "按量计费 · 可开发票" },
          ].map((s) => (
            <div key={s.label} className="px-4 py-3 rounded-xl" style={{ background: "rgba(255,255,255,0.06)" }}>
              <div className="text-lg font-bold text-white">{s.value}</div>
              <div className="text-xs mt-0.5" style={{ color: "rgba(196,181,253,0.7)" }}>{s.label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Right panel */}
      <div className="flex-1 flex flex-col items-center justify-center bg-background">
        <div className="w-[400px]">
          {/* Header */}
          <div className="mb-8">
            <h2 className="text-2xl font-bold text-foreground">
              {tab === "forgot" ? "重置密码" : "欢迎回来"}
            </h2>
            <p className="text-sm text-muted-foreground mt-1.5">
              {tab === "login" ? "登录您的账户" : tab === "register" ? "创建新账户" : "通过短信验证重置密码"}
            </p>
          </div>

          {tab === "forgot" ? (
            /* Forgot password view */
            <div className="slide-up">
              {resetSuccess ? (
                <div className="text-center py-8">
                  <CheckCircle size={48} className="text-success mx-auto mb-4" />
                  <p className="text-lg font-medium text-foreground mb-2">密码重置成功</p>
                  <p className="text-sm text-muted-foreground">正在跳转到登录页面...</p>
                </div>
              ) : (
                <form onSubmit={(e) => { e.preventDefault(); handleResetPassword(); }}>
                  {/* Identifier (Phone or Email) */}
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">用户名</label>
                    <div className="flex gap-2">
                      {!isEmail(identifier) && (
                        <div className="flex items-center px-3 bg-muted border border-border rounded-lg text-sm text-muted-foreground font-medium">
                          +86
                        </div>
                      )}
                      <div className="flex-1 relative">
                        <Phone size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <input
                          className="input-field" style={{ paddingLeft: "2.25rem" }}
                          placeholder="请输入手机号或邮箱"
                          value={identifier}
                          onChange={(e) => setIdentifier(e.target.value)}
                        />
                      </div>
                    </div>
                  </div>

                  {/* Verify code */}
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">验证码</label>
                    <div className="flex gap-2">
                      <div className="flex-1 relative">
                        <Mail size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <input
                          className="input-field" style={{ paddingLeft: "2.25rem" }}
                          placeholder="请输入验证码"
                          value={verifyCode}
                          onChange={(e) => setVerifyCode(e.target.value)}
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => handleSendCode("reset_password")}
                        disabled={countdown > 0 || loading}
                        className={`px-4 rounded-lg text-sm font-medium border transition-all duration-200 whitespace-nowrap ${
                          countdown > 0 || loading
                            ? "bg-muted border-border text-muted-foreground cursor-not-allowed"
                            : "border-primary text-primary hover:bg-primary hover:text-white cursor-pointer"
                        }`}
                      >
                        {countdown > 0 ? `${countdown}s` : "获取验证码"}
                      </button>
                    </div>
                  </div>

                  {/* New password */}
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">新密码</label>
                    <div className="relative">
                      <Lock size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                      <input
                        type={showPwd ? "text" : "password"}
                        className="input-field" style={{ paddingLeft: "2.25rem", paddingRight: "2.5rem" }}
                        placeholder="设置新密码（至少 6 位）"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                      />
                      <button
                        type="button"
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                        onClick={() => setShowPwd((v) => !v)}
                      >
                        {showPwd ? <EyeOff size={15} /> : <Eye size={15} />}
                      </button>
                    </div>
                  </div>

                  {/* Confirm password */}
                  <div className="mb-6">
                    <label className="block text-sm font-medium text-foreground mb-1.5">确认密码</label>
                    <div className="relative">
                      <Lock size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                      <input
                        type={showPwd ? "text" : "password"}
                        className="input-field" style={{ paddingLeft: "2.25rem" }}
                        placeholder="再次输入新密码"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                      />
                    </div>
                  </div>

                  <button
                    type="submit"
                    disabled={loading}
                    className="w-full btn-primary py-2.5 text-sm font-semibold disabled:opacity-60 disabled:cursor-not-allowed"
                  >
                    {loading ? "重置中..." : "重置密码"}
                  </button>

                  {error && (
                    <p className="text-sm text-destructive mt-3 text-center">{error}</p>
                  )}

                  <p className="text-center text-sm text-muted-foreground mt-5">
                    <button type="button" onClick={() => switchTab("login")} className="text-primary font-medium hover:underline inline-flex items-center gap-1">
                      <ArrowLeft size={14} />
                      返回登录
                    </button>
                  </p>
                </form>
              )}
            </div>
          ) : (
            <>
              {/* Tabs — pill style */}
              <div className="flex mb-8 bg-muted rounded-xl p-1">
                {(["login", "register"] as const).map((t) => (
                  <button
                    key={t}
                    onClick={() => switchTab(t)}
                    className={`flex-1 py-2 text-sm font-medium transition-all duration-200 rounded-lg text-center ${
                      tab === t
                        ? "bg-card text-foreground shadow-card"
                        : "text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    {t === "login" ? "登录" : "注册"}
                  </button>
                ))}
              </div>

              {tab === "login" ? (
                <form className="slide-up" onSubmit={(e) => { e.preventDefault(); handleLogin(); }}>
                  {/* Identifier (Phone or Email) */}
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">用户名</label>
                    <div className="flex gap-2">
                      {!isEmail(identifier) && (
                        <div className="flex items-center px-3 bg-muted border border-border rounded-lg text-sm text-muted-foreground font-medium">
                          +86
                        </div>
                      )}
                      <div className="flex-1 relative">
                        <Phone size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <input
                          className="input-field" style={{ paddingLeft: "2.25rem" }}
                          placeholder="请输入手机号或邮箱"
                          value={identifier}
                          onChange={(e) => setIdentifier(e.target.value)}
                        />
                      </div>
                    </div>
                  </div>

                  {/* Password */}
                  <div className="mb-5">
                    <label className="block text-sm font-medium text-foreground mb-1.5">密码</label>
                    <div className="relative">
                      <Lock size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                      <input
                        type={showPwd ? "text" : "password"}
                        className="input-field" style={{ paddingLeft: "2.25rem", paddingRight: "2.5rem" }}
                        placeholder="请输入密码"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                      />
                      <button
                        type="button"
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                        onClick={() => setShowPwd((v) => !v)}
                      >
                        {showPwd ? <EyeOff size={15} /> : <Eye size={15} />}
                      </button>
                    </div>
                  </div>

                  {/* Remember & forgot */}
                  <div className="flex items-center justify-between mb-6">
                    <label className="flex items-center gap-2 cursor-pointer select-none">
                      <input
                        type="checkbox"
                        checked={remember}
                        onChange={(e) => setRemember(e.target.checked)}
                        className="rounded border-border"
                      />
                      <span className="text-sm text-muted-foreground">记住我</span>
                    </label>
                    <button type="button" onClick={() => switchTab("forgot")} className="text-sm text-primary hover:underline">忘记密码?</button>
                  </div>

                  <button
                    type="submit"
                    disabled={loading}
                    className="w-full btn-primary py-2.5 text-sm font-semibold disabled:opacity-60 disabled:cursor-not-allowed"
                  >
                    {loading ? "登录中..." : "登录"}
                  </button>

                  {error && (
                    <p className="text-sm text-destructive mt-3 text-center">{error}</p>
                  )}

                  <p className="text-center text-sm text-muted-foreground mt-5">
                    还没有账号？{" "}
                    <button type="button" onClick={() => switchTab("register")} className="text-primary font-medium hover:underline">
                      立即注册
                    </button>
                  </p>
                </form>
              ) : (
                <form className="slide-up" onSubmit={(e) => { e.preventDefault(); handleRegister(); }}>
                  {/* Identifier (Phone or Email) */}
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">用户名</label>
                    <div className="flex gap-2">
                      {!isEmail(identifier) && (
                        <div className="flex items-center px-3 bg-muted border border-border rounded-lg text-sm text-muted-foreground font-medium">
                          +86
                        </div>
                      )}
                      <div className="flex-1 relative">
                        <Phone size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <input
                          className="input-field" style={{ paddingLeft: "2.25rem" }}
                          placeholder="请输入手机号或邮箱"
                          value={identifier}
                          onChange={(e) => setIdentifier(e.target.value)}
                        />
                      </div>
                    </div>
                  </div>

                  {/* Verify code */}
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">验证码</label>
                    <div className="flex gap-2">
                      <div className="flex-1 relative">
                        <Mail size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <input
                          className="input-field" style={{ paddingLeft: "2.25rem" }}
                          placeholder="请输入验证码"
                          value={verifyCode}
                          onChange={(e) => setVerifyCode(e.target.value)}
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => handleSendCode("register")}
                        disabled={countdown > 0 || loading}
                        className={`px-4 rounded-lg text-sm font-medium border transition-all duration-200 whitespace-nowrap ${
                          countdown > 0 || loading
                            ? "bg-muted border-border text-muted-foreground cursor-not-allowed"
                            : "border-primary text-primary hover:bg-primary hover:text-white cursor-pointer"
                        }`}
                      >
                        {countdown > 0 ? `${countdown}s` : "获取验证码"}
                      </button>
                    </div>
                  </div>

                  {/* Admin init token (only shown during first-time setup) */}
                  {needsAdminSetup && (
                    <div className="mb-4">
                      <label className="block text-sm font-medium text-foreground mb-1.5">
                        <span className="flex items-center gap-1.5">
                          <Shield size={14} className="text-amber-500" />
                          管理员初始化令牌
                        </span>
                      </label>
                      <div className="relative">
                        <Lock size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <input
                          type="password"
                          className="input-field" style={{ paddingLeft: "2.25rem" }}
                          placeholder="输入管理员令牌以获取管理员权限"
                          value={adminToken}
                          onChange={(e) => setAdminToken(e.target.value)}
                        />
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">首次注册可通过令牌获取管理员权限，留空则注册为普通用户</p>
                    </div>
                  )}

                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">邀请码（选填）</label>
                    <input
                      type="text"
                      className="input-field"
                      placeholder="填写邀请码可获得双向奖励"
                      maxLength={8}
                      value={referralCode}
                      onChange={(e) => setReferralCode(e.target.value.toUpperCase())}
                    />
                  </div>

                  {/* Password */}
                  <div className="mb-6">
                    <label className="block text-sm font-medium text-foreground mb-1.5">密码</label>
                    <div className="relative">
                      <Lock size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                      <input
                        type={showPwd ? "text" : "password"}
                        className="input-field" style={{ paddingLeft: "2.25rem", paddingRight: "2.5rem" }}
                        placeholder="设置密码（至少 6 位）"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                      />
                      <button
                        type="button"
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                        onClick={() => setShowPwd((v) => !v)}
                      >
                        {showPwd ? <EyeOff size={15} /> : <Eye size={15} />}
                      </button>
                    </div>
                  </div>

                  <label className="flex items-start gap-2 mb-4 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={acceptAup}
                      onChange={(e) => setAcceptAup(e.target.checked)}
                      className="mt-0.5 rounded border-border"
                    />
                    <span className="text-xs text-muted-foreground leading-relaxed">
                      我已阅读并同意<a href="/docs" target="_blank" className="text-primary hover:underline">使用条款</a>和<a href="/docs" target="_blank" className="text-primary hover:underline">隐私政策</a>
                    </span>
                  </label>

                  <button
                    type="submit"
                    disabled={loading || !acceptAup}
                    className="w-full btn-primary py-2.5 text-sm font-semibold disabled:opacity-60 disabled:cursor-not-allowed"
                  >
                    {loading ? "注册中..." : "注册"}
                  </button>

                  {error && (
                    <p className="text-sm text-destructive mt-3 text-center">{error}</p>
                  )}

                  <p className="text-center text-sm text-muted-foreground mt-5">
                    已有账号？{" "}
                    <button type="button" onClick={() => switchTab("login")} className="text-primary font-medium hover:underline">
                      去登录
                    </button>
                  </p>
                </form>
              )}
            </>
          )}
        </div>
      </div>

      <div className="absolute bottom-3 left-1/2 -translate-x-1/2 z-10">
        <BeianBar />
      </div>
    </div>
  );
};

export default AuthPage;
