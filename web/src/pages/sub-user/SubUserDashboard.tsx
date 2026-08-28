import { useSubUserAuth } from "@/contexts/SubUserAuthContext";
import { Key, ArrowLeftRight, Gauge } from "lucide-react";

const SubUserDashboard = () => {
  const { user } = useSubUserAuth();

  if (!user) return null;

  const hasQuotaLimit = user.quota_limit != null;
  const usagePercent = hasQuotaLimit && user.quota_limit! > 0
    ? Math.min((user.quota_used / user.quota_limit!) * 100, 100)
    : 0;

  return (
    <div>
      <h1 className="text-2xl font-bold text-foreground mb-6">欢迎, {user.nickname || user.username}</h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-card border border-border/60 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
              <Gauge size={20} className="text-primary" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">已用额度</p>
              <p className="text-xl font-bold text-foreground">{user.quota_used.toFixed(4)} 元</p>
            </div>
          </div>
          {hasQuotaLimit && (
            <div>
              <div className="flex justify-between text-xs text-muted-foreground mb-1">
                <span>使用进度</span>
                <span>{usagePercent.toFixed(1)}%</span>
              </div>
              <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full rounded-full transition-all duration-300"
                  style={{
                    width: `${usagePercent}%`,
                    background: usagePercent > 80
                      ? "linear-gradient(90deg, #f97316, #ef4444)"
                      : "linear-gradient(90deg, #6366f1, #818cf8)",
                  }}
                />
              </div>
            </div>
          )}
        </div>

        <div className="bg-card border border-border/60 rounded-xl p-6">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 flex items-center justify-center">
              <span className="text-emerald-500 font-bold text-lg">¥</span>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">剩余额度</p>
              <p className="text-xl font-bold text-foreground">
                {hasQuotaLimit ? `${user.quota_remaining?.toFixed(4)} 元` : "无限制"}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-card border border-border/60 rounded-xl p-6">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-amber-500/10 flex items-center justify-center">
              <span className="text-amber-500 font-bold text-lg">¥</span>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">总配额</p>
              <p className="text-xl font-bold text-foreground">
                {hasQuotaLimit ? `${user.quota_limit!.toFixed(2)} 元` : "无限制"}
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-card border border-border/60 rounded-xl p-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">快速入口</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <a href="/org/keys" className="flex items-center gap-3 p-4 rounded-lg bg-muted/40 hover:bg-muted/60 transition-colors">
            <Key size={20} className="text-primary" />
            <div>
              <p className="text-sm font-medium text-foreground">API 密钥</p>
              <p className="text-xs text-muted-foreground">创建和管理你的 API 密钥</p>
            </div>
          </a>
          <a href="/org/transactions" className="flex items-center gap-3 p-4 rounded-lg bg-muted/40 hover:bg-muted/60 transition-colors">
            <ArrowLeftRight size={20} className="text-primary" />
            <div>
              <p className="text-sm font-medium text-foreground">使用记录</p>
              <p className="text-xs text-muted-foreground">查看 API 调用和消费明细</p>
            </div>
          </a>
        </div>
      </div>
    </div>
  );
};

export default SubUserDashboard;
