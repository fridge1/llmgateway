import { useNavigate } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { tools } from "@/config/tools";
import { Wrench } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";

const ToolsHubPage = () => {
  const navigate = useNavigate();

  return (
    <div className="page-container fade-in">
      <PageHeader
        eyebrow="工具"
        title={<span className="flex items-center gap-2"><Wrench size={20} className="text-primary" />AI 工具集</span>}
        description="一站式访问 PPT、图片、翻译等内置 AI 工具，直接在线使用。"
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {tools.map((tool, i) => {
          const Icon = tool.icon;
          const isAvailable = tool.status === "available";
          return (
            <div
              key={tool.id}
              onClick={() => isAvailable && navigate(`/tools/${tool.route}`)}
              role={isAvailable ? "button" : undefined}
              tabIndex={isAvailable ? 0 : undefined}
              onKeyDown={(e) => {
                if (isAvailable && (e.key === "Enter" || e.key === " ")) {
                  e.preventDefault();
                  navigate(`/tools/${tool.route}`);
                }
              }}
              className={`bg-card border border-border rounded-xl p-5 transition-all duration-300 shadow-card stagger-item ${
                isAvailable
                  ? "cursor-pointer hover:shadow-elevated hover:-translate-y-0.5"
                  : "opacity-50 cursor-not-allowed"
              }`}
              style={{ animationDelay: `${i * 60}ms` }}
            >
              <div className="flex items-start justify-between mb-3">
                <div
                  className={`w-10 h-10 ${tool.color} rounded-xl flex items-center justify-center`}
                >
                  <Icon size={20} className="text-white" />
                </div>
                {!isAvailable && (
                  <Badge variant="secondary" className="text-xs">
                    即将推出
                  </Badge>
                )}
              </div>
              <div className="text-sm font-semibold text-foreground mb-1">
                {tool.name}
              </div>
              <div className="text-xs text-muted-foreground leading-relaxed">
                {tool.description}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default ToolsHubPage;
