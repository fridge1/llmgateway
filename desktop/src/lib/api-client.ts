import { invoke } from "@tauri-apps/api/core";

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, message: string, code: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

function parseError(err: unknown): ApiError {
  const msg = String(err);
  if (msg.includes("登录已过期") || msg.includes("Unauthorized")) {
    return new ApiError(401, "登录已过期", "unauthorized");
  }
  return new ApiError(500, msg, "unknown");
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  auth: boolean = true,
): Promise<T> {
  try {
    const command = auth ? "api_request" : "api_request_unauth";
    const result = await invoke<T>(command, {
      method,
      path,
      body: body !== undefined ? body : null,
    });
    return result as T;
  } catch (err) {
    throw parseError(err);
  }
}

export function apiGet<T>(path: string): Promise<T> {
  return request<T>("GET", path);
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>("POST", path, body);
}

export function apiPut<T>(path: string, body?: unknown): Promise<T> {
  return request<T>("PUT", path, body);
}

export function apiDelete<T>(path: string): Promise<T> {
  return request<T>("DELETE", path);
}

export function apiGetUnauth<T>(path: string): Promise<T> {
  return request<T>("GET", path, undefined, false);
}

export function apiPostUnauth<T>(path: string, body?: unknown): Promise<T> {
  return request<T>("POST", path, body, false);
}

/**
 * Multipart upload through the Rust IPC bridge. Mirrors the web's
 * `apiPostFormData`: pass a FormData, and File entries are read into byte
 * arrays and forwarded to the gateway as a real multipart/form-data request.
 */
export async function apiPostFormData<T>(path: string, formData: FormData): Promise<T> {
  const textFields: [string, string][] = [];
  const files: { field: string; filename: string; mime: string; data: number[] }[] = [];

  for (const [key, value] of formData.entries()) {
    if (value instanceof File) {
      const buf = new Uint8Array(await value.arrayBuffer());
      files.push({
        field: key,
        filename: value.name || "upload",
        mime: value.type || "application/octet-stream",
        data: Array.from(buf),
      });
    } else {
      textFields.push([key, String(value)]);
    }
  }

  try {
    return await invoke<T>("api_request_multipart", {
      path,
      textFields,
      files,
    });
  } catch (err) {
    throw parseError(err);
  }
}
