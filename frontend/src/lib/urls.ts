export const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "/app/api/v1";

export function apiPath(path: string): string {
  const suffix = path.startsWith("/") ? path : `/${path}`;
  return `${API_BASE}${suffix}`;
}

export const HEALTH_PATH = "/app/health";
