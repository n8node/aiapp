import { apiPath } from "./urls";

export class ApiError extends Error {
  code: string;
  status: number;
  constructor(message: string, code: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

export type User = {
  id: string;
  email: string;
  name: string;
  is_platform_admin: boolean;
  is_blocked: boolean;
  totp_enabled: boolean;
  created_at: string;
};

export type Workspace = {
  id: string;
  name: string;
  slug: string;
  role?: string;
};

export type MeData = {
  user: User;
  workspace: Workspace | null;
  workspaces: Workspace[];
};

export type Invite = {
  id: string;
  code_prefix: string;
  status: string;
  code?: string;
  created_at: string;
  used_at?: string | null;
};

async function parse(res: Response) {
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const err = body.error ?? {};
    throw new ApiError(
      err.message || "Ошибка запроса",
      err.code || "error",
      res.status,
    );
  }
  return body;
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(apiPath(path), {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });
  return parse(res);
}

export function login(email: string, password: string, totpCode?: string) {
  return apiFetch<{ data: MeData }>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password, totp_code: totpCode ?? "" }),
  });
}

export function register(email: string, password: string, name: string, inviteCode: string) {
  return apiFetch<{ data: MeData }>("/auth/register", {
    method: "POST",
    body: JSON.stringify({ email, password, name, invite_code: inviteCode }),
  });
}

export function logout() {
  return apiFetch("/auth/logout", { method: "POST" });
}

export function fetchMe() {
  return apiFetch<{ data: MeData }>("/auth/me");
}

export function verifyInvite(inviteCode: string) {
  return apiFetch("/auth/invite/verify", {
    method: "POST",
    body: JSON.stringify({ invite_code: inviteCode }),
  });
}

export function setupTotp() {
  return apiFetch<{ data: { secret: string; otpauth_url: string } }>("/auth/2fa/setup", {
    method: "POST",
  });
}

export function confirmTotp(code: string) {
  return apiFetch("/auth/2fa/confirm", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
}

export function fetchAdminUsers(query: { q?: string; is_blocked?: boolean; is_platform_admin?: boolean } = {}) {
  const params = new URLSearchParams();
  if (query.q) params.set("q", query.q);
  if (query.is_blocked !== undefined) params.set("is_blocked", String(query.is_blocked));
  if (query.is_platform_admin !== undefined) {
    params.set("is_platform_admin", String(query.is_platform_admin));
  }
  const qs = params.toString();
  return apiFetch<{ data: { users: User[]; total: number } }>(
    `/admin/users${qs ? `?${qs}` : ""}`,
  );
}

export function setAdminUserBlocked(userID: string, isBlocked: boolean) {
  return apiFetch(`/admin/users/${userID}/block`, {
    method: "POST",
    body: JSON.stringify({ is_blocked: isBlocked }),
  });
}

export function fetchAdminInvites() {
  return apiFetch<{ data: { invites: Invite[]; total: number } }>("/admin/invites");
}

export function issueAdminInvites(count: number) {
  return apiFetch<{ data: { invites: Invite[] } }>("/admin/invites", {
    method: "POST",
    body: JSON.stringify({ count }),
  });
}

export function revokeAdminInvite(id: string) {
  return apiFetch(`/admin/invites/${id}/revoke`, { method: "POST" });
}

export function fetchAuthDomains() {
  return apiFetch<{ data: { domains: string[] } }>("/admin/auth-domains");
}

export function saveAuthDomains(domains: string[]) {
  return apiFetch<{ data: { domains: string[] } }>("/admin/auth-domains", {
    method: "PUT",
    body: JSON.stringify({ domains }),
  });
}
