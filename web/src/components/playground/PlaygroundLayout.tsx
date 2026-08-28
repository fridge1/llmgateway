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
    <div className="h-screen flex flex-col bg-background">
      {/* Minimal toolbar */}
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
            <Zap size={14} className="text-primary" />
            <span className="font-semibold text-sm">AI 对话</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* Mode toggle */}
          <div className="flex items-center bg-muted rounded-lg p-0.5">
            <button
              onClick={() => onModeChange("single")}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                mode === "single"
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              单模型
            </button>
            <button
              onClick={() => onModeChange("compare")}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                mode === "compare"
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              模型对比
            </button>
          </div>
          <button className="w-8 h-8 flex items-center justify-center rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer">
            <Moon size={16} />
          </button>
        </div>
      </header>

      {/* Content area */}
      <div className="flex-1 overflow-hidden">
        {children}
      </div>
    </div>
  );
};

export default PlaygroundLayout;
