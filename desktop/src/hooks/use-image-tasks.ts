import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPostFormData, apiDelete } from "@/lib/api-client";
import { queryKeys } from "@/lib/query-keys";
import type {
  ImageTask,
  SubmitTaskRequest,
  SubmitTaskResponse,
  EditTaskRequest,
} from "@/lib/types-api";

export function useImageTasks(limit = 20, offset = 0) {
  return useQuery<ImageTask[]>({
    queryKey: queryKeys.image.tasks(limit, offset),
    queryFn: () => apiGet<ImageTask[]>(`/api/image/tasks?limit=${limit}&offset=${offset}`),
    refetchInterval: (query) => {
      const tasks = query.state.data;
      if (!tasks) return false;
      const active = tasks.filter((t) => t.status === "pending" || t.status === "processing");
      if (active.length === 0) return false;
      // 任务刚提交的高频窗口（< 8s 内）轮询 800ms，让首张图近实时点亮
      const now = Date.now();
      const recentlyCreated = active.some((t) => now - new Date(t.created_at).getTime() < 8000);
      return recentlyCreated ? 800 : 1500;
    },
  });
}

export function useSubmitTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: SubmitTaskRequest) =>
      apiPost<SubmitTaskResponse>("/api/image/tasks", req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["image-tasks"] }),
  });
}

export function useSubmitEditTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: EditTaskRequest) => {
      const fd = new FormData();
      fd.append("model", req.model);
      fd.append("prompt", req.prompt);
      fd.append("size", req.size);
      fd.append("n", req.n.toString());
      if (req.params && Object.keys(req.params).length > 0) {
        fd.append("params", JSON.stringify(req.params));
      }
      for (const img of req.images) fd.append("image[]", img);
      if (req.mask) fd.append("mask", req.mask);
      return apiPostFormData<SubmitTaskResponse>("/api/image/tasks/edit", fd);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["image-tasks"] }),
  });
}

export function useDeleteImageTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: number) =>
      apiDelete<{ success: boolean }>(`/api/image/tasks/${taskId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["image-tasks"] }),
  });
}

export function useDeleteImageFromTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ taskId, url }: { taskId: number; url: string }) =>
      apiDelete<{ success: boolean; remaining: number }>(
        `/api/image/tasks/${taskId}/images?url=${encodeURIComponent(url)}`,
      ),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["image-tasks"] }),
  });
}

// useDeleteImageTasksBatch deletes multiple tasks concurrently, tolerating
// individual failures. Returns the ids that failed to delete.
export function useDeleteImageTasksBatch() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (taskIds: number[]) => {
      const results = await Promise.allSettled(
        taskIds.map((id) => apiDelete<{ success: boolean }>(`/api/image/tasks/${id}`)),
      );
      const failed = taskIds.filter((_, i) => results[i].status === "rejected");
      return { failed };
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["image-tasks"] }),
  });
}
