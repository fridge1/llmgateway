import { useQuery } from "@tanstack/react-query";
import { Cpu } from "lucide-react";
import { apiGet } from "@/lib/api-client";

interface ModelInfo {
  id: string;
  object: string;
  owned_by: string;
}

interface ModelsResponse {
  data: ModelInfo[];
}

const SubUserModelsPage = () => {
  const { data, isLoading } = useQuery({
    queryKey: ["sub-user-models"],
    queryFn: () => apiGet<ModelsResponse>("/api/v1/models"),
  });

  const models = data?.data ?? [];

  return (
    <div>
      <h1 className="text-2xl font-bold text-foreground mb-6">可用模型</h1>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      ) : models.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <Cpu size={48} className="mx-auto mb-4 opacity-30" />
          <p>暂无可用模型</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {models.map((model) => (
            <div key={model.id} className="bg-card border border-border/60 rounded-xl p-4 hover:shadow-card transition-shadow">
              <div className="flex items-start gap-3">
                <div className="w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <Cpu size={16} className="text-primary" />
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-medium text-foreground truncate">{model.id}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">{model.owned_by}</p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default SubUserModelsPage;
