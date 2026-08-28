import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { PptTask, PresentationData, SubmitPptTaskRequest, SubmitPptTaskResponse } from '@/types/ppt';
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api-client';

export function usePptTasks(limit = 20, offset = 0) {
  return useQuery<PptTask[]>({
    queryKey: ['ppt-tasks', limit, offset],
    queryFn: () => apiGet<PptTask[]>(`/api/ppt/tasks?limit=${limit}&offset=${offset}`),
    refetchInterval: (query) => {
      const tasks = query.state.data;
      const hasActive = tasks?.some(t => t.status === 'pending' || t.status === 'processing');
      return hasActive ? 3000 : false;
    },
  });
}

export function usePptTask(taskId: number | null) {
  return useQuery<PptTask>({
    queryKey: ['ppt-task', taskId],
    queryFn: () => apiGet<PptTask>(`/api/ppt/tasks/${taskId}`),
    enabled: taskId !== null,
    refetchInterval: (query) => {
      const task = query.state.data;
      if (task?.status === 'pending' || task?.status === 'processing') return 2000;
      return false;
    },
  });
}

export function useSubmitPptTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: SubmitPptTaskRequest) =>
      apiPost<SubmitPptTaskResponse>('/api/ppt/tasks', req),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ppt-tasks'] }),
  });
}

export function useConfirmOutline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: number) =>
      apiPost<{ status: string }>(`/api/ppt/tasks/${taskId}/confirm`, {}),
    onSuccess: (_data, taskId) => {
      qc.invalidateQueries({ queryKey: ['ppt-task', taskId] });
      qc.invalidateQueries({ queryKey: ['ppt-tasks'] });
    },
  });
}

export function useSavePresentation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ taskId, presentation }: { taskId: number; presentation: PresentationData }) =>
      apiPut<{ status: string }>(`/api/ppt/tasks/${taskId}/presentation`, presentation),
    onSuccess: (_data, { taskId }) => {
      qc.invalidateQueries({ queryKey: ['ppt-task', taskId] });
    },
  });
}

export function useDeletePptTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: number) =>
      apiDelete<void>(`/api/ppt/tasks/${taskId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['ppt-tasks'] });
    },
  });
}
