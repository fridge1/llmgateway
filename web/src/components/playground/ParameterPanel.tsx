import { Plus, Trash2, MessageSquare } from "lucide-react";
import type { ChatSession, GatewayModel, APIKey } from "@/types/api";

interface ParameterPanelProps {
  models: GatewayModel[];
  apiKeys: APIKey[];
  selectedModel: string;
  selectedKeyId: string;
  onModelChange: (model: string) => void;
  onKeyChange: (keyId: string) => void;
  temperature: number;
  maxTokens: number;
  topP: number;
  systemPrompt: string;
  onTemperatureChange: (v: number) => void;
  onMaxTokensChange: (v: number) => void;
  onTopPChange: (v: number) => void;
  onSystemPromptChange: (v: string) => void;
  sessions: ChatSession[];
  activeSessionId: string | null;
  onSelectSession: (id: string) => void;
  onCreateSession: () => void;
  onDeleteSession: (id: string) => void;
}

const ParameterPanel = ({
  models,
  apiKeys,
  selectedModel,
  selectedKeyId,
  onModelChange,
  onKeyChange,
  temperature,
  maxTokens,
  topP,
  systemPrompt,
  onTemperatureChange,
  onMaxTokensChange,
  onTopPChange,
  onSystemPromptChange,
  sessions,
  activeSessionId,
  onSelectSession,
  onCreateSession,
  onDeleteSession,
}: ParameterPanelProps) => {
  return (
    <div className="w-72 h-full border-r border-border bg-card flex flex-col overflow-hidden">
      {/* Parameters section */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* API Key selector */}
        <div>
          <label className="text-xs font-medium text-muted-foreground mb-1.5 block">API 密钥</label>
          <select
            value={selectedKeyId}
            onChange={(e) => onKeyChange(e.target.value)}
            className="w-full h-8 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">选择密钥...</option>
            {apiKeys.map((k) => (
              <option key={k.id} value={k.id}>
                {k.name} ({k.key_prefix}...)
              </option>
            ))}
          </select>
        </div>

        {/* Model selector */}
        <div>
          <label className="text-xs font-medium text-muted-foreground mb-1.5 block">模型</label>
          <select
            value={selectedModel}
            onChange={(e) => onModelChange(e.target.value)}
            className="w-full h-8 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">选择模型...</option>
            {models.map((m) => (
              <option key={m.name} value={m.name}>
                {m.display_name || m.name}
              </option>
            ))}
          </select>
        </div>

        {/* Temperature */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-xs font-medium text-muted-foreground">Temperature</label>
            <span className="text-xs text-muted-foreground tabular-nums">{temperature.toFixed(1)}</span>
          </div>
          <input
            type="range"
            min={0}
            max={2}
            step={0.1}
            value={temperature}
            onChange={(e) => onTemperatureChange(parseFloat(e.target.value))}
            className="w-full accent-primary"
          />
        </div>

        {/* Max Tokens */}
        <div>
          <label className="text-xs font-medium text-muted-foreground mb-1.5 block">Max Tokens</label>
          <input
            type="number"
            min={1}
            max={128000}
            value={maxTokens}
            onChange={(e) => onMaxTokensChange(parseInt(e.target.value) || 4096)}
            className="w-full h-8 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Top P */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-xs font-medium text-muted-foreground">Top P</label>
            <span className="text-xs text-muted-foreground tabular-nums">{topP.toFixed(2)}</span>
          </div>
          <input
            type="range"
            min={0}
            max={1}
            step={0.01}
            value={topP}
            onChange={(e) => onTopPChange(parseFloat(e.target.value))}
            className="w-full accent-primary"
          />
        </div>

        {/* System Prompt */}
        <div>
          <label className="text-xs font-medium text-muted-foreground mb-1.5 block">System Prompt</label>
          <textarea
            value={systemPrompt}
            onChange={(e) => onSystemPromptChange(e.target.value)}
            placeholder="You are a helpful assistant..."
            rows={3}
            className="w-full px-2 py-1.5 text-sm border border-border rounded-lg bg-background text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </div>

      {/* Sessions section */}
      <div className="border-t border-border">
        <div className="flex items-center justify-between px-4 py-2">
          <span className="text-xs font-medium text-muted-foreground">会话列表</span>
          <button
            onClick={onCreateSession}
            className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            <Plus size={14} />
          </button>
        </div>
        <div className="max-h-48 overflow-y-auto px-2 pb-2">
          {sessions.length === 0 && (
            <div className="text-xs text-muted-foreground text-center py-4">暂无会话</div>
          )}
          {sessions.map((s) => (
            <div
              key={s.id}
              onClick={() => onSelectSession(s.id)}
              className={`flex items-center gap-2 px-2 py-1.5 rounded-lg text-sm cursor-pointer group transition-colors ${
                activeSessionId === s.id
                  ? "bg-accent text-foreground"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              }`}
            >
              <MessageSquare size={13} className="shrink-0" />
              <span className="truncate flex-1">{s.title || s.model}</span>
              <button
                onClick={(e) => { e.stopPropagation(); onDeleteSession(s.id); }}
                className="opacity-0 group-hover:opacity-100 p-0.5 rounded hover:bg-destructive/10 hover:text-destructive transition-all cursor-pointer"
              >
                <Trash2 size={12} />
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default ParameterPanel;
