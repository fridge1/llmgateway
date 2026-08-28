import { useState, useRef, useCallback, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Loader2, Upload, X, ImagePlus, Image, Sparkles } from 'lucide-react';
import { toast } from 'sonner';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useSubmitTask, useSubmitEditTask } from '@/hooks/use-image-tasks';
import { useBalance } from '@/hooks/use-api';
import { SubmitTaskRequest, EditTaskRequest, ImageTaskParams } from '@/types/image';
import type { GatewayModel } from '@/types/api';

interface GenerationFormProps {
  keyId: string;
  availableModels: GatewayModel[];
  editImageUrl?: string | null;
  onEditImageConsumed?: () => void;
  imageShareMode?: boolean;
}

const COUNTS = [
  { value: 1, label: '1 张' },
  { value: 2, label: '2 张' },
  { value: 3, label: '3 张' },
  { value: 4, label: '4 张' },
];

const MAX_IMAGES = 4;
const MAX_IMAGE_SIZE = 25 * 1024 * 1024;
const ACCEPTED_TYPES = ['image/png', 'image/jpeg', 'image/webp'];

type SelectOption = { value: string; label: string };
type ParamField = {
  key: string;
  label: string;
  options: SelectOption[];
  default: string;
};

type ModelSpec = {
  fields: ParamField[];
  // 固定送给后端的 size（仅用于计费档位）。空字符串表示由 fields 中的 size 决定。
  fixedSize?: string;
  // 哪个 field 的 key 应当作为请求中的 size 字段（默认无，使用 fixedSize 或 1024x1024）
  sizeFieldKey?: string;
};

const GEMINI_BASE_RATIOS: SelectOption[] = [
  { value: '1:1', label: '1:1 正方形' },
  { value: '16:9', label: '16:9 宽屏' },
  { value: '9:16', label: '9:16 竖屏' },
  { value: '4:3', label: '4:3 横向' },
  { value: '3:4', label: '3:4 纵向' },
  { value: '3:2', label: '3:2 横向' },
  { value: '2:3', label: '2:3 纵向' },
  { value: '5:4', label: '5:4 横向' },
  { value: '4:5', label: '4:5 纵向' },
  { value: '21:9', label: '21:9 超宽' },
];

const GEMINI_FLASH_EXTRA_RATIOS: SelectOption[] = [
  { value: '4:1', label: '4:1 全景' },
  { value: '1:4', label: '1:4 长条' },
  { value: '8:1', label: '8:1 横幅' },
  { value: '1:8', label: '1:8 立柱' },
];

// gpt-image-2 supported sizes: 30 combinations (10 aspect ratios × 1K/2K/4K)
// plus an "auto" entry. Each value satisfies the OpenAI gpt-image-2 size
// constraints enforced by ParseSize on the server (edges multiple of 16,
// long/short <= 3:1, pixels in [655360, 8294400]). Grouped by aspect ratio
// so users see related options together; resolution-tier name is intentionally
// not surfaced because the API only accepts a "size" string.
const GPT_IMAGE_SIZE_OPTIONS: SelectOption[] = [
  { value: 'auto', label: '自动' },
  // 1:1
  { value: '1280x1280', label: '1280×1280 (1:1)' },
  { value: '2048x2048', label: '2048×2048 (1:1)' },
  { value: '2880x2880', label: '2880×2880 (1:1)' },
  // 3:2
  { value: '1280x848',  label: '1280×848 (3:2)' },
  { value: '2048x1360', label: '2048×1360 (3:2)' },
  { value: '3520x2336', label: '3520×2336 (3:2)' },
  // 2:3
  { value: '848x1280',  label: '848×1280 (2:3)' },
  { value: '1360x2048', label: '1360×2048 (2:3)' },
  { value: '2336x3520', label: '2336×3520 (2:3)' },
  // 4:3
  { value: '1280x960',  label: '1280×960 (4:3)' },
  { value: '2048x1536', label: '2048×1536 (4:3)' },
  { value: '3312x2480', label: '3312×2480 (4:3)' },
  // 3:4
  { value: '960x1280',  label: '960×1280 (3:4)' },
  { value: '1536x2048', label: '1536×2048 (3:4)' },
  { value: '2480x3312', label: '2480×3312 (3:4)' },
  // 5:4
  { value: '1280x1024', label: '1280×1024 (5:4)' },
  { value: '2048x1632', label: '2048×1632 (5:4)' },
  { value: '3216x2560', label: '3216×2560 (5:4)' },
  // 4:5
  { value: '1024x1280', label: '1024×1280 (4:5)' },
  { value: '1632x2048', label: '1632×2048 (4:5)' },
  { value: '2560x3216', label: '2560×3216 (4:5)' },
  // 16:9
  { value: '1280x720',  label: '1280×720 (16:9)' },
  { value: '2048x1152', label: '2048×1152 (16:9)' },
  { value: '3840x2160', label: '3840×2160 (16:9)' },
  // 9:16
  { value: '720x1280',  label: '720×1280 (9:16)' },
  { value: '1152x2048', label: '1152×2048 (9:16)' },
  { value: '2160x3840', label: '2160×3840 (9:16)' },
  // 21:9
  { value: '1280x544',  label: '1280×544 (21:9)' },
  { value: '2048x864',  label: '2048×864 (21:9)' },
  { value: '3840x1632', label: '3840×1632 (21:9)' },
];

function getModelSpec(model: string): ModelSpec {
  const isGeminiImage = /^gemini-.*image/i.test(model);
  const isFlash = isGeminiImage && /flash/i.test(model);
  const isProImage = isGeminiImage && /pro/i.test(model);
  const isGptImage = /^gpt-image/i.test(model);

  if (isGeminiImage) {
    const ratios = isFlash
      ? [...GEMINI_BASE_RATIOS, ...GEMINI_FLASH_EXTRA_RATIOS]
      : GEMINI_BASE_RATIOS;
    const resolutions: SelectOption[] = isFlash
      ? [
          { value: '512', label: '512' },
          { value: '1K', label: '1K' },
          { value: '2K', label: '2K' },
          { value: '4K', label: '4K' },
        ]
      : [
          { value: '1K', label: '1K' },
          { value: '2K', label: '2K' },
          { value: '4K', label: '4K' },
        ];
    return {
      fields: [
        {
          key: 'aspect_ratio',
          label: '宽高比',
          options: ratios,
          default: '1:1',
        },
        {
          key: 'image_resolution',
          label: '分辨率',
          options: resolutions,
          default: isProImage ? '2K' : '1K',
        },
      ],
      fixedSize: '1024x1024',
    };
  }

  if (isGptImage) {
    return {
      fields: [
        {
          key: 'size',
          label: '尺寸',
          options: GPT_IMAGE_SIZE_OPTIONS,
          default: '1280x1280',
        },
        {
          key: 'quality',
          label: '质量',
          options: [
            { value: 'auto', label: '自动' },
            { value: 'low', label: '低' },
            { value: 'medium', label: '中' },
            { value: 'high', label: '高' },
          ],
          default: 'auto',
        },
      ],
      sizeFieldKey: 'size',
    };
  }

  // 兜底：保留原有 size 三选一
  return {
    fields: [
      {
        key: 'size',
        label: '尺寸',
        options: [
          { value: '1024x1024', label: '1024×1024 (1:1)' },
          { value: '1024x1792', label: '1024×1792 (9:16)' },
          { value: '1792x1024', label: '1792×1024 (16:9)' },
        ],
        default: '1024x1024',
      },
    ],
    sizeFieldKey: 'size',
  };
}

function buildDefaultValues(spec: ModelSpec): Record<string, string> {
  const values: Record<string, string> = {};
  for (const f of spec.fields) values[f.key] = f.default;
  return values;
}

export function GenerationForm({ keyId, availableModels, editImageUrl, onEditImageConsumed }: GenerationFormProps) {
  const submitTask = useSubmitTask();
  const submitEditTask = useSubmitEditTask();
  const balance = useBalance();
  const navigate = useNavigate();

  const [prompt, setPrompt] = useState('');
  const [model, setModel] = useState((availableModels && availableModels.length > 0) ? availableModels[0].name : '');
  const [count, setCount] = useState(1);
  const [referenceImages, setReferenceImages] = useState<File[]>([]);
  const [maskImage, setMaskImage] = useState<File | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const maskInputRef = useRef<HTMLInputElement>(null);

  const spec = useMemo(() => getModelSpec(model), [model]);
  const [paramValues, setParamValues] = useState<Record<string, string>>(() => buildDefaultValues(spec));

  useEffect(() => {
    setParamValues(buildDefaultValues(spec));
  }, [spec]);

  const isEditing = referenceImages.length > 0;
  const isPending = submitTask.isPending || submitEditTask.isPending;

  useEffect(() => {
    if (!editImageUrl) return;
    let cancelled = false;
    (async () => {
      try {
        let fetchUrl = editImageUrl;
        if (fetchUrl.includes('your-tos-bucket.tos-cn-beijing.volces.com')) {
          fetchUrl = fetchUrl.replace(
            'https://your-tos-bucket.tos-cn-beijing.volces.com',
            '/tos-proxy'
          );
        }
        const res = await fetch(fetchUrl);
        if (cancelled) return;
        const blob = await res.blob();
        if (cancelled) return;
        const ext = blob.type === 'image/png' ? '.png' : blob.type === 'image/webp' ? '.webp' : '.jpg';
        const file = new window.File([blob], `edit${ext}`, { type: blob.type || 'image/png' });
        setReferenceImages(prev => {
          if (prev.length >= MAX_IMAGES) {
            toast.error(`最多上传 ${MAX_IMAGES} 张参考图片`);
            return prev;
          }
          return [...prev, file];
        });
        toast.success('已添加为参考图片，请输入修改提示词');
      } catch {
        if (!cancelled) toast.error('加载图片失败，请重试');
      } finally {
        if (!cancelled) onEditImageConsumed?.();
      }
    })();
    return () => { cancelled = true; };
  }, [editImageUrl, onEditImageConsumed]);

  const validateAndAddFiles = useCallback((files: FileList | File[]) => {
    const newFiles: File[] = [];
    for (const file of Array.from(files)) {
      if (!ACCEPTED_TYPES.includes(file.type)) {
        toast.error(`不支持的格式: ${file.name}，仅支持 PNG/JPEG/WebP`);
        continue;
      }
      if (file.size > MAX_IMAGE_SIZE) {
        toast.error(`文件过大: ${file.name}，最大 25MB`);
        continue;
      }
      newFiles.push(file);
    }
    setReferenceImages(prev => {
      const combined = [...prev, ...newFiles];
      if (combined.length > MAX_IMAGES) {
        toast.error(`最多上传 ${MAX_IMAGES} 张参考图片`);
        return combined.slice(0, MAX_IMAGES);
      }
      return combined;
    });
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    if (e.dataTransfer.files.length > 0) {
      validateAndAddFiles(e.dataTransfer.files);
    }
  }, [validateAndAddFiles]);

  const removeImage = (index: number) => {
    setReferenceImages(prev => prev.filter((_, i) => i !== index));
    if (referenceImages.length <= 1) {
      setMaskImage(null);
    }
  };

  const buildSubmitPayload = (): { size: string; params?: ImageTaskParams } => {
    const sizeFieldKey = spec.sizeFieldKey;
    let size = '1024x1024';
    const params: ImageTaskParams = {};
    for (const f of spec.fields) {
      const v = paramValues[f.key] ?? f.default;
      if (sizeFieldKey === f.key) {
        // size 字段：'auto' 也作为非合法 WxH，回退到 1024x1024 仅用作计费档位，把真实值放进 params
        if (/^\d+x\d+$/.test(v)) {
          size = v;
        } else {
          size = '1024x1024';
          params[f.key] = v;
        }
      } else {
        params[f.key] = v;
      }
    }
    if (spec.fixedSize) size = spec.fixedSize;
    return { size, params: Object.keys(params).length > 0 ? params : undefined };
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!prompt.trim()) {
      toast.error('请输入提示词');
      return;
    }
    if (prompt.length > 2000) {
      toast.error('提示词长度不能超过 2000 个字符');
      return;
    }
    if (!model) {
      toast.error('请选择模型');
      return;
    }

    if (balance.data && balance.data.balance <= 0) {
      toast.error('余额不足，请先充值后再使用图片生成', {
        action: { label: '去充值', onClick: () => navigate('/balance') },
      });
      return;
    }

    const { size, params } = buildSubmitPayload();

    try {
      if (isEditing) {
        const request: EditTaskRequest = {
          model,
          prompt: prompt.trim(),
          size,
          n: count,
          images: referenceImages,
          mask: maskImage || undefined,
          params,
        };
        await submitEditTask.mutateAsync(request);
      } else {
        const request: SubmitTaskRequest = {
          model,
          prompt: prompt.trim(),
          size,
          n: count,
          params,
        };
        await submitTask.mutateAsync(request);
      }
      toast.success('任务已提交');
      setPrompt('');
      setReferenceImages([]);
      setMaskImage(null);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '任务提交失败');
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      const form = e.currentTarget.closest('form');
      if (form) form.requestSubmit();
    }
  };

  if (!availableModels || availableModels.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <Image size={32} className="text-muted-foreground/40 mb-3" />
        <p className="text-sm text-muted-foreground">暂无可用的图片生成模型</p>
        <p className="text-xs text-muted-foreground/60 mt-1">请联系管理员配置</p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5 slide-up">
      {/* Header */}
      <div className="text-center pb-2">
        <div className="w-10 h-10 bg-pink-500 rounded-xl flex items-center justify-center mx-auto mb-2.5">
          <Image size={20} className="text-white" />
        </div>
        <h2 className="text-base font-semibold">{isEditing ? '图片编辑' : '图片生成'}</h2>
        <p className="text-xs text-muted-foreground mt-1">
          {isEditing ? '上传参考图并描述编辑需求' : '输入提示词，AI 为你生成图片'}
        </p>
      </div>

      {/* Prompt */}
      <div className="space-y-1.5">
        <Label htmlFor="prompt" className="text-xs font-medium">提示词</Label>
        <textarea
          id="prompt"
          placeholder={isEditing ? "描述你想要对图片进行的编辑..." : "描述你想要生成的图片..."}
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          onKeyDown={handleKeyDown}
          rows={4}
          maxLength={2000}
          className="w-full px-3 py-2 text-sm border border-border rounded-lg bg-background text-foreground resize-none focus:outline-none focus:ring-1 focus:ring-primary placeholder:text-muted-foreground"
        />
        <div className="text-[11px] text-muted-foreground text-right">
          {prompt.length} / 2000
        </div>
      </div>

      {/* Image Upload */}
      <div className="space-y-1.5">
        <Label className="text-xs font-medium">参考图片 <span className="text-muted-foreground font-normal">（可选，上传后进入编辑模式）</span></Label>
        <div
          className={`border-2 border-dashed rounded-xl p-5 text-center cursor-pointer transition-all duration-200 ${
            dragOver
              ? 'border-pink-400 bg-pink-500/5'
              : 'border-border hover:border-muted-foreground/40 hover:bg-muted/30'
          }`}
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
        >
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            accept="image/png,image/jpeg,image/webp"
            multiple
            onChange={(e) => {
              if (e.target.files) validateAndAddFiles(e.target.files);
              e.target.value = '';
            }}
          />
          <Upload className="w-5 h-5 mx-auto mb-1.5 text-muted-foreground/60" />
          <p className="text-xs text-muted-foreground">
            点击或拖拽上传 · PNG/JPEG/WebP · 最多 {MAX_IMAGES} 张 · ≤ 25MB
          </p>
        </div>

        {/* Thumbnails */}
        {referenceImages.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-2">
            {referenceImages.map((file, index) => (
              <div key={index} className="relative group w-16 h-16">
                <img
                  src={URL.createObjectURL(file)}
                  alt={`参考图 ${index + 1}`}
                  className="w-full h-full object-cover rounded-lg border border-border transition-transform duration-200 group-hover:scale-105"
                />
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); removeImage(index); }}
                  className="absolute -top-1.5 -right-1.5 bg-destructive text-destructive-foreground rounded-full w-4 h-4 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <X className="w-2.5 h-2.5" />
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Mask upload */}
        {referenceImages.length === 1 && (
          <div className="mt-2 space-y-1">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => maskInputRef.current?.click()}
                className="flex items-center gap-1 px-2.5 py-1 text-xs border border-border rounded-md hover:bg-muted transition-colors cursor-pointer"
              >
                <ImagePlus className="w-3 h-3" />
                {maskImage ? '更换蒙版' : '添加蒙版'}
              </button>
              {maskImage && (
                <div className="flex items-center gap-1 text-xs text-muted-foreground">
                  <span className="truncate max-w-[120px]">{maskImage.name}</span>
                  <button type="button" onClick={() => setMaskImage(null)} className="cursor-pointer">
                    <X className="w-3 h-3" />
                  </button>
                </div>
              )}
            </div>
            <input
              ref={maskInputRef}
              type="file"
              className="hidden"
              accept="image/png"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) setMaskImage(f);
                e.target.value = '';
              }}
            />
            <p className="text-[11px] text-muted-foreground">
              PNG 格式，透明区域表示需要编辑的位置
            </p>
          </div>
        )}
      </div>

      {/* Model + dynamic params + count */}
      <div className="space-y-3">
        <div className="space-y-1.5">
          <Label htmlFor="model" className="text-xs font-medium">模型</Label>
          <Select value={model} onValueChange={setModel}>
            <SelectTrigger id="model" className="h-8 text-xs">
              <SelectValue placeholder="选择模型" />
            </SelectTrigger>
            <SelectContent>
              {availableModels.map((m) => (
                <SelectItem key={m.name} value={m.name}>
                  {m.display_name || m.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="grid grid-cols-2 gap-3">
          {spec.fields.map((field) => (
            <div key={field.key} className="space-y-1.5">
              <Label htmlFor={field.key} className="text-xs font-medium">{field.label}</Label>
              <Select
                value={paramValues[field.key] ?? field.default}
                onValueChange={(v) => setParamValues(prev => ({ ...prev, [field.key]: v }))}
              >
                <SelectTrigger id={field.key} className="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {field.options.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ))}

          <div className="space-y-1.5">
            <Label htmlFor="count" className="text-xs font-medium">数量</Label>
            <Select value={count.toString()} onValueChange={(v) => setCount(parseInt(v))}>
              <SelectTrigger id="count" className="h-8 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {COUNTS.map((c) => (
                  <SelectItem key={c.value} value={c.value.toString()}>
                    {c.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      {/* Submit */}
      <button
        type="submit"
        disabled={isPending || !prompt.trim() || !model}
        className="w-full flex items-center justify-center gap-2 h-10 text-sm font-medium text-white rounded-lg transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed active:scale-[0.97]"
        style={{
          background: 'linear-gradient(135deg, #EC4899, #F43F5E)',
          boxShadow: '0 1px 2px rgba(236, 72, 153, 0.2), 0 2px 4px rgba(236, 72, 153, 0.12)',
        }}
      >
        {isPending ? (
          <>
            <Loader2 className="w-4 h-4 animate-spin" />
            提交中...
          </>
        ) : (
          <>
            <Sparkles className="w-4 h-4" />
            {isEditing ? '提交编辑任务' : '提交生成任务'}
          </>
        )}
      </button>

      <p className="text-[11px] text-center text-muted-foreground">Ctrl + Enter 快速提交</p>
    </form>
  );
}
