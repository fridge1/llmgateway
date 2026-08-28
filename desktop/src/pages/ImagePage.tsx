import { useState } from 'react';
import { Image, KeyRound } from 'lucide-react';
import { GenerationForm } from '@/components/image/GenerationForm';
import { TaskList } from '@/components/image/TaskList';
import { useApiKeys, useGatewayModels } from '@/hooks/use-api';

export default function ImagePage() {
  // Desktop is always a normal logged-in user — no image-share mode.
  const { data: apiKeys = [] } = useApiKeys();
  const { data: allModels = [] } = useGatewayModels();

  const [userPickedKeyId, setUserPickedKeyId] = useState('');
  const [editImageUrl, setEditImageUrl] = useState<string | null>(null);

  const selectedKeyId =
    apiKeys.length > 0
      ? apiKeys.some((k) => k.id === userPickedKeyId)
        ? userPickedKeyId
        : apiKeys[0].id
      : userPickedKeyId;

  const imageModels = allModels.filter(
    (m) =>
      m.category === 'image-generation' ||
      m.category === 'image' ||
      m.name.includes('dall-e') ||
      m.name.includes('stable-diffusion') ||
      m.name.includes('image'),
  );

  const ready = selectedKeyId && apiKeys.length > 0;

  return (
    <div className="h-screen flex flex-col bg-obsidian-950">
      {/* Header */}
      <header className="h-12 flex items-center justify-between px-4 border-b border-obsidian-700 bg-obsidian-900 shrink-0">
        <div className="flex items-center gap-1.5">
          <Image size={14} className="text-pink-500" />
          <span className="font-semibold text-sm text-obsidian-50">图片生成</span>
        </div>
        <div className="flex items-center gap-2">
          <KeyRound size={12} className="text-obsidian-400" />
          <select
            value={selectedKeyId}
            onChange={(e) => setUserPickedKeyId(e.target.value)}
            className="h-7 px-2 text-xs border border-obsidian-700 rounded-md bg-obsidian-800 text-obsidian-100 focus:outline-none focus:border-amber-500"
          >
            <option value="">选择密钥...</option>
            {apiKeys.map((k) => (
              <option key={k.id} value={k.id}>{k.name}</option>
            ))}
          </select>
        </div>
      </header>

      {/* Main */}
      <div className="flex-1 overflow-hidden">
        {ready ? (
          <div className="flex h-full">
            <div className="w-[380px] shrink-0 border-r border-obsidian-700 overflow-y-auto p-5">
              <GenerationForm
                keyId={selectedKeyId}
                availableModels={imageModels}
                editImageUrl={editImageUrl}
                onEditImageConsumed={() => setEditImageUrl(null)}
              />
            </div>
            <div className="flex-1 overflow-y-auto p-5">
              <TaskList onEditImage={setEditImageUrl} />
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <KeyRound size={40} className="text-obsidian-600 mb-3" />
            <p className="text-sm text-obsidian-400">请先选择 API 密钥</p>
            <p className="text-xs text-obsidian-500 mt-1">在右上角选择一个已创建的密钥开始使用</p>
          </div>
        )}
      </div>
    </div>
  );
}
