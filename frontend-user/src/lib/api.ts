const base = "";

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(base + path, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
  });
  const body = await res.json();
  if (!body.ok) {
    const code = body.error?.code || "ERROR";
    const msg = body.error?.message || res.statusText;
    throw new Error(`${code}: ${msg}`);
  }
  return body.data as T;
}

export const api = {
  health: () => req<{ status: string; time: string; profile: string }>("/api/health"),
  get: (key: string) => req<{ key: string; value: string }>(`/api/kv/${encodeURIComponent(key)}`),
  put: (key: string, value: string) =>
    req(`/api/kv/${encodeURIComponent(key)}`, { method: "PUT", body: JSON.stringify({ value }) }),
  del: (key: string) => req(`/api/kv/${encodeURIComponent(key)}`, { method: "DELETE" }),
  scan: (start: string, end: string, limit: number) =>
    req<{ items: { key: string; value: string }[]; count: number }>(
      `/api/scan?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}&limit=${limit}`,
    ),
  state: () => req("/api/lsm/state"),
  metrics: () => req<Record<string, unknown>>("/api/metrics"),
  flush: () => req("/api/admin/flush", { method: "POST" }),
  compact: () => req("/api/admin/compact", { method: "POST" }),
  profile: (profile: string, sync?: boolean) =>
    req("/api/admin/profile", { method: "POST", body: JSON.stringify({ profile, sync }) }),
  benchStart: (workers: number, qps: number, value_size: number) =>
    req("/api/bench/start", { method: "POST", body: JSON.stringify({ workers, qps, value_size }) }),
  benchStop: () => req("/api/bench/stop", { method: "POST" }),
};

export function wsURL(): string {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  return `${proto}://${location.host}/ws/events`;
}

export function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}
