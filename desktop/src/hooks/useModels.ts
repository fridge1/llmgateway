import { useState, useEffect } from "react";
import { api } from "../lib/tauri";
import type { Model } from "../lib/types";

export function useModels() {
  const [models, setModels] = useState<Model[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.listModels()
      .then(data => setModels(data.data || []))
      .catch(err => console.error("获取模型失败:", err))
      .finally(() => setLoading(false));
  }, []);

  return { models, loading };
}
