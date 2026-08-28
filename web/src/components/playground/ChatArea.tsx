import { useState, useRef, useEffect } from "react";
import { Send, Square } from "lucide-react";
import MessageBubble from "./MessageBubble";

export interface DisplayMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  model?: string;
  tokensUsed?: number;
  cost?: number;
  duration?: number;
  streaming?: boolean;
}

interface ChatAreaProps {
  messages: DisplayMessage[];
  streaming: boolean;
  onSend: (content: string) => void;
  onAbort: () => void;
  onRegenerate?: (messageId: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

const ChatArea = ({
  messages,
  streaming,
  onSend,
  onAbort,
  onRegenerate,
  disabled,
  placeholder = "输入消息...",
}: ChatAreaProps) => {
  const [input, setInput] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);
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
  }, [messages]);

  const handleSend = () => {
    const text = input.trim();
    if (!text || disabled) return;
    setInput("");
    onSend(text);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex-1 flex flex-col h-full">
      {/* Message list */}
      <div ref={scrollRef} onScroll={handleScroll} className="flex-1 overflow-y-auto px-6 py-4">
        {messages.length === 0 && (
          <div className="flex items-center justify-center h-full">
            <div className="text-center text-muted-foreground">
              <p className="text-lg font-medium mb-1">开始对话</p>
              <p className="text-sm">选择模型并输入消息开始体验</p>
            </div>
          </div>
        )}
        {messages.map((msg) => (
          <MessageBubble
            key={msg.id}
            role={msg.role}
            content={msg.content}
            model={msg.model}
            tokensUsed={msg.tokensUsed}
            cost={msg.cost}
            duration={msg.duration}
            streaming={msg.streaming}
            onCopy={() => navigator.clipboard.writeText(msg.content)}
            onRegenerate={
              msg.role === "assistant" && !msg.streaming && onRegenerate
                ? () => onRegenerate(msg.id)
                : undefined
            }
          />
        ))}
      </div>

      {/* Input area */}
      <div className="border-t border-border px-6 py-3">
        <div className="flex items-end gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            rows={1}
            disabled={disabled}
            className="flex-1 resize-none px-3 py-2 text-sm border border-border rounded-xl bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50 max-h-32"
            style={{ minHeight: "40px" }}
          />
          {streaming ? (
            <button
              onClick={onAbort}
              className="shrink-0 w-9 h-9 flex items-center justify-center rounded-xl bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors cursor-pointer"
            >
              <Square size={16} />
            </button>
          ) : (
            <button
              onClick={handleSend}
              disabled={!input.trim() || disabled}
              className="shrink-0 w-9 h-9 flex items-center justify-center rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 cursor-pointer"
            >
              <Send size={16} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ChatArea;
