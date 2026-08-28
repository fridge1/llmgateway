import { useState, useEffect } from "react";
import { AnimatePresence, motion } from "motion/react";
import type { ToolStatus, Model } from "../lib/types";
import type { ConfigResult } from "../lib/tauri";
import type { LucideIcon } from "./icons";
import { Terminal, Code2, Bot, Zap, ChevronDown, Wrench, Trash2, Loader2, CheckCircle2, AlertCircle } from "./icons";
import Badge from "./Badge";
import Card from "./Card";
import { isWindows } from "../lib/utils";

const TOOL_NAMES: Record<string, string> = {
  claude_code: "Claude Code",
  codex_cli: "Codex CLI",
  openclaw: "OpenClaw",
  hermes_agent: "Hermes Agent",
};

const TOOL_ICONS: Record<string, LucideIcon> = {
  claude_code: Terminal,
  codex_cli: Code2,
  openclaw: Bot,
  hermes_agent: Zap,
};

interface Props {
  tool: ToolStatus;
  models: Model[];
  modelsLoading: boolean;
  onConfigure: (tool: string, model?: string) => Promise<ConfigResult>;
  onClear: (tool: string) => Promise<void>;
}

export default function ToolCard({ tool, models, modelsLoading, onConfigure, onClear }: Props) {
  const name = TOOL_NAMES[tool.tool] || tool.tool;
  const Icon = TOOL_ICONS[tool.tool];
  const borderClass = tool.configured ? "border-l-emerald-700/50" : "border-l-amber-700/50";

  const [expanded, setExpanded] = useState(false);
  const [selectedModel, setSelectedModel] = useState(tool.current_config?.current_model || "");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  useEffect(() => {
    setSelectedModel(tool.current_config?.current_model || "");
  }, [tool.current_config?.current_model]);

  const handleConfigure = async () => {
    setBusy(true);
    setMessage(null);
    try {
      const result = await onConfigure(tool.tool, selectedModel || undefined);
      if (result.success) {
        setMessage({ type: "success", text: isWindows ? "配置成功。请新开终端窗口后生效" : "配置成功。已打开的终端需要新开窗口或执行 source ~/.zshrc 后生效" });
      } else {
        setMessage({ type: "error", text: result.message });
      }
    } catch (err) {
      setMessage({ type: "error", text: "配置失败: " + String(err) });
    } finally {
      setBusy(false);
    }
  };

  const handleClear = async () => {
    setBusy(true);
    setMessage(null);
    try {
      await onClear(tool.tool);
      setSelectedModel("");
      setMessage({ type: "success", text: "配置已清除" });
    } catch (err) {
      setMessage({ type: "error", text: "清除失败: " + String(err) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card hover={false} className={`border-l-[3px] ${borderClass}`}>
      <div
        className="flex items-center justify-between cursor-pointer select-none"
        onClick={() => setExpanded(v => !v)}
      >
        <div className="flex items-center gap-2">
          {Icon && <Icon size={20} className="text-obsidian-400" />}
          <span className="font-semibold text-obsidian-100">{name}</span>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={tool.configured ? "success" : "warning"} dot>
            {tool.configured ? "已配置" : "未配置"}
          </Badge>
          <motion.div
            animate={{ rotate: expanded ? 180 : 0 }}
            transition={{ duration: 0.2 }}
          >
            <ChevronDown size={16} className="text-obsidian-500" />
          </motion.div>
        </div>
      </div>

      <div className="flex items-center gap-3 mt-2">
        <p className="font-mono text-xs text-obsidian-500 truncate" title={tool.path}>
          {tool.path}
        </p>
        <p className="font-mono text-xs text-obsidian-600 shrink-0">{tool.version}</p>
      </div>

      {tool.current_config?.current_model && !expanded && (
        <p className="text-xs text-obsidian-500 mt-1">
          模型: <span className="text-obsidian-300">{tool.current_config.current_model}</span>
        </p>
      )}

      <AnimatePresence>
        {expanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden"
          >
            <div className="mt-3 pt-3 border-t border-obsidian-700/50">
              {/* Model selector */}
              <div className="mb-3">
                <label className="text-xs text-obsidian-500 mb-1 block">默认模型</label>
                <select
                  value={selectedModel}
                  onChange={e => setSelectedModel(e.target.value)}
                  disabled={busy || modelsLoading}
                  className="w-full bg-obsidian-800 border border-obsidian-700 rounded-lg px-3 py-2 text-sm text-obsidian-100 focus:border-amber-500/50 focus:outline-none disabled:opacity-50 appearance-none"
                >
                  <option value="">使用工具默认</option>
                  {models.map(m => (
                    <option key={m.id} value={m.id}>{m.id}</option>
                  ))}
                </select>
              </div>

              {/* Action buttons */}
              <div className="flex gap-2">
                <button
                  onClick={handleConfigure}
                  disabled={busy}
                  className="flex-1 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 rounded-lg text-sm font-semibold transition-colors inline-flex items-center justify-center gap-1.5"
                >
                  {busy ? (
                    <><Loader2 size={14} className="animate-spin" /> 配置中...</>
                  ) : (
                    <><Wrench size={14} /> {tool.configured ? "重新配置" : "配置"}</>
                  )}
                </button>
                {tool.configured && (
                  <button
                    onClick={handleClear}
                    disabled={busy}
                    className="py-2 px-3 border border-obsidian-700 hover:border-red-500/50 hover:text-red-400 disabled:opacity-50 text-obsidian-400 rounded-lg text-sm transition-colors inline-flex items-center gap-1.5"
                  >
                    <Trash2 size={14} />
                    清除
                  </button>
                )}
              </div>

              {/* Feedback message */}
              {message && (
                <div className={`flex items-start gap-2 p-2 rounded-lg mt-3 text-xs ${
                  message.type === "success"
                    ? "bg-emerald-500/10 text-emerald-400"
                    : "bg-red-500/10 text-red-400"
                }`}>
                  {message.type === "success"
                    ? <CheckCircle2 size={14} className="mt-0.5 shrink-0" />
                    : <AlertCircle size={14} className="mt-0.5 shrink-0" />
                  }
                  <span>{message.text}</span>
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </Card>
  );
}
