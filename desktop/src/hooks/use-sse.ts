import { useState, useCallback, useRef, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import type { ChatCompletionMessage, SSEUsage } from "@/lib/types-api";

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

/**
 * Desktop SSE hook. Matches the web `useSSE` contract, but instead of a browser
 * `fetch` stream it drives the Rust `playground_stream` command, which keeps the
 * JWT in the OS keyring and relays chunks back as Tauri events. The frontend never
 * sees the token.
 */
export function useSSE(): UseSSEReturn {
  const [streaming, setStreaming] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const requestIdRef = useRef<string | null>(null);
  const unlistenersRef = useRef<UnlistenFn[]>([]);

  const cleanup = useCallback(() => {
    for (const un of unlistenersRef.current) un();
    unlistenersRef.current = [];
  }, []);

  const abort = useCallback(() => {
    const id = requestIdRef.current;
    if (id) {
      invoke("playground_stream_abort", { requestId: id }).catch(() => {});
    }
    requestIdRef.current = null;
    cleanup();
    setStreaming(false);
  }, [cleanup]);

  const send = useCallback(
    (opts: UseSSEOptions) => {
      // Tear down any prior stream first.
      abort();

      const requestId = `pg-${Date.now()}-${Math.random().toString(36).slice(2)}`;
      requestIdRef.current = requestId;
      setStreaming(true);
      setError(null);

      let fullText = "";

      (async () => {
        try {
          const unDelta = await listen<{ text: string }>(
            `playground-sse-delta-${requestId}`,
            (e) => {
              fullText += e.payload.text;
              opts.onDelta?.(fullText);
            },
          );
          const unComplete = await listen<{ usage: SSEUsage | null }>(
            `playground-sse-complete-${requestId}`,
            (e) => {
              cleanup();
              if (requestIdRef.current === requestId) requestIdRef.current = null;
              setStreaming(false);
              opts.onComplete?.(fullText, e.payload.usage ?? null);
            },
          );
          const unError = await listen<{ message: string }>(
            `playground-sse-error-${requestId}`,
            (e) => {
              cleanup();
              if (requestIdRef.current === requestId) requestIdRef.current = null;
              setStreaming(false);
              const err = new Error(e.payload.message);
              setError(err);
              opts.onError?.(err);
            },
          );
          unlistenersRef.current = [unDelta, unComplete, unError];

          await invoke("playground_stream", {
            requestId,
            body: {
              key_id: opts.keyId,
              model: opts.model,
              messages: opts.messages,
              stream: true,
              temperature: opts.temperature,
              max_tokens: opts.max_tokens,
              top_p: opts.top_p,
            },
          });
        } catch (err: unknown) {
          // invoke rejects after the error event already fired; avoid double-reporting.
          if (requestIdRef.current === requestId) {
            cleanup();
            requestIdRef.current = null;
            setStreaming(false);
            const e = err instanceof Error ? err : new Error(String(err));
            setError(e);
            opts.onError?.(e);
          }
        }
      })();
    },
    [abort, cleanup],
  );

  useEffect(() => {
    return () => {
      abort();
    };
  }, [abort]);

  return { send, abort, streaming, error };
}
