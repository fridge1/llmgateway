import { useState, useCallback, useRef, useEffect } from "react";
import type {
  ChatCompletionMessage,
  SSEChunk,
  SSEUsage,
} from "@/types/api";

export interface UseSSEOptions {
  keyId: string;
  model: string;
  messages: ChatCompletionMessage[];
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  onDelta?: (text: string) => void;
  onComplete?: (fullText: string, usage: SSEUsage | null) => void;
  onError?: (error: Error) => void;
}

export interface UseSSEReturn {
  send: (opts: UseSSEOptions) => void;
  abort: () => void;
  streaming: boolean;
  error: Error | null;
}

export function useSSE(): UseSSEReturn {
  const [streaming, setStreaming] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const abort = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setStreaming(false);
  }, []);

  const send = useCallback((opts: UseSSEOptions) => {
    abortRef.current?.abort();

    const controller = new AbortController();
    abortRef.current = controller;
    setStreaming(true);
    setError(null);

    let fullText = "";
    let lastUsage: SSEUsage | null = null;

    (async () => {
      try {
        const res = await fetch("/api/playground/completions", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            key_id: opts.keyId,
            model: opts.model,
            messages: opts.messages,
            stream: true,
            temperature: opts.temperature,
            max_tokens: opts.max_tokens,
            top_p: opts.top_p,
          }),
          signal: controller.signal,
        });

        if (!res.ok) {
          const errBody = await res.text();
          let msg = res.statusText;
          try {
            const parsed = JSON.parse(errBody);
            msg = parsed?.error?.message || msg;
          } catch { /* ignore */ }
          throw new Error(msg);
        }

        const reader = res.body!.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        const processLines = (lines: string[]) => {
          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed || !trimmed.startsWith("data: ")) continue;

            const data = trimmed.slice(6);
            if (data === "[DONE]") continue;

            try {
              const chunk: SSEChunk = JSON.parse(data);
              if (chunk.usage) {
                lastUsage = chunk.usage;
              }
              const delta = chunk.choices?.[0]?.delta?.content;
              if (delta) {
                fullText += delta;
                opts.onDelta?.(fullText);
              }
            } catch {
              // Skip unparseable lines
            }
          }
        };

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() || "";
          processLines(lines);
        }

        // Flush decoder for any remaining multi-byte characters
        // and process any data left in the buffer
        buffer += decoder.decode();
        if (buffer.trim()) {
          processLines([buffer]);
        }

        opts.onComplete?.(fullText, lastUsage);
      } catch (err: unknown) {
        if (err instanceof Error && err.name === "AbortError") return;
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        opts.onError?.(e);
      } finally {
        setStreaming(false);
        abortRef.current = null;
      }
    })();
  }, []);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return { send, abort, streaming, error };
}
