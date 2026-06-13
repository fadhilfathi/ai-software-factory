"use client";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "/api";

type RequestOptions = {
  params?: Record<string, string | number | boolean | undefined>;
  headers?: Record<string, string>;
  body?: unknown;
  method?: "GET" | "POST" | "PATCH" | "DELETE" | "PUT";
};

async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { params, headers, body, method = "GET" } = options;

  let url = `${API_BASE}${path}`;
  if (params) {
    const search = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value === undefined || value === null) continue;
      // Booleans, numbers, and strings all go on the wire as strings.
      if (typeof value === "boolean") {
        search.set(key, value ? "true" : "false");
      } else {
        search.set(key, String(value));
      }
    }
    const query = search.toString();
    if (query) url += `?${query}`;
  }

  const finalHeaders: Record<string, string> = {
    Accept: "application/json",
    ...headers,
  };
  // Inject the bearer token if one is set in module memory. The
  // AuthProvider calls `setAccessToken(...)` on login / logout.
  const token = getAccessToken();
  if (token) finalHeaders.Authorization = `Bearer ${token}`;
  if (body && !finalHeaders["Content-Type"] && !finalHeaders["content-type"]) {
    finalHeaders["Content-Type"] = "application/json";
  }

  const init: RequestInit = { method, headers: finalHeaders };
  if (body && method !== "GET") {
    init.body = typeof body === "string" ? body : JSON.stringify(body);
  }

  const res = await fetch(url, init);

  if (!res.ok) {
    let errBody: unknown = null;
    try {
      errBody = await res.json();
    } catch {
      // ignore — non-JSON error
    }
    const message =
      (typeof errBody === "object" && errBody && "message" in errBody
        ? String((errBody as { message?: unknown }).message)
        : null) ?? res.statusText ?? `Request failed with ${res.status}`;
    const err = new Error(message) as Error & {
      status?: number;
      body?: unknown;
    };
    err.status = res.status;
    err.body = errBody;
    throw err;
  }

  if (res.status === 204) return undefined as T;

  const contentType = res.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return (await res.json()) as T;
  }
  return (await res.text()) as unknown as T;
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "GET" }),

  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, {
      ...options,
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    }),

  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, {
      ...options,
      method: "PATCH",
      body: body ? JSON.stringify(body) : undefined,
    }),

  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, {
      ...options,
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    }),

  delete: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "DELETE" }),
};

/**
 * In-memory access token. The AuthProvider sets this on login and
 * clears it on logout; the request helper reads it on every fetch.
 * Kept as a module-level variable rather than a context so we can
 * use it in client-side fetch wrappers without plumbing through
 * React.
 */
let accessToken: string | null = null;

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export function getAccessToken(): string | null {
  return accessToken;
}
