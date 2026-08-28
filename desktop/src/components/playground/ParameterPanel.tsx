import { Plus, Trash2, MessageSquare } from "lucide-react";
import type { ChatSession, GatewayModel, APIKey } from "@/lib/types-api";

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

const selectClass =
  "w-full h-8 px-2 text-sm border border-obsidian-700 rounded-lg bg-obsidian-900 text-obsidian-100 focus:outline-none focus:ring-1 focus:ring-amber-500";

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
    <div className="w-72 h-full border-r border-obsidian-700 bg-obsidian-900 flex flex-col overflow-hidden">
      {/* Parameters section */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* API Key selector */}
        <div>
          <label className="text-xs font-medium text-obsidian-400 mb-1.5 block">API 密钥</label>
          <select
            value={selectedKeyId}
            onChange={(e) => onKeyChange(e.target.value)}
            className={selectClass}
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
          <label className="text-xs font-medium text-obsidian-400 mb-1.5 block">模型</label>
          <select
            value={selectedModel}
            onChange={(e) => onModelChange(e.target.value)}
            className={selectClass}
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
            <label className="text-xs font-medium text-obsidian-400">Temperature</label>
            <span className="text-xs text-obsidian-400 tabular-nums">{temperature.toFixed(1)}</span>
          </div>
          <input
            type="range"
            min={0}
            max={2}
            step={0.1}
            value={temperature}
            onChange={(e) => onTemperatureChange(parseFloat(e.target.value))}
            className="w-full accent-amber-500"
          />
        </div>

        {/* Max Tokens */}
        <div>
          <label className="text-xs font-medium text-obsidian-400 mb-1.5 block">Max Tokens</label>
          <input
            type="number"
            min={1}
            max={128000}
            value={maxTokens}
            onChange={(e) => onMaxTokensChange(parseInt(e.target.value) || 4096)}
            className={selectClass}
          />
        </div>

        {/* Top P */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-xs font-medium text-obsidian-400">Top P</label>
            <span className="text-xs text-obsidian-400 tabular-nums">{topP.toFixed(2)}</span>
          </div>
          <input
            type="range"
            min={0}
            max={1}
            step={0.01}
            value={topP}
            onChange={(e) => onTopPChange(parseFloat(e.target.value))}
            className="w-full accent-amber-500"
          />
        </div>

        {/* System Prompt */}
        <div>
          <label className="text-xs font-medium text-obsidian-400 mb-1.5 block">System Prompt</label>
          <textarea
            value={systemPrompt}
            onChange={(e) => onSystemPromptChange(e.target.value)}
            placeholder="You are a helpful assistant..."
            rows={3}
            className="w-full px-2 py-1.5 text-sm border border-obsidian-700 rounded-lg bg-obsidian-900 text-obsidian-100 placeholder:text-obsidian-500 resize-none focus:outline-none focus:ring-1 focus:ring-amber-500"
          />
        </div>
      </div>

      {/* Sessions section */}
      <div className="border-t border-obsidian-700">
        <div className="flex items-center justify-between px-4 py-2">
          <span className="text-xs font-medium text-obsidian-400">会话列表</span>
          <button
            onClick={onCreateSession}
            className="p-1 rounded hover:bg-obsidian-800 text-obsidian-400 hover:text-obsidian-100 transition-colors cursor-pointer"
          >
            <Plus size={14} />
          </button>
        </div>
        <div className="max-h-48 overflow-y-auto px-2 pb-2">
          {sessions.length === 0 && (
            <div className="text-xs text-obsidian-400 text-center py-4">暂无会话</div>
          )}
          {sessions.map((s) => (
            <div
              key={s.id}
              onClick={() => onSelectSession(s.id)}
              className={`flex items-center gap-2 px-2 py-1.5 rounded-lg text-sm cursor-pointer group transition-colors ${
                activeSessionId === s.id
                  ? "bg-obsidian-800 text-obsidian-50"
                  : "text-obsidian-400 hover:bg-obsidian-800 hover:text-obsidian-100"
              }`}
            >
              <MessageSquare size={13} className="shrink-0" />
              <span className="truncate flex-1">{s.title || s.model}</span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteSession(s.id);
                }}
                className="opacity-0 group-hover:opacity-100 p-0.5 rounded hover:bg-red-500/10 hover:text-red-400 transition-all cursor-pointer"
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
