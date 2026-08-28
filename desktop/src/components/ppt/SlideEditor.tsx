import { useState } from "react";
import { X, Plus, Trash2, GripVertical } from "lucide-react";
import type { SlideData, SlideLayout } from "@/types/ppt";

const layoutOptions: { value: SlideLayout; label: string }[] = [
  { value: "title", label: "标题页" },
  { value: "content", label: "内容页" },
  { value: "section", label: "章节页" },
  { value: "closing", label: "结束页" },
  { value: "comparison", label: "对比页" },
  { value: "timeline", label: "时间线" },
  { value: "data_highlight", label: "数据亮点" },
  { value: "image_text", label: "图文页" },
  { value: "quote", label: "引用页" },
  { value: "two_column", label: "双栏页" },
  { value: "chart", label: "图表页" },
];

interface SlideEditorProps {
  slide: SlideData;
  slideIndex: number;
  onSave: (slide: SlideData) => void;
  onClose: () => void;
}

export default function SlideEditor({ slide, slideIndex, onSave, onClose }: SlideEditorProps) {
  const [layout, setLayout] = useState<SlideLayout>(slide.layout);
  const [title, setTitle] = useState(slide.title);
  const [subtitle, setSubtitle] = useState(slide.subtitle || "");
  const [bullets, setBullets] = useState<string[]>(slide.bullets || []);
  const [speakerNotes, setSpeakerNotes] = useState(slide.speakerNotes || "");
  const [dragIndex, setDragIndex] = useState<number | null>(null);

  const handleAddBullet = () => {
    setBullets([...bullets, ""]);
  };

  const handleRemoveBullet = (index: number) => {
    setBullets(bullets.filter((_, i) => i !== index));
  };

  const handleBulletChange = (index: number, value: string) => {
    const updated = [...bullets];
    updated[index] = value;
    setBullets(updated);
  };

  const handleDragStart = (index: number) => {
    setDragIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (dragIndex === null || dragIndex === index) return;
    const updated = [...bullets];
    const [moved] = updated.splice(dragIndex, 1);
    updated.splice(index, 0, moved);
    setBullets(updated);
    setDragIndex(index);
  };

  const handleDragEnd = () => {
    setDragIndex(null);
  };

  const handleSave = () => {
    const updated: SlideData = {
      ...slide,
      layout,
      title,
      subtitle: subtitle || undefined,
      bullets: bullets.filter((b) => b.trim()),
      speakerNotes: speakerNotes || undefined,
    };
    onSave(updated);
  };

  const inputClass =
    "w-full h-8 px-2 text-sm border border-obsidian-700 rounded-md bg-obsidian-900 text-obsidian-100 focus:outline-none focus:ring-1 focus:ring-amber-500";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-obsidian-900 border border-obsidian-700 rounded-xl shadow-xl w-full max-w-lg max-h-[85vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-3 border-b border-obsidian-700">
          <h3 className="text-sm font-semibold text-obsidian-50">编辑幻灯片 #{slideIndex + 1}</h3>
          <button onClick={onClose} className="p-1 rounded text-obsidian-400 hover:bg-obsidian-800 hover:text-obsidian-100 transition-colors cursor-pointer">
            <X size={16} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
          <div>
            <label className="block text-xs font-medium mb-1 text-obsidian-200">布局</label>
            <select
              value={layout}
              onChange={(e) => setLayout(e.target.value as SlideLayout)}
              className={inputClass}
            >
              {layoutOptions.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium mb-1 text-obsidian-200">标题</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className={inputClass}
            />
          </div>

          <div>
            <label className="block text-xs font-medium mb-1 text-obsidian-200">副标题</label>
            <input
              type="text"
              value={subtitle}
              onChange={(e) => setSubtitle(e.target.value)}
              placeholder="可选"
              className={`${inputClass} placeholder:text-obsidian-500`}
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-xs font-medium text-obsidian-200">要点</label>
              <button
                onClick={handleAddBullet}
                className="flex items-center gap-1 text-xs text-amber-400 hover:text-amber-300 cursor-pointer"
              >
                <Plus size={12} />
                添加
              </button>
            </div>
            <div className="space-y-1.5">
              {bullets.map((bullet, i) => (
                <div
                  key={i}
                  draggable
                  onDragStart={() => handleDragStart(i)}
                  onDragOver={(e) => handleDragOver(e, i)}
                  onDragEnd={handleDragEnd}
                  className={`flex items-center gap-1.5 group ${dragIndex === i ? "opacity-50" : ""}`}
                >
                  <GripVertical size={14} className="text-obsidian-400 cursor-grab shrink-0" />
                  <input
                    type="text"
                    value={bullet}
                    onChange={(e) => handleBulletChange(i, e.target.value)}
                    className="flex-1 h-7 px-2 text-sm border border-obsidian-700 rounded-md bg-obsidian-900 text-obsidian-100 focus:outline-none focus:ring-1 focus:ring-amber-500"
                  />
                  <button
                    onClick={() => handleRemoveBullet(i)}
                    className="p-0.5 text-obsidian-400 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium mb-1 text-obsidian-200">演讲备注</label>
            <textarea
              value={speakerNotes}
              onChange={(e) => setSpeakerNotes(e.target.value)}
              placeholder="可选"
              rows={3}
              className="w-full px-2 py-1.5 text-sm border border-obsidian-700 rounded-md bg-obsidian-900 text-obsidian-100 placeholder:text-obsidian-500 resize-none focus:outline-none focus:ring-1 focus:ring-amber-500"
            />
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-obsidian-700">
          <button
            onClick={onClose}
            className="px-4 py-1.5 text-sm text-obsidian-200 bg-obsidian-800 hover:bg-obsidian-700 rounded-md transition-colors cursor-pointer"
          >
            取消
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-1.5 text-sm font-medium text-obsidian-950 bg-amber-500 hover:bg-amber-400 rounded-md transition-colors cursor-pointer"
          >
            保存
          </button>
        </div>
      </div>
    </div>
  );
}
