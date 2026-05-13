import {
  ApiError,
  CalculateResponseSchema,
  ErrorResponseSchema,
  PackSetSchema,
  type CalculateResponse,
  type PackSet,
} from "./types";

const API_BASE = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

interface CallResult<T> {
  data: T;
  etag: string;
}

async function call<T>(
  path: string,
  init: RequestInit,
  schema: { parse: (x: unknown) => T },
): Promise<CallResult<T>> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...(init.headers ?? {}),
      },
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : "network error";
    throw new ApiError(0, "network", msg, "");
  }

  const raw = await res.text();
  let parsed: unknown = null;
  if (raw) {
    try {
      parsed = JSON.parse(raw);
    } catch {
      throw new ApiError(res.status, "decode", `non-JSON response: ${raw.slice(0, 80)}`, "");
    }
  }

  if (!res.ok) {
    const env = ErrorResponseSchema.safeParse(parsed);
    if (env.success) {
      throw new ApiError(res.status, env.data.code, env.data.message, env.data.request_id);
    }
    throw new ApiError(res.status, "unknown", `HTTP ${res.status}`, "");
  }

  return {
    data: schema.parse(parsed),
    etag: res.headers.get("etag") ?? "",
  };
}

export async function getPacks(): Promise<{ data: PackSet; etag: string }> {
  const { data, etag } = await call("/api/v1/packs", { method: "GET" }, PackSetSchema);
  return { data, etag: etag || data.version };
}

export async function putPacks(
  sizes: number[],
  ifMatch?: string,
): Promise<{ data: PackSet; etag: string }> {
  const headers: Record<string, string> = {};
  if (ifMatch) headers["If-Match"] = ifMatch;
  const { data, etag } = await call(
    "/api/v1/packs",
    { method: "PUT", body: JSON.stringify({ sizes }), headers },
    PackSetSchema,
  );
  return { data, etag: etag || data.version };
}

export async function calculate(order: number): Promise<CalculateResponse> {
  const { data } = await call(
    "/api/v1/calculate",
    { method: "POST", body: JSON.stringify({ order }) },
    CalculateResponseSchema,
  );
  return data;
}

export { API_BASE };
