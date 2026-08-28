import { useNavigate } from "react-router-dom";
import { ArrowLeft, Zap, Moon } from "lucide-react";

interface PlaygroundLayoutProps {
  mode: "single" | "compare";
  onModeChange: (mode: "single" | "compare") => void;
  children: React.ReactNode;
}

const PlaygroundLayout = ({ mode, onModeChange, children }: PlaygroundLayoutProps) => {
  const navigate = useNavigate();

  return (
    <div className="h-screen flex flex-col bg-obsidian-950">
      {/* Minimal toolbar */}
      <header className="h-12 flex items-center justify-between px-4 border-b border-obsidian-700 bg-obsidian-900 shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate("/tools")}
            className="flex items-center gap-1.5 text-sm text-obsidian-400 hover:text-obsidian-100 transition-colors cursor-pointer"
          >
            <ArrowLeft size={16} />
            返回
          </button>
          <div className="w-px h-4 bg-obsidian-700" />
          <div className="flex items-center gap-1.5">
            <Zap size={14} className="text-amber-400" />
            <span className="font-semibold text-sm text-obsidian-50">AI 对话</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* Mode toggle */}
          <div className="flex items-center bg-obsidian-800 rounded-lg p-0.5">
            <button
              onClick={() => onModeChange("single")}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                mode === "single"
                  ? "bg-amber-500 text-obsidian-950"
                  : "text-obsidian-400 hover:text-obsidian-100"
              }`}
            >
              单模型
            </button>
            <button
              onClick={() => onModeChange("compare")}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                mode === "compare"
                  ? "bg-amber-500 text-obsidian-950"
                  : "text-obsidian-400 hover:text-obsidian-100"
              }`}
            >
              模型对比
            </button>
          </div>
          <button className="w-8 h-8 flex items-center justify-center rounded-lg text-obsidian-400 hover:text-obsidian-100 hover:bg-obsidian-800 transition-colors cursor-pointer">
            <Moon size={16} />
          </button>
        </div>
      </header>

      {/* Content area */}
      <div className="flex-1 overflow-hidden">{children}</div>
    </div>
  );
};

export default PlaygroundLayout;
