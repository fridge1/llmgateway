import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Zap, Eye, EyeOff, User, Lock, Building2 } from "lucide-react";
import { useSubUserAuth } from "@/contexts/SubUserAuthContext";

const SubUserLoginPage = () => {
  const navigate = useNavigate();
  const auth = useSubUserAuth();

  const [tenantId, setTenantId] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPwd, setShowPwd] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  if (auth.isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  if (auth.isAuthenticated) {
    return <Navigate to="/org" replace />;
  }

  const handleLogin = async () => {
    if (loading) return;
    setError("");
    setLoading(true);
    try {
      await auth.login(tenantId, username, password);
      navigate("/org");
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen w-full flex-col lg:flex-row">
      {/* Left panel */}
      <div className="relative flex min-h-[420px] w-full flex-shrink-0 flex-col justify-center overflow-hidden px-6 py-10 sm:px-16 lg:min-h-screen lg:w-1/2 lg:py-0" style={{ background: "linear-gradient(145deg, #0C4A6E 0%, #075985 40%, #0369A1 100%)" }}>

        <div className="relative z-10 mb-12 flex items-center gap-2.5 sm:mb-16">
          <div className="w-9 h-9 rounded-xl flex items-center justify-center" style={{ background: "linear-gradient(135deg, #0EA5E9, #38BDF8)" }}>
            <Building2 size={18} className="text-white" />
          </div>
          <span className="text-lg font-bold text-white">LLM Gateway</span>
        </div>

        <div className="relative z-10 mb-12">
          <h1 className="text-4xl font-extrabold text-white leading-tight mb-2">企业组织</h1>
          <h1 className="text-4xl font-extrabold leading-tight mb-6" style={{ background: "linear-gradient(135deg, #7DD3FC, #BAE6FD)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent" }}>子用户登录</h1>
          <p className="text-sm leading-relaxed" style={{ color: "rgba(186,230,253,0.8)" }}>
            使用组织管理员分配的账号登录。<br />
            查看可用模型、管理 API 密钥、追踪使用额度。
          </p>
        </div>

        <div className="relative z-10 flex flex-wrap gap-3 sm:gap-8">
          {[
            { value: "API 密钥", label: "自主管理" },
            { value: "额度管控", label: "独立配额" },
            { value: "使用明细", label: "交易可查" },
          ].map((s) => (
            <div key={s.label} className="px-4 py-3 rounded-xl" style={{ background: "rgba(255,255,255,0.06)" }}>
              <div className="text-lg font-bold text-white">{s.value}</div>
              <div className="text-xs mt-0.5" style={{ color: "rgba(186,230,253,0.7)" }}>{s.label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Right panel */}
      <div className="flex flex-1 flex-col items-center justify-center bg-background px-4 py-10 sm:px-6">
        <div className="w-full max-w-[400px]">
          <div className="mb-8">
            <h2 className="text-2xl font-bold text-foreground">子用户登录</h2>
            <p className="text-sm text-muted-foreground mt-1.5">请输入组织 ID 和分配的账号密码</p>
          </div>

          <form className="slide-up" onSubmit={(e) => { e.preventDefault(); handleLogin(); }}>
            {/* Tenant ID */}
            <div className="mb-4">
              <label className="block text-sm font-medium text-foreground mb-1.5">组织 ID</label>
              <div className="relative">
                <Building2 size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                <input
                  className="input-field" style={{ paddingLeft: "2.25rem" }}
                  placeholder="请输入组织 ID"
                  value={tenantId}
                  onChange={(e) => setTenantId(e.target.value)}
                />
              </div>
            </div>

            {/* Username */}
            <div className="mb-4">
              <label className="block text-sm font-medium text-foreground mb-1.5">用户名</label>
              <div className="relative">
                <User size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                <input
                  className="input-field" style={{ paddingLeft: "2.25rem" }}
                  placeholder="请输入用户名"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                />
              </div>
            </div>

            {/* Password */}
            <div className="mb-6">
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
          </form>
        </div>
      </div>
    </div>
  );
};

export default SubUserLoginPage;
