import { useState, useCallback, useMemo } from "react";
import { Plus, Send } from "lucide-react";
import ModelColumn from "./ModelColumn";
import { useGatewayModels, useApiKeys } from "@/hooks/use-api";
import { isChatModel } from "@/lib/utils";
import type { ChatCompletionMessage, GatewayModel } from "@/lib/types-api";

interface CompareModel {
  id: string;
  model: string;
  displayName: string;
  temperature: number;
}

const MAX_COLUMNS = 4;

interface CompareViewProps {
  keyId: string;
  selectedKeyId: string;
  onKeyChange: (keyId: string) => void;
}

const CompareView = ({ keyId, selectedKeyId, onKeyChange }: CompareViewProps) => {
  const { data: allModels = [] } = useGatewayModels();
  const models = useMemo(() => allModels.filter(isChatModel), [allModels]);
  const { data: apiKeys = [] } = useApiKeys();

  const [columns, setColumns] = useState<CompareModel[]>([]);
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState<ChatCompletionMessage[]>([]);
  const [triggerSend, setTriggerSend] = useState(0);
  const [showAddModel, setShowAddModel] = useState(false);

  const addColumn = useCallback(
    (model: GatewayModel) => {
      if (columns.length >= MAX_COLUMNS) return;
      setColumns((prev) => [
        ...prev,
        {
          id: `${model.name}-${Date.now()}`,
          model: model.name,
          displayName: model.display_name || model.name,
          temperature: 0.7,
        },
      ]);
      setShowAddModel(false);
    },
    [columns.length],
  );

  const removeColumn = useCallback((id: string) => {
    setColumns((prev) => prev.filter((c) => c.id !== id));
  }, []);

  const handleSend = useCallback(() => {
    const text = input.trim();
    if (!text || columns.length === 0) return;

    const newMessages: ChatCompletionMessage[] = [
      ...messages,
      { role: "user", content: text },
    ];
    setMessages(newMessages);
    setInput("");
    setTriggerSend((prev) => prev + 1);
  }, [input, columns, messages]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* API Key selector (top bar) */}
      <div className="flex items-center gap-2 px-4 py-2 border-b border-obsidian-700 bg-obsidian-800/20">
        <label className="text-xs text-obsidian-400">API 密钥:</label>
        <select
          value={selectedKeyId}
          onChange={(e) => onKeyChange(e.target.value)}
          className="h-7 px-2 text-xs border border-obsidian-700 rounded-md bg-obsidian-900 text-obsidian-100 focus:outline-none focus:ring-1 focus:ring-amber-500"
        >
          <option value="">选择密钥...</option>
          {apiKeys.map((k) => (
            <option key={k.id} value={k.id}>
              {k.name} ({k.key_prefix}...)
            </option>
          ))}
        </select>

        <div className="ml-auto">
          {columns.length < MAX_COLUMNS && (
            <div className="relative">
              <button
                onClick={() => setShowAddModel(!showAddModel)}
                className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-amber-400 hover:bg-amber-500/10 rounded-md transition-colors cursor-pointer"
              >
                <Plus size={14} />
                添加模型
              </button>
              {showAddModel && (
                <div className="absolute right-0 top-full mt-1 w-60 bg-obsidian-900 border border-obsidian-700 rounded-lg shadow-card-hover z-50 max-h-64 overflow-y-auto py-1">
                  {models.map((m) => (
                    <button
                      key={m.name}
                      onClick={() => addColumn(m)}
                      className="w-full text-left px-3 py-2 text-sm text-obsidian-200 hover:bg-obsidian-800 transition-colors"
                    >
                      {m.display_name || m.name}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Columns area */}
      <div className="flex-1 flex overflow-hidden">
        {columns.length === 0 ? (
          <div className="flex-1 flex items-center justify-center text-obsidian-400">
            <div className="text-center">
              <p className="text-lg font-medium mb-1">模型对比</p>
              <p className="text-sm mb-4">点击"添加模型"开始对比（最多 {MAX_COLUMNS} 个）</p>
            </div>
          </div>
        ) : (
          columns.map((col) => (
            <ModelColumn
              key={col.id}
              model={col.model}
              displayName={col.displayName}
              temperature={col.temperature}
              keyId={keyId}
              messages={messages}
              triggerSend={triggerSend}
              onRemove={() => removeColumn(col.id)}
            />
          ))
        )}
      </div>

      {/* Shared input */}
      {columns.length > 0 && (
        <div className="border-t border-obsidian-700 px-6 py-3">
          <div className="flex items-end gap-2">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={
                !keyId ? "请先选择 API 密钥..." : "输入消息，同时发送给所有模型..."
              }
              rows={1}
              disabled={!keyId}
              className="flex-1 resize-none px-3 py-2 text-sm border border-obsidian-700 rounded-xl bg-obsidian-900 text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:ring-1 focus:ring-amber-500 disabled:opacity-50 max-h-32"
              style={{ minHeight: "40px" }}
            />
            <button
              onClick={handleSend}
              disabled={!input.trim() || !keyId || columns.length === 0}
              className="shrink-0 w-9 h-9 flex items-center justify-center rounded-xl bg-amber-500 text-obsidian-950 hover:bg-amber-400 transition-colors disabled:opacity-50 cursor-pointer"
            >
              <Send size={16} />
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default CompareView;
