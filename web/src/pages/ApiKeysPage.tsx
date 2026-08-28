import { useState } from "react";
import { Plus, Copy, Trash2, Key, Shield, Terminal, CheckCircle, Loader2 } from "lucide-react";
import { useApiKeys, useCreateApiKey, useDeleteApiKey, useApiKeyUsage, useApiKeyUsageToday, useSubscriptionCurrent } from "@/hooks/use-api";

import { PageHeader } from "@/components/ui/page-header";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const ApiKeysPage = () => {
  const { data: keys = [], isLoading } = useApiKeys();
  const { data: keyUsage = [] } = useApiKeyUsage(30);
  const { data: keyUsageToday = [] } = useApiKeyUsageToday();
  const { data: subData } = useSubscriptionCurrent();
  const activeSubs = subData?.subscriptions ?? [];
  const createKey = useCreateApiKey();
  const deleteKey = useDeleteApiKey();
  const [copied, setCopied] = useState<string | null>(null);
  const [copyError, setCopyError] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyPlanId, setNewKeyPlanId] = useState<number | undefined>(undefined);
  const [createdFullKey, setCreatedFullKey] = useState<string | null>(null);

  const handleCopy = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(id);
      setTimeout(() => setCopied(null), 2000);
    } catch {
      // Fallback for non-secure contexts (HTTP without localhost)
      try {
        const textarea = document.createElement("textarea");
        textarea.value = text;
        textarea.style.position = "fixed";
        textarea.style.opacity = "0";
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand("copy");
        document.body.removeChild(textarea);
        setCopied(id);
        setTimeout(() => setCopied(null), 2000);
      } catch {
        setCopyError(true);
        setTimeout(() => setCopyError(false), 3000);
      }
    }
  };

  const handleCreate = async () => {
    if (!newKeyName.trim()) return;
    try {
      const res = await createKey.mutateAsync({ name: newKeyName.trim(), plan_id: newKeyPlanId });
      setCreatedFullKey(res.key);
      setShowCreate(false);
      setNewKeyName("");
      setNewKeyPlanId(undefined);
    } catch {
      // error handled by React Query
    }
  };

  const handleDelete = (id: string, name: string) => {
    if (window.confirm(`确定要停用密钥「${name}」吗？密钥将被停用，但历史用量记录将保留。`)) {
      deleteKey.mutate(id);
    }
  };

  const getKeyUsage = (keyId: string) => {
    return keyUsage.find(u => u.key_id === keyId);
  };

  const getKeyUsageToday = (keyId: string) => {
    return keyUsageToday.find(u => u.key_id === keyId);
  };

  const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

  const curlCode = `curl https://your-gateway-host/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`;

  if (isLoading) {
    return (
      <div className="page-container fade-in flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={28} className="animate-spin text-primary" />
          <span className="text-sm text-muted-foreground">加载中...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container fade-in">
      <PageHeader
        eyebrow="访问凭证"
        title="API 密钥"
        description="统一管理访问凭证、套餐绑定与密钥级消费情况"
        actions={
          <button onClick={() => setShowCreate(true)} className="btn-primary flex items-center gap-2">
            <Plus size={16} />
            创建密钥
          </button>
        }
      />

      {/* Warning banner */}
      <div className="flex items-start gap-3 bg-indigo-50 dark:bg-indigo-500/10 border border-indigo-200 dark:border-indigo-500/20 rounded-xl px-4 py-3.5 mb-6">
        <div className="w-6 h-6 bg-indigo-100 dark:bg-indigo-500/20 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5">
          <Shield size={13} className="text-indigo-500" />
        </div>
        <p className="text-sm text-indigo-700 dark:text-indigo-300 leading-relaxed">
          API 密钥用于调用 LLM Gateway 的 OpenAI 兼容接口。请妥善保管，密钥仅在创建时显示一次，不要泄露给他人。
        </p>
      </div>

      {/* Keys table / empty state */}
      <div className="data-table-card mb-6">
        <div className="px-5 py-4 border-b border-border flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-foreground">密钥列表</div>
            <div className="text-xs text-muted-foreground mt-0.5">按创建时间倒序展示</div>
          </div>
          <div className="rounded-full border border-border bg-muted/50 px-2.5 py-1 text-xs text-muted-foreground">
            {keys.length} 个密钥
          </div>
        </div>

        {keys.length === 0 ? (
          <div className="empty-state py-20">
            <div className="w-14 h-14 bg-primary/8 dark:bg-primary/10 rounded-2xl flex items-center justify-center mb-4">
              <Key size={24} className="text-primary/60" />
            </div>
            <div className="text-sm font-semibold text-foreground mb-1">暂无 API 密钥</div>
            <div className="text-xs text-muted-foreground mb-5">点击右上角按钮创建您的第一个 API 密钥</div>
            <button
              onClick={() => setShowCreate(true)}
              className="btn-primary flex items-center gap-2"
            >
              <Plus size={14} />
              创建密钥
            </button>
          </div>
        ) : (
          <Table className="w-full">
            <TableHeader>
              <TableRow className="table-header">
                <TableHead className="text-left px-5 py-3">名称</TableHead>
                <TableHead className="text-left px-5 py-3">密钥</TableHead>
                <TableHead className="text-left px-5 py-3">适用套餐</TableHead>
                <TableHead className="text-left px-5 py-3">创建时间</TableHead>
                <TableHead className="text-left px-5 py-3">最后使用</TableHead>
                <TableHead className="text-right px-5 py-3">今日消费</TableHead>
                <TableHead className="text-right px-5 py-3">30天消费</TableHead>
                <TableHead className="text-right px-5 py-3">请求次数</TableHead>
                <TableHead className="text-right px-5 py-3">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((k) => {
                const usage = getKeyUsage(k.id);
                const usageToday = getKeyUsageToday(k.id);
                const planName = k.plan_id
                  ? (activeSubs.find(s => s.subscription.plan_id === k.plan_id)?.subscription.plan?.display_name ?? `套餐 ${k.plan_id}`)
                  : "全部模型";
                return (
                  <TableRow key={k.id} className="border-t border-border hover:bg-muted/30 transition-colors">
                    <TableCell className="px-5 py-4 text-sm font-medium text-foreground">{k.name}</TableCell>
                    <TableCell className="px-5 py-4 text-sm font-mono text-muted-foreground">{k.key_prefix}****</TableCell>
                    <TableCell className="px-5 py-4 text-sm text-muted-foreground">{planName}</TableCell>
                    <TableCell className="px-5 py-4 text-sm text-muted-foreground">{formatTime(k.created_at)}</TableCell>
                    <TableCell className="px-5 py-4 text-sm text-muted-foreground">{k.last_used_at ? formatTime(k.last_used_at) : "从未使用"}</TableCell>
                    <TableCell className="px-5 py-4 text-sm text-right font-medium text-foreground">
                      {usageToday ? `¥${usageToday.total_cost.toFixed(2)}` : "-"}
                    </TableCell>
                    <TableCell className="px-5 py-4 text-sm text-right text-muted-foreground">
                      {usage ? `¥${usage.total_cost.toFixed(2)}` : "-"}
                    </TableCell>
                    <TableCell className="px-5 py-4 text-sm text-right text-muted-foreground">
                      {usage ? usage.request_count.toLocaleString() : "-"}
                    </TableCell>
                    <TableCell className="px-5 py-4">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => handleDelete(k.id, k.name)}
                          className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/8 transition-colors"
                          aria-label={`删除密钥 ${k.name}`}
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Quick start */}
      <div className="bg-card border border-border rounded-2xl shadow-card overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex items-center gap-2">
          <Terminal size={16} className="text-muted-foreground" />
          <div className="text-sm font-semibold text-foreground">快速开始</div>
        </div>
        <div className="px-5 py-4">
          <p className="text-sm text-muted-foreground mb-4">
            使用以下 curl 命令测试你的 API 密钥（替换 <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono text-foreground">YOUR_API_KEY</code> 为你的实际密钥）：
          </p>
          <div className="code-block relative">
            <div className="flex items-center justify-between px-4 py-2.5 border-b" style={{ borderColor: "rgba(51,65,85,0.5)" }}>
              <span className="text-xs font-medium" style={{ color: "rgba(100,116,139,1)" }}>BASH</span>
              <button
                onClick={() => handleCopy(curlCode, "curl")}
                className="flex items-center gap-1.5 text-xs transition-colors px-2 py-1 rounded"
                style={{ color: copied === "curl" ? "rgba(16,185,129,1)" : "rgba(100,116,139,1)" }}
              >
                {copied === "curl" ? <CheckCircle size={12} /> : <Copy size={12} />}
                {copied === "curl" ? "已复制" : "复制"}
              </button>
            </div>
            <pre className="px-4 py-4 text-sm overflow-x-auto" style={{ color: "rgba(226,232,240,1)" }}>
              <code>
                <span style={{ color: "rgba(125,211,252,1)" }}>curl</span>
                <span style={{ color: "rgba(226,232,240,1)" }}> https://your-gateway-host/v1/chat/completions \{"\n"}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>  -H </span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"Content-Type: application/json"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}> \{"\n"}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>  -H </span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"Authorization: Bearer `}</span>
                <span style={{ color: "rgba(251,191,36,1)" }}>YOUR_API_KEY</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}> \{"\n"}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`  -d '{\n    `}</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"model"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`: `}</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"gpt-4o"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`,\n    `}</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"messages"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`: [\n      {`}</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"role"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`: `}</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"user"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`, `}</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"content"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`: `}</span>
                <span style={{ color: "rgba(167,243,208,1)" }}>{`"Hello!"`}</span>
                <span style={{ color: "rgba(226,232,240,1)" }}>{`}\n    ]\n  }'`}</span>
              </code>
            </pre>
          </div>
        </div>
      </div>

      {/* Create key modal */}
      <Dialog open={showCreate} onOpenChange={(o) => { if (!o) { setShowCreate(false); setNewKeyName(""); setNewKeyPlanId(undefined); } }}>
        <DialogContent className="sm:max-w-[440px]">
          <DialogHeader>
            <DialogTitle>创建 API 密钥</DialogTitle>
            <DialogDescription>为您的密钥设置一个易于识别的名称</DialogDescription>
          </DialogHeader>
            <label className="block text-sm font-medium text-foreground mb-1.5">密钥名称</label>
            <input
              className="input-field mb-4"
              placeholder="例如：生产环境密钥"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
            />
            <label className="block text-sm font-medium text-foreground mb-1.5">适用套餐</label>
            <select
              className="input-field mb-5"
              value={newKeyPlanId ?? ""}
              onChange={(e) => setNewKeyPlanId(e.target.value ? Number(e.target.value) : undefined)}
            >
              <option value="">不限制（全部模型）</option>
              {activeSubs.map(({ subscription: s }) => (
                <option key={s.plan_id} value={s.plan_id}>
                  {s.plan?.display_name ?? `套餐 ${s.plan_id}`}
                </option>
              ))}
            </select>
            <div className="flex gap-3 justify-end">
              <button onClick={() => { setShowCreate(false); setNewKeyName(""); setNewKeyPlanId(undefined); }} className="btn-secondary">取消</button>
              <button
                onClick={handleCreate}
                disabled={createKey.isPending || !newKeyName.trim()}
                className="btn-primary"
              >
                {createKey.isPending ? "创建中..." : "创建"}
              </button>
            </div>
        </DialogContent>
      </Dialog>

      {/* Key created modal */}
      <Dialog open={!!createdFullKey} onOpenChange={(o) => { if (!o) setCreatedFullKey(null); }}>
        <DialogContent className="sm:max-w-[520px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <CheckCircle size={18} className="text-success" />
              密钥已创建
            </DialogTitle>
            <DialogDescription>请立即复制并妥善保存，此密钥仅显示一次，关闭后将无法再次查看。</DialogDescription>
          </DialogHeader>
            <div className="flex items-center gap-2 bg-muted dark:bg-muted/50 rounded-lg px-4 py-3 mb-5 border border-border">
              <code className="flex-1 text-sm font-mono text-foreground break-all select-all">{createdFullKey}</code>
              <button
                onClick={() => handleCopy(createdFullKey, "created-key")}
                className="flex-shrink-0 btn-secondary flex items-center gap-1.5 text-xs"
              >
                {copied === "created-key" ? <CheckCircle size={12} className="text-success" /> : <Copy size={12} />}
                {copied === "created-key" ? "已复制" : "复制"}
              </button>
            </div>
            {copyError && (
              <p className="text-xs text-destructive -mt-3 mb-3">
                复制失败，请手动选择并复制上方密钥。
              </p>
            )}
            <div className="flex justify-end">
              <button onClick={() => setCreatedFullKey(null)} className="btn-primary">我已保存，关闭</button>
            </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default ApiKeysPage;
