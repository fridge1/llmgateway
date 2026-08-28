import { useState, useCallback, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft, Languages, ArrowRightLeft, Copy, Check, Loader2 } from "lucide-react";
import { useSSE } from "@/hooks/use-sse";
import { useGatewayModels, useApiKeys } from "@/hooks/use-api";
import { isChatModel } from "@/lib/utils";
import { languages, targetLanguages } from "@/config/languages";
import type { SSEUsage } from "@/types/api";

const TranslatePage = () => {
  const navigate = useNavigate();

  const [sourceLang, setSourceLang] = useState("auto");
  const [targetLang, setTargetLang] = useState("en");
  const [sourceText, setSourceText] = useState("");
  const [translatedText, setTranslatedText] = useState("");
  const [selectedModel, setSelectedModel] = useState("");
  const [selectedKeyId, setSelectedKeyId] = useState("");
  const [usage, setUsage] = useState<SSEUsage | null>(null);
  const [duration, setDuration] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const startTimeRef = useRef<number>(0);

  const { data: allModels = [] } = useGatewayModels();
  const models = useMemo(() => allModels.filter(isChatModel), [allModels]);
  const { data: apiKeys = [] } = useApiKeys();

  const sse = useSSE();

  const buildSystemPrompt = useCallback((src: string, tgt: string) => {
    const from =
      src === "auto"
        ? "the source language (auto-detect it)"
        : languages.find((l) => l.code === src)?.englishName;
    const to = languages.find((l) => l.code === tgt)?.englishName;
    return `You are a professional translator. Translate the following text from ${from} to ${to}. Output ONLY the translated text. Do not add explanations, notes, or any extra content.`;
  }, []);

  const handleTranslate = useCallback(() => {
    if (!selectedKeyId || !selectedModel || !sourceText.trim()) return;

    setTranslatedText("");
    setUsage(null);
    setDuration(null);
    setError(null);
    startTimeRef.current = Date.now();

    sse.send({
      keyId: selectedKeyId,
      model: selectedModel,
      messages: [
        { role: "system", content: buildSystemPrompt(sourceLang, targetLang) },
        { role: "user", content: sourceText },
      ],
      temperature: 0.3,
      max_tokens: 4096,
      onDelta: (fullText) => setTranslatedText(fullText),
      onComplete: (fullText, u) => {
        setTranslatedText(fullText);
        setUsage(u);
        setDuration(Date.now() - startTimeRef.current);
      },
      onError: (err) => setError(err.message),
    });
  }, [selectedKeyId, selectedModel, sourceText, sourceLang, targetLang, sse, buildSystemPrompt]);

  const handleSwapLanguages = useCallback(() => {
    if (sourceLang === "auto") return;
    setSourceLang(targetLang);
    setTargetLang(sourceLang);
    setSourceText(translatedText);
    setTranslatedText(sourceText);
    setUsage(null);
    setDuration(null);
  }, [sourceLang, targetLang, sourceText, translatedText]);

  const handleCopy = useCallback(async () => {
    if (!translatedText) return;
    await navigator.clipboard.writeText(translatedText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [translatedText]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        handleTranslate();
      }
    },
    [handleTranslate],
  );

  const canTranslate = selectedKeyId && selectedModel && sourceText.trim() && !sse.streaming;

  return (
    <div className="h-screen flex flex-col bg-background">
      {/* Header */}
      <header className="h-12 flex items-center justify-between px-4 border-b border-border bg-card shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate("/tools")}
            className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            <ArrowLeft size={16} />
            返回
          </button>
          <div className="w-px h-4 bg-border" />
          <div className="flex items-center gap-1.5">
            <Languages size={14} className="text-emerald-500" />
            <span className="font-semibold text-sm">文本翻译</span>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={selectedKeyId}
            onChange={(e) => setSelectedKeyId(e.target.value)}
            className="h-7 px-2 text-xs border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">选择密钥...</option>
            {apiKeys.map((k) => (
              <option key={k.id} value={k.id}>
                {k.name} ({k.key_prefix}...)
              </option>
            ))}
          </select>
          <select
            value={selectedModel}
            onChange={(e) => setSelectedModel(e.target.value)}
            className="h-7 px-2 text-xs border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="">选择模型...</option>
            {models.map((m) => (
              <option key={m.name} value={m.name}>
                {m.display_name || m.name}
              </option>
            ))}
          </select>
        </div>
      </header>

      {/* Language bar */}
      <div className="flex items-center justify-center gap-4 px-4 py-2 border-b border-border bg-card/50 shrink-0">
        <div className="flex-1 flex justify-end">
          <select
            value={sourceLang}
            onChange={(e) => setSourceLang(e.target.value)}
            className="h-8 px-3 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {languages.map((l) => (
              <option key={l.code} value={l.code}>
                {l.name}
              </option>
            ))}
          </select>
        </div>
        <button
          onClick={handleSwapLanguages}
          disabled={sourceLang === "auto"}
          className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed"
          title="交换语言"
        >
          <ArrowRightLeft size={18} />
        </button>
        <div className="flex-1">
          <select
            value={targetLang}
            onChange={(e) => setTargetLang(e.target.value)}
            className="h-8 px-3 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            {targetLanguages.map((l) => (
              <option key={l.code} value={l.code}>
                {l.name}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Main content: side-by-side panels */}
      <div className="flex-1 flex overflow-hidden">
        {/* Source panel */}
        <div className="flex-1 flex flex-col border-r border-border">
          <textarea
            value={sourceText}
            onChange={(e) => setSourceText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="输入要翻译的文本..."
            className="flex-1 p-4 text-sm bg-background text-foreground resize-none focus:outline-none"
          />
          <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-card/50">
            <span className="text-xs text-muted-foreground">
              {sourceText.length} 字符
            </span>
            <button
              onClick={handleTranslate}
              disabled={!canTranslate}
              className="flex items-center gap-1.5 px-4 py-1.5 text-sm font-medium text-white bg-emerald-500 hover:bg-emerald-600 rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {sse.streaming ? (
                <>
                  <Loader2 size={14} className="animate-spin" />
                  翻译中...
                </>
              ) : (
                "翻译"
              )}
            </button>
          </div>
        </div>

        {/* Target panel */}
        <div className="flex-1 flex flex-col">
          <div className="flex-1 p-4 text-sm bg-background text-foreground overflow-y-auto whitespace-pre-wrap">
            {error ? (
              <span className="text-destructive">错误: {error}</span>
            ) : translatedText ? (
              translatedText
            ) : (
              <span className="text-muted-foreground">翻译结果将显示在这里...</span>
            )}
          </div>
          <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-card/50">
            <span className="text-xs text-muted-foreground">
              {usage && (
                <>
                  {usage.total_tokens} tokens
                  {duration !== null && <> · {(duration / 1000).toFixed(1)}s</>}
                </>
              )}
            </span>
            <button
              onClick={handleCopy}
              disabled={!translatedText || sse.streaming}
              className="flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground rounded-md hover:bg-muted transition-colors cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed"
            >
              {copied ? <Check size={13} /> : <Copy size={13} />}
              {copied ? "已复制" : "复制"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TranslatePage;
