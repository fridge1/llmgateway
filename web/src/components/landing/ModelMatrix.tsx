import { useState } from "react";
import { usePublicModels } from "@/hooks/use-public";
import { Skeleton } from "@/components/ui/skeleton";
import { Cpu } from "lucide-react";

const PROVIDER_LABEL: Record<string, string> = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  google: "Google",
  gemini: "Google",
  volcengine: "火山引擎",
  deepseek: "DeepSeek",
  dashscope: "阿里百炼",
  siliconflow: "SiliconFlow",
  xai: "xAI",
  other: "其他",
};

const PROVIDER_COLOR: Record<string, string> = {
  openai: "from-emerald-500 to-teal-500",
  anthropic: "from-orange-500 to-amber-500",
  google: "from-blue-500 to-indigo-500",
  gemini: "from-blue-500 to-indigo-500",
  volcengine: "from-rose-500 to-red-500",
  deepseek: "from-blue-500 to-indigo-500",
  dashscope: "from-amber-500 to-orange-500",
  siliconflow: "from-purple-500 to-fuchsia-500",
  xai: "from-slate-600 to-slate-700",
  other: "from-slate-500 to-slate-600",
};

const ModelMatrix = () => {
  const { data, isLoading } = usePublicModels();
  const providers = data?.providers ?? [];
  const [activeIdx, setActiveIdx] = useState(0);

  const active = providers[activeIdx];

  return (
    <section id="models" className="py-20">
      <div className="max-w-6xl mx-auto px-6">
        <div className="text-center mb-12">
          <h2 className="text-3xl md:text-4xl font-bold text-foreground tracking-tight mb-3">
            一个网关，多家模型
          </h2>
          <p className="text-muted-foreground">
            国内外主流模型统一接入，按需切换，无需重写客户端代码
          </p>
        </div>

        {isLoading ? (
          <div className="space-y-4">
            <div className="flex justify-center gap-2">
              <Skeleton className="h-9 w-24" />
              <Skeleton className="h-9 w-24" />
              <Skeleton className="h-9 w-24" />
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {Array.from({ length: 8 }).map((_, i) => (
                <Skeleton key={i} className="h-20" />
              ))}
            </div>
          </div>
        ) : providers.length === 0 ? (
          <p className="text-center text-muted-foreground">暂无可用模型</p>
        ) : (
          <>
            <div className="flex flex-wrap justify-center gap-2 mb-8 max-w-full overflow-x-auto">
              {providers.map((p, idx) => (
                <button
                  key={p.provider}
                  onClick={() => setActiveIdx(idx)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                    idx === activeIdx
                      ? "bg-primary text-primary-foreground shadow-sm"
                      : "bg-muted/60 text-muted-foreground hover:bg-muted"
                  }`}
                >
                  {PROVIDER_LABEL[p.provider] ?? p.provider}
                  <span className="ml-1.5 text-xs opacity-70">{p.models.length}</span>
                </button>
              ))}
            </div>

            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
              {active?.models.map((m) => (
                <div
                  key={m.id}
                  className="group relative bg-card border border-border rounded-xl p-4 hover:shadow-elevated hover:border-primary/40 transition-all"
                >
                  <div className={`w-9 h-9 rounded-lg bg-gradient-to-br ${PROVIDER_COLOR[active.provider] ?? "from-slate-500 to-slate-600"} flex items-center justify-center mb-3`}>
                    <Cpu size={16} className="text-white" />
                  </div>
                  <div className="font-semibold text-sm text-foreground truncate" title={m.display_name}>
                    {m.display_name}
                  </div>
                  <div className="text-xs text-muted-foreground truncate mt-0.5" title={m.id}>
                    {m.id}
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </section>
  );
};

export default ModelMatrix;
