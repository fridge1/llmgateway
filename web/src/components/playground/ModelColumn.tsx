import { useState, useRef, useEffect } from "react";
import { X, Loader2, Square } from "lucide-react";
import { useSSE } from "@/hooks/use-sse";
import type { ChatCompletionMessage, SSEUsage } from "@/types/api";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

interface ModelColumnProps {
  model: string;
  displayName: string;
  temperature: number;
  keyId: string;
  messages: ChatCompletionMessage[];
  triggerSend: number;
  onRemove: () => void;
}

const ModelColumn = ({
  model,
  displayName,
  temperature,
  keyId,
  messages,
  triggerSend,
  onRemove,
}: ModelColumnProps) => {
  const sse = useSSE();
  const [response, setResponse] = useState("");
  const [usage, setUsage] = useState<SSEUsage | null>(null);
  const [duration, setDuration] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [history, setHistory] = useState<ChatCompletionMessage[]>([]);
  const startTimeRef = useRef(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const lastTriggerRef = useRef(0);
  const userScrolledUpRef = useRef(false);

  const handleScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    userScrolledUpRef.current = el.scrollHeight - el.scrollTop - el.clientHeight > 100;
  };

  useEffect(() => {
    if (scrollRef.current && !userScrolledUpRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [response]);

  useEffect(() => {
    if (triggerSend === 0 || triggerSend === lastTriggerRef.current) return;
    lastTriggerRef.current = triggerSend;

    if (!keyId || messages.length === 0) return;

    const latestUserMsg = messages[messages.length - 1];
    const fullMessages: ChatCompletionMessage[] = [...history, latestUserMsg];

    // Resetting per-send state then kicking off SSE is an imperative action
    // tied to the parent's triggerSend counter, not a state-sync effect — the
    // setState calls here are intentional and safe.
    /* eslint-disable react-hooks/set-state-in-effect */
    setResponse("");
    setUsage(null);
    setDuration(null);
    setError(null);
    /* eslint-enable react-hooks/set-state-in-effect */
    startTimeRef.current = Date.now();

    sse.send({
      keyId,
      model,
      messages: fullMessages,
      temperature,
      onDelta: (fullText) => setResponse(fullText),
      onComplete: (fullText, u) => {
        setResponse(fullText);
        setUsage(u);
        setDuration(Date.now() - startTimeRef.current);
        setHistory([...fullMessages, { role: "assistant", content: fullText }]);
      },
      onError: (err) => {
        setError(err.message);
        setHistory(fullMessages);
      },
    });
  }, [triggerSend]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="flex-1 min-w-0 border-r border-border last:border-r-0 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-border bg-muted/30">
        <div>
          <div className="text-sm font-medium truncate">{displayName || model}</div>
          <div className="text-xs text-muted-foreground">temp: {temperature}</div>
        </div>
        <div className="flex items-center gap-1">
          {sse.streaming && (
            <button
              onClick={() => sse.abort()}
              className="p-1 rounded hover:bg-destructive/10 hover:text-destructive text-muted-foreground transition-colors cursor-pointer"
              title="停止生成"
            >
              <Square size={14} />
            </button>
          )}
          <button
            onClick={onRemove}
            className="p-1 rounded hover:bg-destructive/10 hover:text-destructive text-muted-foreground transition-colors cursor-pointer"
          >
            <X size={14} />
          </button>
        </div>
      </div>

      {/* Response */}
      <div ref={scrollRef} onScroll={handleScroll} className="flex-1 overflow-y-auto p-3">
        {sse.streaming && !response && (
          <div className="flex items-center gap-2 text-muted-foreground text-sm">
            <Loader2 size={14} className="animate-spin" />
            等待响应...
          </div>
        )}
        {response && (
          <div className="prose prose-sm dark:prose-invert max-w-none prose-pre:bg-muted prose-pre:text-foreground prose-code:before:content-none prose-code:after:content-none">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{response}</ReactMarkdown>
          </div>
        )}
        {sse.streaming && response && (
          <span className="inline-block w-1.5 h-4 bg-foreground/60 animate-pulse ml-0.5" />
        )}
        {error && (
          <div className="text-sm text-destructive bg-destructive/10 rounded-lg p-3">
            {error}
          </div>
        )}
      </div>

      {/* Footer metrics */}
      {usage && !sse.streaming && (
        <div className="px-3 py-2 border-t border-border text-xs text-muted-foreground flex items-center gap-3">
          <span>{usage.total_tokens} tokens</span>
          {duration != null && <span>{(duration / 1000).toFixed(1)}s</span>}
        </div>
      )}
    </div>
  );
};

export default ModelColumn;
