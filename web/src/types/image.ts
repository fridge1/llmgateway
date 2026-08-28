export interface ImageTask {
  id: number;
  user_id: string;
  type: 'generate' | 'edit';
  status: 'pending' | 'processing' | 'completed' | 'failed';
  model: string;
  prompt: string;
  size: string;
  image_count: number;
  result_urls: string[] | null;
  cost: number;
  error_message: string;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
}

export type ImageTaskParams = Record<string, string | number>;

export interface SubmitTaskRequest {
  model: string;
  prompt: string;
  size: string;
  n: number;
  params?: ImageTaskParams;
}

export interface SubmitTaskResponse {
  id: number;
  status: string;
}

export interface EditTaskRequest {
  model: string;
  prompt: string;
  size: string;
  n: number;
  images: File[];
  mask?: File;
  params?: ImageTaskParams;
}
