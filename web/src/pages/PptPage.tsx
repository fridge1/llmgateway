import { useState, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft, Presentation, Download, RefreshCw, Loader2, CheckCircle2, FileText, Pencil, Upload, Trash2, Home } from "lucide-react";
import { useGatewayModels } from "@/hooks/use-api";
import { isChatModel } from "@/lib/utils";
import { pptThemes, pptLanguages, slideCountOptions, getTheme } from "@/config/ppt-themes";
import { exportPptx } from "@/lib/ppt-export";
import SlidePreview from "@/components/ppt/SlidePreview";
import SlideEditor from "@/components/ppt/SlideEditor";
import { useSubmitPptTask, usePptTask, useConfirmOutline, useSavePresentation, usePptTasks, useDeletePptTask } from "@/hooks/use-ppt-tasks";
import type { PresentationData, SlideData, StoryArc, PptTask } from "@/types/ppt";
import { SidebarProvider, Sidebar, SidebarContent, SidebarHeader, SidebarMenu, SidebarMenuItem, SidebarMenuButton, SidebarMenuAction, SidebarTrigger, SidebarInset } from "@/components/ui/sidebar";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";

type Phase = "config" | "generating" | "outline" | "preview";

const audienceOptions = [
  { value: "general", label: "通用" },
  { value: "executive", label: "高管" },
  { value: "technical", label: "技术人员" },
  { value: "investor", label: "投资人" },
  { value: "student", label: "学生" },
];

const toneOptions = [
  { value: "professional", label: "专业" },
  { value: "casual", label: "轻松" },
  { value: "confident", label: "自信" },
  { value: "academic", label: "学术" },
];

const purposeOptions = [
  { value: "inform", label: "信息传达" },
  { value: "persuade", label: "说服" },
  { value: "educate", label: "教育培训" },
  { value: "pitch", label: "商业提案" },
];

const PptPage = () => {
  const navigate = useNavigate();

  const [phase, setPhase] = useState<Phase>("config");
  const [topic, setTopic] = useState("");
  const [slideCount, setSlideCount] = useState(8);
  const [themeId, setThemeId] = useState("clinical-teal");
  const [language, setLanguage] = useState("zh");
  const [audience, setAudience] = useState("general");
  const [tone, setTone] = useState("professional");
  const [purpose, setPurpose] = useState("inform");

  const [presentation, setPresentation] = useState<PresentationData | null>(null);
  const [currentSlide, setCurrentSlide] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState("");
  const [exporting, setExporting] = useState(false);
  const [activeTaskId, setActiveTaskId] = useState<number | null>(null);
  const [outlineOnly, setOutlineOnly] = useState(false);
  const [generateImages, setGenerateImages] = useState(true);
  const [storyArc, setStoryArc] = useState<StoryArc | null>(null);
  const [contextText, setContextText] = useState("");
  const [editingSlideIndex, setEditingSlideIndex] = useState<number | null>(null);
  const [lastSavedTaskId, setLastSavedTaskId] = useState<number | null>(null);
  const [completedTaskId, setCompletedTaskId] = useState<number | null>(null);

  // History sidebar states
  const [loadedTaskId, setLoadedTaskId] = useState<number | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [taskToDelete, setTaskToDelete] = useState<number | null>(null);

  const { data: allModels = [] } = useGatewayModels();
  const models = useMemo(() => allModels.filter(isChatModel), [allModels]);
  const submitTask = useSubmitPptTask();
  const confirmOutline = useConfirmOutline();
  const savePresentation = useSavePresentation();
  const { data: historyTasks = [] } = usePptTasks(20, 0);
  const deleteMutation = useDeletePptTask();

  const selectedTheme = useMemo(() => getTheme(themeId), [themeId]);

  // Poll active task
  const { data: activeTask } = usePptTask(activeTaskId);

  // React to task status changes
  if (activeTask && activeTaskId !== null) {
    if (activeTask.status === "completed" && phase === "generating") {
      const pres = activeTask.presentation_json;
      if (pres && pres.slides?.length) {
        setPresentation(pres);
        setPhase("preview");
        setCompletedTaskId(activeTaskId);
        setActiveTaskId(null);
      } else {
        setError("生成内容解析失败");
        setPhase("config");
        setActiveTaskId(null);
      }
    } else if (activeTask.status === "outline_ready" && phase === "generating") {
      if (activeTask.story_arc) {
        setStoryArc(activeTask.story_arc);
        setPhase("outline");
      }
    } else if (activeTask.status === "failed" && phase === "generating") {
      setError(activeTask.error_message || "生成失败，请重试");
      setPhase("config");
      setActiveTaskId(null);
    }
  }

  const phaseLabel = activeTask?.phase
    ? { brief_analyst: "分析需求...", content_strategist: "构建叙事...", outline_review: "等待确认大纲...", info_architect: "设计架构...", visual_designer: "AI 配图规划...", image_generation: "生成图片..." }[activeTask.phase] || "处理中..."
    : "排队中...";

  const handleGenerate = useCallback(async () => {
    if (!selectedModel || !topic.trim()) return;

    setPhase("generating");
    setPresentation(null);
    setCurrentSlide(0);
    setError(null);

    try {
      const result = await submitTask.mutateAsync({
        model: selectedModel,
        topic: topic.trim(),
        slide_count: slideCount,
        language,
        theme: themeId,
        audience,
        tone,
        purpose,
        outline_only: outlineOnly,
        generate_images: generateImages,
        context_text: contextText.trim() || undefined,
      });
      setActiveTaskId(result.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "提交失败");
      setPhase("config");
    }
  }, [selectedModel, topic, slideCount, language, themeId, audience, tone, purpose, outlineOnly, generateImages, contextText, submitTask]);

  const handleExport = useCallback(async () => {
    if (!presentation) return;
    setExporting(true);
    try {
      await exportPptx(presentation, selectedTheme);
    } catch (err) {
      setError(err instanceof Error ? err.message : "导出失败");
    } finally {
      setExporting(false);
    }
  }, [presentation, selectedTheme]);

  const handleConfirmOutline = useCallback(async () => {
    if (activeTaskId === null) return;
    try {
      await confirmOutline.mutateAsync(activeTaskId);
      setPhase("generating");
      setStoryArc(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "确认大纲失败");
    }
  }, [activeTaskId, confirmOutline]);

  const handleSlideEdit = useCallback((updatedSlide: SlideData) => {
    if (!presentation || editingSlideIndex === null) return;
    const updatedSlides = [...presentation.slides];
    updatedSlides[editingSlideIndex] = updatedSlide;
    const updatedPres = { ...presentation, slides: updatedSlides };
    setPresentation(updatedPres);
    setEditingSlideIndex(null);

    // Persist to backend if we have a task ID
    if (completedTaskId !== null) {
      savePresentation.mutate({ taskId: completedTaskId, presentation: updatedPres });
      setLastSavedTaskId(completedTaskId);
    }
  }, [presentation, editingSlideIndex, completedTaskId, savePresentation]);

  const handleReset = useCallback(() => {
    setPhase("config");
    setPresentation(null);
    setStoryArc(null);
    setError(null);
    setCurrentSlide(0);
    setActiveTaskId(null);
    setCompletedTaskId(null);
    setEditingSlideIndex(null);
    setLastSavedTaskId(null);
    setLoadedTaskId(null);
  }, []);

  const handleLoadHistory = useCallback((task: PptTask) => {
    if (task.presentation_json) {
      setLoadedTaskId(task.id);
      setPresentation(task.presentation_json);
      setCompletedTaskId(task.id);
      setPhase("preview");
      setCurrentSlide(0);
      // Fill config fields for reference
      setTopic(task.topic);
      setSlideCount(task.slide_count);
      setThemeId(task.theme);
      setLanguage(task.language);
      setAudience(task.audience);
      setTone(task.tone);
      setPurpose(task.purpose);
      setSelectedModel(task.model);
    } else if (task.story_arc && task.status === "outline_ready") {
      setLoadedTaskId(task.id);
      setStoryArc(task.story_arc);
      setActiveTaskId(task.id);
      setPhase("outline");
      setTopic(task.topic);
      setSlideCount(task.slide_count);
      setThemeId(task.theme);
      setLanguage(task.language);
      setAudience(task.audience);
      setTone(task.tone);
      setPurpose(task.purpose);
      setSelectedModel(task.model);
    }
  }, []);

  const handleBackToNew = useCallback(() => {
    setLoadedTaskId(null);
    handleReset();
  }, [handleReset]);

  const handleDeleteClick = useCallback((taskId: number) => {
    setTaskToDelete(taskId);
    setDeleteDialogOpen(true);
  }, []);

  const handleConfirmDelete = useCallback(() => {
    if (taskToDelete) {
      deleteMutation.mutate(taskToDelete, {
        onSuccess: () => {
          if (taskToDelete === loadedTaskId) {
            handleBackToNew();
          }
          setDeleteDialogOpen(false);
          setTaskToDelete(null);
        },
      });
    }
  }, [taskToDelete, loadedTaskId, deleteMutation, handleBackToNew]);

  const getStatusBadge = (status: string) => {
    const badges = {
      pending: { label: "排队中", className: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300" },
      processing: { label: "生成中", className: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300" },
      completed: { label: "已完成", className: "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300" },
      failed: { label: "失败", className: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300" },
      outline_ready: { label: "待确认", className: "bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300" },
    };
    return badges[status as keyof typeof badges] || badges.pending;
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "刚刚";
    if (diffMins < 60) return `${diffMins}分钟前`;
    if (diffHours < 24) return `${diffHours}小时前`;
    if (diffDays === 1) return "昨天";
    if (diffDays < 7) return `${diffDays}天前`;
    return date.toLocaleDateString("zh-CN", { month: "numeric", day: "numeric" });
  };

  const canGenerate = selectedModel && topic.trim();

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      handleGenerate();
    }
  };

  return (
    <SidebarProvider>
      <Sidebar side="left" collapsible="offcanvas">
        <SidebarHeader className="border-b px-4 py-3">
          <div className="flex items-center justify-between">
            <SidebarTrigger />
            <h3 className="font-semibold text-sm ml-2">历史记录</h3>
          </div>
        </SidebarHeader>
        <SidebarContent>
          {historyTasks.length === 0 ? (
            <div className="p-4 text-center text-sm text-muted-foreground">
              暂无历史记录
            </div>
          ) : (
            <SidebarMenu>
              {historyTasks.map((task) => {
                const badge = getStatusBadge(task.status);
                return (
                  <SidebarMenuItem key={task.id}>
                    <SidebarMenuButton
                      onClick={() => handleLoadHistory(task)}
                      className="flex items-start gap-2 p-2"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className={`text-xs px-1.5 py-0.5 rounded ${badge.className}`}>
                            {badge.label}
                          </span>
                        </div>
                        <div className="truncate font-medium text-sm">{task.topic}</div>
                        <div className="text-xs text-muted-foreground mt-0.5">
                          {formatDate(task.created_at)} · {task.slide_count} 页
                        </div>
                      </div>
                    </SidebarMenuButton>
                    <SidebarMenuAction onClick={() => handleDeleteClick(task.id)}>
                      <Trash2 className="w-4 h-4" />
                    </SidebarMenuAction>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          )}
        </SidebarContent>
      </Sidebar>

      <SidebarInset>
        <div className="h-screen flex flex-col bg-background">
          {/* Header */}
          <header className="h-12 flex items-center justify-between px-4 border-b border-border bg-card shrink-0">
            <div className="flex items-center gap-3">
              <SidebarTrigger />
              <button
                onClick={() => navigate("/tools")}
                className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
              >
                <ArrowLeft size={16} />
                返回
              </button>
              <div className="w-px h-4 bg-border" />
              <div className="flex items-center gap-1.5">
                <Presentation size={14} className="text-blue-500" />
                <span className="font-semibold text-sm">PPT 生成</span>
              </div>
              {loadedTaskId && (
                <>
                  <div className="w-px h-4 bg-border" />
                  <button
                    onClick={handleBackToNew}
                    className="flex items-center gap-1.5 text-xs px-2 py-1 rounded-md bg-primary/10 text-primary hover:bg-primary/20 transition-colors cursor-pointer"
                  >
                    <Home size={12} />
                    返回新建
                  </button>
                </>
              )}
            </div>
            <div className="flex items-center gap-3">
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

          {/* Main content */}
      <div className="flex-1 overflow-y-auto">
        {phase === "config" && (
          <div className="flex items-center justify-center h-full p-8">
            <div className="w-full max-w-lg space-y-5">
              <div className="text-center mb-6">
                <Presentation size={36} className="mx-auto text-blue-500 mb-2" />
                <h2 className="text-lg font-semibold">AI PPT 生成</h2>
                <p className="text-sm text-muted-foreground mt-1">输入主题，多智能体协作为你生成专业演示文稿</p>
              </div>

              {error && (
                <div className="p-3 rounded-lg bg-destructive/10 text-destructive text-sm">{error}</div>
              )}

              <div>
                <label className="block text-sm font-medium mb-1.5">演示主题</label>
                <textarea
                  value={topic}
                  onChange={(e) => setTopic(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="例如：人工智能在医疗领域的应用与前景"
                  rows={3}
                  className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1.5">页数</label>
                  <select value={slideCount} onChange={(e) => setSlideCount(Number(e.target.value))}
                    className="w-full h-9 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary">
                    {slideCountOptions.map((n) => <option key={n} value={n}>{n} 页</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">风格</label>
                  <select value={themeId} onChange={(e) => setThemeId(e.target.value)}
                    className="w-full h-9 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary">
                    {pptThemes.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">语言</label>
                  <select value={language} onChange={(e) => setLanguage(e.target.value)}
                    className="w-full h-9 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary">
                    {pptLanguages.map((l) => <option key={l.code} value={l.code}>{l.name}</option>)}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1.5">受众</label>
                  <select value={audience} onChange={(e) => setAudience(e.target.value)}
                    className="w-full h-9 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary">
                    {audienceOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">语调</label>
                  <select value={tone} onChange={(e) => setTone(e.target.value)}
                    className="w-full h-9 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary">
                    {toneOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1.5">目的</label>
                  <select value={purpose} onChange={(e) => setPurpose(e.target.value)}
                    className="w-full h-9 px-2 text-sm border border-border rounded-lg bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary">
                    {purposeOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                  </select>
                </div>
              </div>

              <label className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={outlineOnly}
                  onChange={(e) => setOutlineOnly(e.target.checked)}
                  className="w-4 h-4 rounded border-border text-blue-500 focus:ring-blue-500"
                />
                <span className="text-sm">预览大纲后再生成</span>
                <span className="text-xs text-muted-foreground">（生成大纲后可审阅确认）</span>
              </label>

              <label className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={generateImages}
                  onChange={(e) => setGenerateImages(e.target.checked)}
                  className="w-4 h-4 rounded border-border text-blue-500 focus:ring-blue-500"
                />
                <span className="text-sm">AI 配图</span>
                <span className="text-xs text-muted-foreground">（自动为适合的页面生成配图，会增加费用）</span>
              </label>

              <div>
                <label className="block text-sm font-medium mb-1.5">参考资料（可选）</label>
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <label className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-muted-foreground bg-muted hover:bg-muted/80 rounded-lg transition-colors cursor-pointer">
                      <Upload size={12} />
                      上传文件
                      <input
                        type="file"
                        accept=".txt,.md,.csv"
                        className="hidden"
                        onChange={(e) => {
                          const file = e.target.files?.[0];
                          if (!file) return;
                          if (file.size > 100 * 1024) {
                            setError("文件大小不能超过 100KB");
                            return;
                          }
                          const reader = new FileReader();
                          reader.onload = () => {
                            setContextText(reader.result as string);
                          };
                          reader.readAsText(file);
                          e.target.value = "";
                        }}
                      />
                    </label>
                    {contextText && (
                      <span className="text-xs text-muted-foreground">
                        已加载 {contextText.length} 字符
                        <button
                          onClick={() => setContextText("")}
                          className="ml-1 text-destructive hover:underline cursor-pointer"
                        >
                          清除
                        </button>
                      </span>
                    )}
                  </div>
                  <textarea
                    value={contextText}
                    onChange={(e) => setContextText(e.target.value)}
                    placeholder="粘贴参考文本，或上传 .txt / .md / .csv 文件。AI 将基于此内容生成 PPT。"
                    rows={3}
                    className="w-full px-3 py-2 text-xs border border-border rounded-lg bg-background text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>

              <button
                onClick={handleGenerate}
                disabled={!canGenerate || submitTask.isPending}
                className="w-full flex items-center justify-center gap-2 h-10 text-sm font-medium text-white bg-blue-500 hover:bg-blue-600 rounded-lg transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {submitTask.isPending ? (
                  <><Loader2 size={16} className="animate-spin" />提交中...</>
                ) : (
                  "生成 PPT"
                )}
              </button>

              <p className="text-xs text-center text-muted-foreground">Ctrl + Enter 快速生成</p>
            </div>
          </div>
        )}

        {phase === "generating" && (
          <div className="flex flex-col items-center justify-center h-full p-8 gap-6">
            <Loader2 size={40} className="animate-spin text-blue-500" />
            <div className="text-center">
              <p className="text-sm font-medium">{phaseLabel}</p>
              <p className="text-xs text-muted-foreground mt-1">多智能体协作生成中，请稍候</p>
            </div>
            <div className="flex gap-2">
              {["brief_analyst", "content_strategist", "info_architect", "visual_designer"].map((p, i) => {
                const labels = ["需求分析", "内容策略", "信息架构", "AI 配图"];
                const phases = ["brief_analyst", "content_strategist", "info_architect", "visual_designer", "image_generation", "completed"];
                const currentIdx = phases.indexOf(activeTask?.phase || "");
                const isActive = activeTask?.phase === p || (p === "visual_designer" && activeTask?.phase === "image_generation");
                const isDone = currentIdx > i;
                return (
                  <div key={p} className={`px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${
                    isActive ? "bg-blue-500 text-white" : isDone ? "bg-green-100 text-green-700" : "bg-muted text-muted-foreground"
                  }`}>
                    {labels[i]}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {phase === "outline" && storyArc && (
          <div className="flex items-center justify-center h-full p-8">
            <div className="w-full max-w-2xl space-y-5">
              <div className="text-center mb-4">
                <FileText size={32} className="mx-auto text-blue-500 mb-2" />
                <h2 className="text-lg font-semibold">大纲预览</h2>
                <p className="text-sm text-muted-foreground mt-1">
                  叙事模式：{storyArc.narrative_pattern}
                </p>
              </div>

              <div className="space-y-3 max-h-[calc(100vh-320px)] overflow-y-auto">
                {storyArc.slides.map((slide, i) => (
                  <div key={i} className="p-4 rounded-lg border border-border bg-card">
                    <div className="flex items-start gap-3">
                      <div className="flex-shrink-0 w-7 h-7 rounded-full bg-blue-500 text-white flex items-center justify-center text-xs font-medium">
                        {slide.position}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-blue-100 text-blue-700">
                            {slide.role}
                          </span>
                          <span className="text-xs text-muted-foreground">{slide.emotional_beat}</span>
                        </div>
                        <p className="text-sm font-medium">{slide.core_message}</p>
                        {slide.transition_logic && (
                          <p className="text-xs text-muted-foreground mt-1">过渡：{slide.transition_logic}</p>
                        )}
                        {slide.data_points && slide.data_points.length > 0 && (
                          <div className="flex flex-wrap gap-1 mt-2">
                            {slide.data_points.map((dp, j) => (
                              <span key={j} className="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground">{dp}</span>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              {error && (
                <div className="p-3 rounded-lg bg-destructive/10 text-destructive text-sm">{error}</div>
              )}

              <div className="flex items-center justify-center gap-3 pt-2">
                <button
                  onClick={handleConfirmOutline}
                  disabled={confirmOutline.isPending}
                  className="flex items-center gap-1.5 px-5 py-2.5 text-sm font-medium text-white bg-blue-500 hover:bg-blue-600 rounded-lg transition-colors cursor-pointer disabled:opacity-50"
                >
                  {confirmOutline.isPending ? (
                    <Loader2 size={14} className="animate-spin" />
                  ) : (
                    <CheckCircle2 size={14} />
                  )}
                  {confirmOutline.isPending ? "确认中..." : "确认生成"}
                </button>
                <button
                  onClick={handleReset}
                  className="flex items-center gap-1.5 px-5 py-2.5 text-sm font-medium text-muted-foreground bg-muted hover:bg-muted/80 rounded-lg transition-colors cursor-pointer"
                >
                  <RefreshCw size={14} />
                  放弃
                </button>
              </div>
            </div>
          </div>
        )}

        {phase === "preview" && presentation && (
          <div className="flex flex-col items-center p-6 gap-4">
            <div className="flex items-center gap-3">
              <button
                onClick={handleExport}
                disabled={exporting}
                className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-white bg-blue-500 hover:bg-blue-600 rounded-lg transition-colors cursor-pointer disabled:opacity-50"
              >
                {exporting ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
                {exporting ? "导出中..." : "下载 PPTX"}
              </button>
              <button
                onClick={handleReset}
                className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-muted-foreground bg-muted hover:bg-muted/80 rounded-lg transition-colors cursor-pointer"
              >
                <RefreshCw size={14} />
                重新生成
              </button>
              <button
                onClick={() => setEditingSlideIndex(currentSlide)}
                className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-muted-foreground bg-muted hover:bg-muted/80 rounded-lg transition-colors cursor-pointer"
              >
                <Pencil size={14} />
                编辑当前页
              </button>
              {savePresentation.isPending && (
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Loader2 size={12} className="animate-spin" />保存中...
                </span>
              )}
              {lastSavedTaskId && !savePresentation.isPending && (
                <span className="text-xs text-green-600">已保存</span>
              )}
            </div>
            <SlidePreview
              data={presentation}
              theme={selectedTheme}
              currentSlide={currentSlide}
              onSlideChange={setCurrentSlide}
            />
          </div>
        )}

        {editingSlideIndex !== null && presentation && (
          <SlideEditor
            slide={presentation.slides[editingSlideIndex]}
            slideIndex={editingSlideIndex}
            onSave={handleSlideEdit}
            onClose={() => setEditingSlideIndex(null)}
          />
        )}
      </div>
    </div>
      </SidebarInset>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将删除该演示文稿，是否继续？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete} className="bg-destructive hover:bg-destructive/90">
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SidebarProvider>
  );
};

export default PptPage;
