import { useState, useMemo } from "react";
import { Server, Shield, ChevronDown, ChevronRight, Code, Loader } from "lucide-react";
import { useGatewayConfig } from "@/hooks/use-api";

interface ConfigRowProps {
  label: string;
  value: string;
  isSecret?: boolean;
}

const ConfigRow = ({ label = "", value = "", isSecret = false }: ConfigRowProps) => (
  <div className="flex flex-col border-t border-border first:border-t-0 sm:flex-row">
    <div className="w-full shrink-0 border-b border-border bg-muted/30 px-4 py-3 text-sm font-medium text-muted-foreground sm:w-[340px] sm:border-b-0 sm:border-r">
      {label}
    </div>
    <div className={`flex-1 px-4 py-3 text-sm font-mono ${isSecret ? "text-muted-foreground" : "text-foreground"}`}>
      {value}
    </div>
  </div>
);

const AdminConfig = () => {
  const [activeTab, setActiveTab] = useState<"structured" | "json">("structured");
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(["server"]));

  const { data: config, isLoading } = useGatewayConfig();

  const toggleSection = (name: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  // Extract structured sections from config
  const serverConfig = useMemo(() => {
    if (!config) return null;
    const server = config.server as Record<string, unknown> | undefined;
    if (!server) return null;
    return server;
  }, [config]);

  const cbConfig = useMemo(() => {
    if (!config) return null;
    const cb = config.circuit_breaker as Record<string, unknown> | undefined;
    if (!cb) return null;
    return cb;
  }, [config]);

  const modelsConfig = useMemo(() => {
    if (!config) return [];
    const models = config.models as Array<Record<string, unknown>> | undefined;
    return models ?? [];
  }, [config]);

  const jsonFormatted = useMemo(() => {
    if (!config) return "";
    return JSON.stringify(config, null, 2);
  }, [config]);

  // Helper to render unknown values as strings
  const toStr = (v: unknown): string => {
    if (v == null) return "";
    if (typeof v === "string") return v;
    if (typeof v === "number" || typeof v === "boolean") return String(v);
    if (Array.isArray(v)) return v.map(toStr).join(", ");
    return JSON.stringify(v);
  };

  const isSecretKey = (key: string) =>
    key.includes("token") || key.includes("key") || key.includes("secret") || key.includes("password");

  return (
    <div className="page-container">
      {/* Page header */}
      <div className="mb-6">
        <h1 className="text-lg font-bold text-foreground">系统配置</h1>
        <p className="text-sm text-muted-foreground mt-0.5">查看网关运行时配置信息</p>
      </div>

      {/* Tab switcher */}
      <div className="flex items-center gap-1 bg-card border border-border rounded-xl p-1 w-fit shadow-card mb-5">
        <button
          onClick={() => setActiveTab("structured")}
          className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
            activeTab === "structured" ? "bg-primary text-primary-foreground shadow-button" : "text-muted-foreground hover:text-foreground hover:bg-muted"
          }`}
        >
          <Server size={13} />
          结构化视图
        </button>
        <button
          onClick={() => setActiveTab("json")}
          className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
            activeTab === "json" ? "bg-primary text-primary-foreground shadow-button" : "text-muted-foreground hover:text-foreground hover:bg-muted"
          }`}
        >
          <Code size={13} />
          JSON 原文
        </button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader size={20} className="animate-spin text-muted-foreground" />
        </div>
      ) : activeTab === "structured" ? (
        <div className="flex flex-col gap-5">
          {/* Server section */}
          {serverConfig && (
            <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
              <button
                onClick={() => toggleSection("server")}
                className="w-full px-4 py-3.5 border-b border-border flex items-center justify-between bg-muted/20 hover:bg-muted/40 transition-colors"
              >
                <div className="flex items-center gap-2">
                  <Server size={14} className="text-primary" />
                  <span className="text-sm font-semibold text-foreground">Server</span>
                </div>
                {expandedSections.has("server") ? <ChevronDown size={14} className="text-muted-foreground" /> : <ChevronRight size={14} className="text-muted-foreground" />}
              </button>
              {expandedSections.has("server") && (
                <div>
                  {Object.entries(serverConfig).map(([key, val]) => (
                    <ConfigRow key={key} label={key} value={toStr(val)} isSecret={isSecretKey(key)} />
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Circuit Breaker section */}
          {cbConfig && (
            <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
              <button
                onClick={() => toggleSection("circuit_breaker")}
                className="w-full px-4 py-3.5 border-b border-border flex items-center justify-between bg-muted/20 hover:bg-muted/40 transition-colors"
              >
                <div className="flex items-center gap-2">
                  <Shield size={14} className="text-warning" />
                  <span className="text-sm font-semibold text-foreground">Circuit Breaker</span>
                </div>
                {expandedSections.has("circuit_breaker") ? <ChevronDown size={14} className="text-muted-foreground" /> : <ChevronRight size={14} className="text-muted-foreground" />}
              </button>
              {expandedSections.has("circuit_breaker") && (
                <div>
                  {Object.entries(cbConfig).map(([key, val]) => (
                    <ConfigRow key={key} label={key} value={toStr(val)} />
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Models section */}
          {modelsConfig.map((model, idx) => {
            const modelName = toStr(model.name);
            const sectionKey = `model-${idx}`;
            const upstreams = (model.upstreams as Array<Record<string, unknown>>) ?? [];
            return (
              <div key={sectionKey} className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
                <button
                  onClick={() => toggleSection(sectionKey)}
                  className="w-full px-4 py-3.5 border-b border-border flex items-center justify-between bg-muted/20 hover:bg-muted/40 transition-colors"
                >
                  <span className="text-sm font-semibold text-foreground">模型: {modelName}</span>
                  {expandedSections.has(sectionKey) ? <ChevronDown size={14} className="text-muted-foreground" /> : <ChevronRight size={14} className="text-muted-foreground" />}
                </button>
                {expandedSections.has(sectionKey) && upstreams.map((up, upIdx) => (
                  <div key={upIdx}>
                    {Object.entries(up).map(([key, val]) => (
                      <ConfigRow key={key} label={`上游${upIdx + 1} - ${key}`} value={toStr(val)} isSecret={isSecretKey(key)} />
                    ))}
                  </div>
                ))}
              </div>
            );
          })}

          {/* Other config keys not in server/circuit_breaker/models */}
          {config && Object.entries(config)
            .filter(([key]) => !["server", "circuit_breaker", "models"].includes(key))
            .map(([key, val]) => (
              <div key={key} className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
                <button
                  onClick={() => toggleSection(key)}
                  className="w-full px-4 py-3.5 border-b border-border flex items-center justify-between bg-muted/20 hover:bg-muted/40 transition-colors"
                >
                  <span className="text-sm font-semibold text-foreground">{key}</span>
                  {expandedSections.has(key) ? <ChevronDown size={14} className="text-muted-foreground" /> : <ChevronRight size={14} className="text-muted-foreground" />}
                </button>
                {expandedSections.has(key) && (
                  <div className="px-4 py-3">
                    <pre className="text-sm font-mono text-foreground whitespace-pre-wrap">{typeof val === "object" ? JSON.stringify(val, null, 2) : toStr(val)}</pre>
                  </div>
                )}
              </div>
            ))}
        </div>
      ) : (
        /* JSON view */
        <div className="code-block rounded-xl overflow-hidden shadow-card">
          <div className="flex items-center justify-between px-4 py-3 border-b" style={{ borderColor: "rgba(51,65,85,0.5)" }}>
            <span className="text-xs font-medium text-muted-foreground">config.json</span>
            <div className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-full bg-destructive/60" />
              <span className="w-2.5 h-2.5 rounded-full bg-warning/60" />
              <span className="w-2.5 h-2.5 rounded-full bg-success/60" />
            </div>
          </div>
          <pre className="px-5 py-4 text-sm overflow-x-auto" style={{ color: "rgba(226,232,240,1)" }}>
            <code>{jsonFormatted}</code>
          </pre>
        </div>
      )}
    </div>
  );
};

export default AdminConfig;
