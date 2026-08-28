import { Copy, RefreshCw, Loader2 } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

interface MessageBubbleProps {
  role: "user" | "assistant" | "system";
  content: string;
  model?: string;
  tokensUsed?: number;
  cost?: number;
  duration?: number;
  streaming?: boolean;
  onCopy?: () => void;
  onRegenerate?: () => void;
}

const MessageBubble = ({
  role,
  content,
  model,
  tokensUsed,
  cost,
  duration,
  streaming,
  onCopy,
  onRegenerate,
}: MessageBubbleProps) => {
  const isUser = role === "user";

  return (
    <div className={`flex ${isUser ? "justify-end" : "justify-start"} mb-4`}>
      <div
        className={`max-w-[70%] rounded-2xl px-4 py-3 ${
          isUser
            ? "bg-amber-500 text-obsidian-950"
            : "bg-obsidian-900 border border-obsidian-700"
        }`}
      >
        {/* Content */}
        <div className="text-sm break-words">
          {isUser ? (
            <div className="whitespace-pre-wrap">{content}</div>
          ) : streaming && !content ? (
            <div className="flex items-center gap-2 text-obsidian-400">
              <Loader2 size={14} className="animate-spin" />
              <span>思考中...</span>
            </div>
          ) : (
            <div className="md-body">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
            </div>
          )}
        </div>

        {/* Streaming indicator */}
        {streaming && (
          <span className="inline-block w-1.5 h-4 bg-current opacity-60 animate-pulse ml-0.5" />
        )}

        {/* Metadata (assistant only) */}
        {!isUser && !streaming && content && (
          <div className="flex items-center gap-3 mt-2 pt-2 border-t border-obsidian-700/50">
            <span className="text-xs text-obsidian-400">
              {model && <>{model} · </>}
              {tokensUsed != null && <>{tokensUsed} tokens · </>}
              {cost != null && <>&yen;{cost.toFixed(4)} · </>}
              {duration != null && <>{(duration / 1000).toFixed(1)}s</>}
            </span>
            <div className="flex items-center gap-1 ml-auto">
              {onCopy && (
                <button
                  onClick={onCopy}
                  className="p-1 rounded hover:bg-obsidian-800 text-obsidian-400 hover:text-obsidian-100 transition-colors cursor-pointer"
                >
                  <Copy size={12} />
                </button>
              )}
              {onRegenerate && (
                <button
                  onClick={onRegenerate}
                  className="p-1 rounded hover:bg-obsidian-800 text-obsidian-400 hover:text-obsidian-100 transition-colors cursor-pointer"
                >
                  <RefreshCw size={12} />
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default MessageBubble;
