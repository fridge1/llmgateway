import { useState, useCallback, useRef, useMemo } from "react";
import PlaygroundLayout from "@/components/playground/PlaygroundLayout";
import ParameterPanel from "@/components/playground/ParameterPanel";
import { isChatModel } from "@/lib/utils";
import ChatArea from "@/components/playground/ChatArea";
import type { DisplayMessage } from "@/components/playground/ChatArea";
import CompareView from "@/components/playground/CompareView";
import { useSSE } from "@/hooks/use-sse";
import {
  useGatewayModels,
  useApiKeys,
  useChatSessions,
  useCreateChatSession,
  useChatMessages,
  useSendChatMessage,
  useDeleteChatSession,
} from "@/hooks/use-api";
import type { ChatCompletionMessage } from "@/types/api";

const PlaygroundPage = () => {
  const [mode, setMode] = useState<"single" | "compare">("single");

  // --- Model & params state ---
  const [selectedModel, setSelectedModel] = useState("");
  const [selectedKeyId, setSelectedKeyId] = useState("");
  const [temperature, setTemperature] = useState(0.7);
  const [maxTokens, setMaxTokens] = useState(4096);
  const [topP, setTopP] = useState(1.0);
  const [systemPrompt, setSystemPrompt] = useState("");

  // --- Session state ---
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [displayMessages, setDisplayMessages] = useState<DisplayMessage[]>([]);

  // --- Data hooks ---
  const { data: allModels = [] } = useGatewayModels();
  const models = useMemo(() => allModels.filter(isChatModel), [allModels]);
  const { data: apiKeys = [] } = useApiKeys();
  const { data: sessions = [] } = useChatSessions();
  const createSession = useCreateChatSession();
  const deleteSession = useDeleteChatSession();
  const { data: sessionMessages } = useChatMessages(activeSessionId || "");
  const sendMessage = useSendChatMessage(activeSessionId || "");

  // --- SSE ---
  const sse = useSSE();
  const startTimeRef = useRef<number>(0);

  // --- Session management ---
  const handleSelectSession = useCallback((id: string) => {
    setActiveSessionId(id);
    setDisplayMessages([]);
  }, []);

  // Convert session messages to display format
  const messagesForDisplay: DisplayMessage[] = sessionMessages
    ? sessionMessages.map((m) => ({
        id: m.id,
        role: m.role as "user" | "assistant",
        content: m.content,
        tokensUsed: m.tokens_used ?? undefined,
        cost: m.cost ?? undefined,
      }))
    : activeSessionId
      ? []
      : displayMessages;

  // --- Send message (Single mode) ---
  const handleSend = useCallback(
    async (content: string) => {
      if (!selectedKeyId || !selectedModel) return;

      let sessionId = activeSessionId;
      if (!sessionId) {
        const session = await createSession.mutateAsync({
          model: selectedModel,
          title: content.slice(0, 50),
        });
        sessionId = session.id;
        setActiveSessionId(sessionId);
      }

      // Save user message
      await sendMessage.mutateAsync({ role: "user", content, session_id: sessionId });

      // Build messages for API
      const apiMessages: ChatCompletionMessage[] = [];
      if (systemPrompt) {
        apiMessages.push({ role: "system", content: systemPrompt });
      }
      const current = sessionMessages || [];
      for (const m of current) {
        apiMessages.push({ role: m.role as "system" | "user" | "assistant", content: m.content });
      }
      apiMessages.push({ role: "user", content });

      // Add streaming placeholder
      const assistantId = `streaming-${Date.now()}`;
      setDisplayMessages((prev) => [
        ...prev,
        { id: `user-${Date.now()}`, role: "user", content },
        { id: assistantId, role: "assistant", content: "", streaming: true },
      ]);

      startTimeRef.current = Date.now();

      const capturedSessionId = sessionId;

      sse.send({
        keyId: selectedKeyId,
        model: selectedModel,
        messages: apiMessages,
        temperature,
        max_tokens: maxTokens,
        top_p: topP,
        onDelta: (fullText) => {
          setDisplayMessages((prev) =>
            prev.map((m) =>
              m.id === assistantId ? { ...m, content: fullText } : m,
            ),
          );
        },
        onComplete: (fullText, usage) => {
          const duration = Date.now() - startTimeRef.current;

          setDisplayMessages((prev) =>
            prev.map((m) =>
              m.id === assistantId
                ? {
                    ...m,
                    content: fullText,
                    streaming: false,
                    model: selectedModel,
                    tokensUsed: usage?.total_tokens,
                    duration,
                  }
                : m,
            ),
          );

          // Persist assistant message
          if (capturedSessionId) {
            sendMessage.mutate({
              role: "assistant",
              content: fullText,
              tokens_used: usage?.total_tokens,
              session_id: capturedSessionId,
            });
          }
        },
        onError: (err) => {
          setDisplayMessages((prev) =>
            prev.map((m) =>
              m.id === assistantId
                ? { ...m, content: `错误: ${err.message}`, streaming: false }
                : m,
            ),
          );
          if (capturedSessionId) {
            sendMessage.mutate({
              role: "assistant",
              content: `[错误] ${err.message}`,
              session_id: capturedSessionId,
            });
          }
        },
      });
    },
    [
      activeSessionId,
      selectedModel,
      selectedKeyId,
      temperature,
      maxTokens,
      topP,
      systemPrompt,
      sessionMessages,
      sse,
      createSession,
      sendMessage,
    ],
  );

  const handleCreateSession = useCallback(async () => {
    if (!selectedModel) return;
    const session = await createSession.mutateAsync({
      model: selectedModel,
      title: "新会话",
    });
    setActiveSessionId(session.id);
    setDisplayMessages([]);
  }, [selectedModel, createSession]);

  const handleDeleteSession = useCallback(
    (id: string) => {
      deleteSession.mutate(id);
      if (activeSessionId === id) {
        setActiveSessionId(null);
        setDisplayMessages([]);
      }
    },
    [activeSessionId, deleteSession],
  );

  return (
    <PlaygroundLayout mode={mode} onModeChange={setMode}>
      {mode === "single" ? (
        <div className="flex h-full">
          <ParameterPanel
            models={models}
            apiKeys={apiKeys}
            selectedModel={selectedModel}
            selectedKeyId={selectedKeyId}
            onModelChange={setSelectedModel}
            onKeyChange={setSelectedKeyId}
            temperature={temperature}
            maxTokens={maxTokens}
            topP={topP}
            systemPrompt={systemPrompt}
            onTemperatureChange={setTemperature}
            onMaxTokensChange={setMaxTokens}
            onTopPChange={setTopP}
            onSystemPromptChange={setSystemPrompt}
            sessions={sessions}
            activeSessionId={activeSessionId}
            onSelectSession={handleSelectSession}
            onCreateSession={handleCreateSession}
            onDeleteSession={handleDeleteSession}
          />
          <ChatArea
            messages={messagesForDisplay}
            streaming={sse.streaming}
            onSend={handleSend}
            onAbort={sse.abort}
            disabled={!selectedModel || !selectedKeyId}
            placeholder={
              !selectedModel
                ? "请先选择模型..."
                : !selectedKeyId
                  ? "请先选择 API 密钥..."
                  : "输入消息..."
            }
          />
        </div>
      ) : (
        <CompareView
          keyId={selectedKeyId}
          selectedKeyId={selectedKeyId}
          onKeyChange={setSelectedKeyId}
        />
      )}
    </PlaygroundLayout>
  );
};

export default PlaygroundPage;
