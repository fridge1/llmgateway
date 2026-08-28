import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CheckCircle2, Copy, Check, Loader2, Terminal, Sparkles, ArrowRight, Zap, Eye, EyeOff } from "lucide-react";
import { apiPost } from "@/lib/api-client";
import { useApiKeys } from "@/hooks/use-api";
import { useAuth } from "@/contexts/AuthContext";
import type { CreateKeyResponse } from "@/types/api";

const BASE_URL = "https://your-domain.com";
const TEST_MODEL = "claude-haiku-4-5-20251001";

type Tool = "claude-code" | "cursor" | "codex";

function copy(text: string): Promise<boolean> {
  return navigator.clipboard.writeText(text).then(() => true).catch(() => {
    try {
      const ta = document.createElement("textarea");
      ta.value = text; ta.style.position = "fixed"; ta.style.opacity = "0";
      document.body.appendChild(ta); ta.select();
      const ok = document.execCommand("copy");
      document.body.removeChild(ta);
      return ok;
    } catch { return false; }
  });
}

const OnboardingPage = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { data: existingKeys, isLoading: keysLoading } = useApiKeys();
  const [created, setCreated] = useState<CreateKeyResponse | null>(null);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [showKey, setShowKey] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const [tool, setTool] = useState<Tool>("claude-code");

  // Auto-create one onboarding key for first-time users
  const triedRef = useRef(false);
  useEffect(() => {
    if (triedRef.current) return;
    if (keysLoading) return;
    if (created) return;
    if ((existingKeys?.length ?? 0) > 0) return; // already has keys, skip
    triedRef.current = true;
    setCreating(true);
    apiPost<CreateKeyResponse>("/api/keys", { name: "快速开始" })
      .then((r) => setCreated(r))
      .catch((e) => setCreateError(e instanceof Error ? e.message : "创建失败"))
      .finally(() => setCreating(false));
  }, [existingKeys, keysLoading, created]);

  const apiKey = created?.key ?? "";
  const keyId = created?.api_key?.id ?? "";
  const maskedKey = apiKey ? apiKey.slice(0, 7) + "•".repeat(Math.max(0, apiKey.length - 11)) + apiKey.slice(-4) : "";

  const onCopy = async (label: string, text: string) => {
    if (!text) return;
    if (await copy(text)) {
      setCopiedField(label);
      setTimeout(() => setCopiedField(null), 1800);
    }
  };

  // ---- Test stream ----
  const [testing, setTesting] = useState(false);
  const [testText, setTestText] = useState("");
  const [testDone, setTestDone] = useState(false);
  const [testError, setTestError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const runTest = async () => {
    if (testing || !keyId) return;
    setTesting(true); setTestText(""); setTestDone(false); setTestError(null);
    const controller = new AbortController();
    abortRef.current = controller;
    try {
      const res = await fetch("/api/playground/completions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          key_id: keyId,
          model: TEST_MODEL,
          messages: [{ role: "user", content: "用一句中文向我打招呼，并说明你已经接入成功。" }],
          stream: true,
          max_tokens: 80,
        }),
        signal: controller.signal,
      });
      if (!res.ok) {
        const t = await res.text();
        try { const j = JSON.parse(t); throw new Error(j?.error?.message || res.statusText); }
        catch { throw new Error(t || res.statusText); }
      }
      const reader = res.body!.getReader();
      const dec = new TextDecoder();
      let buf = "";
      let acc = "";
      const flush = (lines: string[]) => {
        for (const ln of lines) {
          const t = ln.trim();
          if (!t.startsWith("data:")) continue;
          const data = t.slice(5).trim();
          if (data === "[DONE]") continue;
          try {
            const c = JSON.parse(data);
            const d = c?.choices?.[0]?.delta?.content;
            if (d) { acc += d; setTestText(acc); }
          } catch { /* ignore */ }
        }
      };
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true });
        const parts = buf.split("\n");
        buf = parts.pop() || "";
        flush(parts);
      }
      if (buf.trim()) flush([buf]);
      setTestDone(true);
    } catch (e) {
      const isAbort = e instanceof Error && e.name === "AbortError";
      if (!isAbort) setTestError(e instanceof Error ? e.message : "测试失败");
    } finally {
      setTesting(false);
    }
  };
  useEffect(() => () => abortRef.current?.abort(), []);

  // ---- Snippets ----
  const snippets: Record<Tool, { title: string; lang: string; code: string; desc: string }> = useMemo(() => ({
    "claude-code": {
      title: "Claude Code",
      lang: "bash",
      desc: "复制下方命令，粘贴到终端运行一次。下次打开 claude 即可使用。",
      code: `export ANTHROPIC_BASE_URL="${BASE_URL}"
export ANTHROPIC_AUTH_TOKEN="${apiKey || "<你的 API Key>"}"
export ANTHROPIC_MODEL="claude-sonnet-4-6"

# 启动
claude`,
    },
    cursor: {
      title: "Cursor",
      lang: "text",
      desc: "Cursor → Settings → Models → 添加自定义 OpenAI Base URL。",
      code: `Base URL:  ${BASE_URL}/v1
API Key:   ${apiKey || "<你的 API Key>"}
Model:     claude-sonnet-4-6`,
    },
    codex: {
      title: "Codex CLI / OpenAI 兼容",
      lang: "bash",
      desc: "任何兼容 OpenAI 协议的工具都可以接入。",
      code: `export OPENAI_BASE_URL="${BASE_URL}/v1"
export OPENAI_API_KEY="${apiKey || "<你的 API Key>"}"

# curl 测试
curl ${BASE_URL}/v1/chat/completions \\
  -H "Authorization: Bearer ${apiKey || "<你的 API Key>"}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "messages": [{"role":"user","content":"hi"}]
  }'`,
    },
  }), [apiKey]);

  const initBusy = keysLoading || creating;
  const hasExistingKeys = !created && (existingKeys?.length ?? 0) > 0;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-indigo-50 dark:from-slate-950 dark:to-indigo-950">
      <div className="max-w-4xl mx-auto px-6 py-12">
        {/* Header */}
        <div className="flex items-center gap-2 mb-8">
          <div className="w-9 h-9 rounded-xl flex items-center justify-center bg-gradient-to-br from-indigo-500 to-violet-500">
            <Zap size={18} className="text-white" />
          </div>
          <span className="text-lg font-bold">LLM Gateway</span>
        </div>

        <div className="mb-10">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-100 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-xs font-medium mb-4">
            <Sparkles size={14} />
            欢迎，已为你发放 ¥5 试用金
          </div>
          <h1 className="text-3xl md:text-4xl font-bold text-foreground leading-tight">
            60 秒接入 Claude Code
          </h1>
          <p className="text-muted-foreground mt-2">
            照着下面三步走，第一条请求就能跑通。{user?.phone ? `账号 ${user.phone}` : ""}
          </p>
        </div>

        {/* Step 1: API Key */}
        <Section index={1} title="你的 API Key" subtitle="已为你自动创建，妥善保管，不要分享给他人">
          {initBusy ? (
            <div className="flex items-center gap-2 text-muted-foreground py-6">
              <Loader2 className="animate-spin" size={16} /> 正在为你生成 API Key...
            </div>
          ) : createError ? (
            <div className="text-sm text-destructive py-4">
              创建失败：{createError}
              <button
                className="ml-3 underline"
                onClick={() => { triedRef.current = false; setCreateError(null); }}
              >重试</button>
            </div>
          ) : hasExistingKeys ? (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">检测到你已有 API Key。出于安全考虑无法回显已有 Key 的明文，可生成一个新的：</p>
              <button
                onClick={() => { triedRef.current = false; setCreated(null); /* trigger effect */ setCreating(true);
                  apiPost<CreateKeyResponse>("/api/keys", { name: "快速开始" })
                    .then(setCreated).catch((e)=>setCreateError(e instanceof Error ? e.message : "创建失败"))
                    .finally(()=>setCreating(false));
                }}
                className="px-4 py-2 rounded-lg bg-primary text-white text-sm font-medium hover:opacity-90"
              >生成一个新的 Key</button>
            </div>
          ) : (
            <div className="space-y-3">
              <Field label="Base URL" value={BASE_URL} onCopy={() => onCopy("base", BASE_URL)} copied={copiedField === "base"} />
              <Field
                label="API Key"
                value={showKey ? apiKey : maskedKey}
                onCopy={() => onCopy("key", apiKey)}
                copied={copiedField === "key"}
                trailing={
                  <button onClick={() => setShowKey(v => !v)} className="text-muted-foreground hover:text-foreground" aria-label="toggle visibility">
                    {showKey ? <EyeOff size={15} /> : <Eye size={15} />}
                  </button>
                }
              />
            </div>
          )}
        </Section>

        {/* Step 2: tool config */}
        <Section index={2} title="选择你的工具" subtitle="复制配置，粘贴运行——3 个工具任选其一">
          <div className="flex gap-1 mb-4 p-1 bg-muted rounded-lg w-fit">
            {(Object.keys(snippets) as Tool[]).map(t => (
              <button
                key={t}
                onClick={() => setTool(t)}
                className={`px-4 py-1.5 rounded-md text-sm font-medium transition ${tool === t ? "bg-card shadow-sm text-foreground" : "text-muted-foreground hover:text-foreground"}`}
              >{snippets[t].title}</button>
            ))}
          </div>
          <p className="text-sm text-muted-foreground mb-3">{snippets[tool].desc}</p>
          <div className="relative group">
            <pre className="bg-slate-950 text-slate-100 rounded-xl p-4 text-xs leading-relaxed overflow-x-auto font-mono"><code>{snippets[tool].code}</code></pre>
            <button
              onClick={() => onCopy(`code-${tool}`, snippets[tool].code)}
              className="absolute top-3 right-3 px-2.5 py-1.5 rounded-md bg-white/10 hover:bg-white/20 text-white text-xs flex items-center gap-1.5 backdrop-blur"
              disabled={!apiKey}
            >
              {copiedField === `code-${tool}` ? <><Check size={13} />已复制</> : <><Copy size={13} />复制</>}
            </button>
          </div>
        </Section>

        {/* Step 3: live test */}
        <Section index={3} title="在线测试" subtitle="不离开网页，立刻验证 API 是否通">
          <div className="flex flex-wrap items-center gap-3 mb-4">
            <button
              onClick={runTest}
              disabled={testing || !keyId}
              className="px-5 py-2.5 rounded-lg bg-gradient-to-r from-indigo-500 to-violet-500 text-white text-sm font-semibold disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2 shadow-md hover:shadow-lg transition"
            >
              {testing ? <><Loader2 size={15} className="animate-spin" /> 测试中...</>
                : testDone ? <><CheckCircle2 size={15} /> 再测一次</>
                : <><Terminal size={15} /> 立即测试 API</>}
            </button>
            <span className="text-xs text-muted-foreground">使用 {TEST_MODEL}，单次约 ¥0.001，从试用金扣减</span>
          </div>

          {(testText || testError || testDone) && (
            <div className={`rounded-xl border p-4 text-sm ${testError ? "bg-destructive/5 border-destructive/30" : testDone ? "bg-emerald-50 dark:bg-emerald-500/5 border-emerald-200 dark:border-emerald-500/20" : "bg-muted border-border"}`}>
              {testError ? (
                <div className="text-destructive">
                  <div className="font-medium mb-1">测试失败</div>
                  <div className="text-xs">{testError}</div>
                </div>
              ) : (
                <>
                  {testDone && (
                    <div className="flex items-center gap-2 text-emerald-700 dark:text-emerald-400 font-medium mb-2">
                      <CheckCircle2 size={16} /> 接入成功，第一条请求已通
                    </div>
                  )}
                  <div className="whitespace-pre-wrap text-foreground leading-relaxed">{testText}{testing && <span className="inline-block w-1.5 h-4 bg-foreground/60 animate-pulse ml-0.5 align-middle" />}</div>
                </>
              )}
            </div>
          )}
        </Section>

        {/* CTA */}
        <div className="mt-12 flex items-center justify-between rounded-2xl bg-card border border-border p-6 shadow-card">
          <div>
            <p className="font-semibold text-foreground">完成了？进入控制台查看用量、充值和更多模型</p>
            <p className="text-sm text-muted-foreground mt-1">遇到问题可在控制台联系客服</p>
          </div>
          <button
            onClick={() => navigate("/dashboard")}
            className="px-5 py-2.5 rounded-lg bg-foreground text-background text-sm font-semibold flex items-center gap-2 hover:opacity-90"
          >
            进入控制台 <ArrowRight size={15} />
          </button>
        </div>
      </div>
    </div>
  );
};

function Section({ index, title, subtitle, children }: { index: number; title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <section className="mb-8 rounded-2xl bg-card border border-border p-6 shadow-card">
      <div className="flex items-start gap-3 mb-4">
        <div className="w-7 h-7 rounded-full bg-indigo-100 dark:bg-indigo-500/15 text-indigo-600 dark:text-indigo-400 flex items-center justify-center text-sm font-bold flex-shrink-0">{index}</div>
        <div>
          <h2 className="text-lg font-bold text-foreground">{title}</h2>
          <p className="text-sm text-muted-foreground mt-0.5">{subtitle}</p>
        </div>
      </div>
      {children}
    </section>
  );
}

function Field({ label, value, onCopy, copied, trailing }: { label: string; value: string; onCopy: () => void; copied: boolean; trailing?: React.ReactNode }) {
  return (
    <div>
      <div className="text-xs font-medium text-muted-foreground mb-1">{label}</div>
      <div className="flex items-center gap-2 bg-muted rounded-lg px-3 py-2">
        <code className="flex-1 text-sm font-mono text-foreground truncate">{value}</code>
        {trailing}
        <button
          onClick={onCopy}
          className="px-2.5 py-1 rounded-md bg-card border border-border text-xs font-medium hover:bg-accent flex items-center gap-1.5"
        >
          {copied ? <><Check size={12} className="text-emerald-500" />已复制</> : <><Copy size={12} />复制</>}
        </button>
      </div>
    </div>
  );
}

export default OnboardingPage;
