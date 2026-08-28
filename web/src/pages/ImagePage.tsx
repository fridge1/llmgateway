import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Image, KeyRound } from 'lucide-react';
import { GenerationForm } from '@/components/image/GenerationForm';
import { TaskList } from '@/components/image/TaskList';
import { useApiKeys, useGatewayModels, useImageShareModels } from '@/hooks/use-api';
import { useAuth } from '@/contexts/AuthContext';

export default function ImagePage() {
  const navigate = useNavigate();
  const auth = useAuth();
  const isImageShare = auth.isImageShare;
  const { data: apiKeys = [] } = useApiKeys({ enabled: !isImageShare });
  const { data: regularModels = [] } = useGatewayModels({ enabled: !isImageShare });
  const { data: shareModels = [] } = useImageShareModels({ enabled: isImageShare });
  const allModels = isImageShare ? shareModels : regularModels;

  // userPickedKeyId is the explicit user selection; falls back to apiKeys[0]
  // when unset so we don't need a setState-in-effect.
  const [userPickedKeyId, setUserPickedKeyId] = useState<string>('');
  const [editImageUrl, setEditImageUrl] = useState<string | null>(null);

  const selectedKeyId =
    !isImageShare && apiKeys.length > 0
      ? apiKeys.some((k) => k.id === userPickedKeyId)
        ? userPickedKeyId
        : apiKeys[0].id
      : userPickedKeyId;
  const setSelectedKeyId = setUserPickedKeyId;

  const imageModels = (allModels || []).filter(m => {
    // Image-Share 用户仅显示 gpt-image-2
    if (isImageShare) {
      return m.name === 'gpt-image-2';
    }
    // 普通 JWT 用户保持原有逻辑
    return m.category === 'image-generation' ||
      m.category === 'image' ||
      m.name.includes('dall-e') ||
      m.name.includes('stable-diffusion') ||
      m.name.includes('image');
  });

  // image-share callers don't need a regular API key — pass a sentinel so the form renders.
  const effectiveKeyId = isImageShare ? 'image-share' : selectedKeyId;
  const ready = isImageShare || (selectedKeyId && apiKeys.length > 0);

  return (
    <div className={isImageShare ? 'flex flex-col bg-background' : 'h-screen flex flex-col bg-background'}>
      {!isImageShare && (
        <header className="h-12 flex items-center justify-between px-4 border-b border-border bg-card shrink-0">
          <div className="flex items-center gap-3">
            <button
              onClick={() => navigate('/tools')}
              className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              <ArrowLeft size={16} />
              返回
            </button>
            <div className="w-px h-4 bg-border" />
            <div className="flex items-center gap-1.5">
              <Image size={14} className="text-pink-500" />
              <span className="font-semibold text-sm">图片生成</span>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <KeyRound size={12} />
            </div>
            <select
              value={selectedKeyId}
              onChange={(e) => setSelectedKeyId(e.target.value)}
              className="h-7 px-2 text-xs border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="">选择密钥...</option>
              {apiKeys.map((k) => (
                <option key={k.id} value={k.id}>
                  {k.name}
                </option>
              ))}
            </select>
          </div>
        </header>
      )}

      {/* Main content */}
      <div className={isImageShare ? 'min-h-[calc(100vh-7rem)]' : 'flex-1 overflow-hidden'}>
        {ready ? (
          <div className={isImageShare ? 'flex flex-col md:flex-row h-full' : 'flex h-full'}>
            <div className={isImageShare ? 'w-full md:w-[420px] shrink-0 border-b md:border-b-0 md:border-r border-border overflow-y-auto p-5' : 'w-[420px] shrink-0 border-r border-border overflow-y-auto p-5'}>
              <GenerationForm
                keyId={effectiveKeyId}
                availableModels={imageModels}
                editImageUrl={editImageUrl}
                onEditImageConsumed={() => setEditImageUrl(null)}
                imageShareMode={isImageShare}
              />
            </div>
            <div className="flex-1 overflow-y-auto p-5">
              <TaskList onEditImage={setEditImageUrl} />
            </div>
          </div>
        ) : (
          <div className="empty-state flex flex-col items-center justify-center py-16">
            <KeyRound size={40} className="text-muted-foreground/40 mb-3" />
            <p className="text-sm text-muted-foreground">请先选择 API 密钥</p>
            <p className="text-xs text-muted-foreground/60 mt-1">在右上角选择一个已创建的密钥开始使用</p>
          </div>
        )}
      </div>
    </div>
  );
}
