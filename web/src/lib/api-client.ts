/**
 * Lightweight fetch wrapper for the LLM Gateway API.
 *
 * - All requests include credentials (HttpOnly JWT cookie).
 * - POST/PUT sends JSON body.
 * - Non-ok responses are parsed and thrown as ApiError.
 */

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

async function request<T>(
  method: string,
  url: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(url, {
    method,
    headers,
    credentials: "include",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let message = res.statusText;
    let code = "unknown";
    try {
      const errBody = await res.json();
      if (errBody?.error) {
        message = errBody.error.message || message;
        code = errBody.error.code || code;
      }
    } catch {
      // body wasn't JSON — keep statusText
    }
    throw new ApiError(res.status, message, code);
  }

  // Handle 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

export function apiGet<T>(url: string): Promise<T> {
  return request<T>("GET", url);
}

export function apiPost<T>(url: string, body?: unknown): Promise<T> {
  return request<T>("POST", url, body);
}

export function apiPut<T>(url: string, body?: unknown): Promise<T> {
  return request<T>("PUT", url, body);
}

export function apiPatch<T>(url: string, body?: unknown): Promise<T> {
  return request<T>("PATCH", url, body);
}

export function apiDelete<T>(url: string): Promise<T> {
  return request<T>("DELETE", url);
}

export async function apiPostFormData<T>(url: string, formData: FormData): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    credentials: "include",
    body: formData,
  });

  if (!res.ok) {
    let message = res.statusText;
    let code = "unknown";
    try {
      const errBody = await res.json();
      if (errBody?.error) {
        message = errBody.error.message || message;
        code = errBody.error.code || code;
      }
    } catch {
      // body wasn't JSON
    }
    throw new ApiError(res.status, message, code);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}
