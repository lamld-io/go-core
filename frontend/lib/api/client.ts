import { getAppEnv } from "@/lib/env";

import type { ApiEnvelope, HealthResponse } from "./types";

export class ApiClientError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code?: string,
  ) {
    super(message);
    this.name = "ApiClientError";
  }
}

type RequestOptions = RequestInit & {
  baseUrl?: string;
  fetchImpl?: typeof fetch;
};

function isApiEnvelope(value: unknown): value is ApiEnvelope<unknown> {
  if (!value || typeof value !== "object") {
    return false;
  }

  const candidate = value as Record<string, unknown>;
  return typeof candidate.code === "string" && typeof candidate.message === "string";
}

export async function parseApiResponse<T>(response: Response): Promise<T | undefined> {
  if (response.status === 204) {
    return undefined;
  }

  const payload = (await response.json()) as unknown;

  if (!isApiEnvelope(payload)) {
    throw new ApiClientError("Auth API returned an unexpected response envelope.", response.status);
  }

  if (!response.ok) {
    throw new ApiClientError(payload.message, response.status, payload.code);
  }

  return payload.data as T;
}

export function createApiUrl(path: string, baseUrl = getAppEnv().authApiBaseUrl): string {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return `${baseUrl}${normalizedPath}`;
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T | undefined> {
  const { baseUrl, fetchImpl = fetch, headers, ...init } = options;
  const response = await fetchImpl(createApiUrl(path, baseUrl), {
    ...init,
    headers: {
      Accept: "application/json",
      ...headers,
    },
    cache: init.cache ?? "no-store",
  });

  return parseApiResponse<T>(response);
}

export async function checkAuthServiceHealth(options: {
  baseUrl?: string;
  fetchImpl?: typeof fetch;
} = {}): Promise<
  | { ok: true; data: HealthResponse }
  | { ok: false; message: string }
> {
  const { baseUrl, fetchImpl = fetch } = options;

  try {
    const response = await fetchImpl(createApiUrl("/health", baseUrl), {
      cache: "no-store",
      headers: { Accept: "application/json" },
    });

    if (!response.ok) {
      return { ok: false, message: `Health probe failed with status ${response.status}.` };
    }

    const payload = (await response.json()) as HealthResponse;

    if (typeof payload?.status !== "string" || typeof payload?.service !== "string") {
      return { ok: false, message: "Health probe returned an unexpected payload." };
    }

    return { ok: true, data: payload };
  } catch (error) {
    return {
      ok: false,
      message:
        error instanceof Error
          ? error.message
          : "Health probe failed before reaching the auth service.",
    };
  }
}
