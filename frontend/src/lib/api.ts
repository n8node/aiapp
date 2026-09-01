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
  studio_access?: boolean;
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
  ui_locale?: string;
};

export type Invite = {
  id: string;
  code_prefix: string;
  status: string;
  code?: string;
  created_at: string;
  used_at?: string | null;
  used_by_email?: string | null;
};

export type InviteListQuery = {
  status?: string;
  created_from?: string;
  created_to?: string;
  used_from?: string;
  used_to?: string;
  used_email?: string;
  limit?: number;
  offset?: number;
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

export function switchWorkspace(workspaceID: string) {
  return apiFetch<{ data: { workspace: Workspace } }>("/auth/workspace", {
    method: "PUT",
    body: JSON.stringify({ workspace_id: workspaceID }),
  });
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

export function fetchAdminInvites(query: InviteListQuery = {}) {
  const params = new URLSearchParams();
  if (query.status) params.set("status", query.status);
  if (query.created_from) params.set("created_from", query.created_from);
  if (query.created_to) params.set("created_to", query.created_to);
  if (query.used_from) params.set("used_from", query.used_from);
  if (query.used_to) params.set("used_to", query.used_to);
  if (query.used_email) params.set("used_email", query.used_email);
  if (query.limit) params.set("limit", String(query.limit));
  if (query.offset) params.set("offset", String(query.offset));
  const qs = params.toString();
  return apiFetch<{ data: { invites: Invite[]; total: number } }>(
    `/admin/invites${qs ? `?${qs}` : ""}`,
  );
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

export function deleteAdminInvites(ids: string[]) {
  return apiFetch<{ data: { deleted: number } }>("/admin/invites/delete", {
    method: "POST",
    body: JSON.stringify({ ids }),
  });
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

export type BitrixStatus = {
  configured: boolean;
  portal_host?: string;
  webhook_masked?: string;
  last_sync_at?: string | null;
  last_sync_status?: string;
  last_sync_error?: string;
  departments_count: number;
  users_count: number;
};

export type BitrixDepartment = {
  bitrix_id: number;
  parent_bitrix_id?: number | null;
  name: string;
  sort: number;
  head_bitrix_id?: number | null;
  workspace_id?: string | null;
  workspace_name?: string | null;
  workspace_inherited?: boolean;
  inherited_from?: string;
  include_descendants?: boolean | null;
};

export type WorkspaceBitrixLink = {
  workspace_id: string;
  workspace_name?: string;
  bitrix_department_id: number;
  department_name?: string;
  include_descendants: boolean;
};

export type AdminWorkspace = {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  created_at: string;
  member_count: number;
  bitrix_departments: WorkspaceBitrixLink[];
};

export type MembershipApplyResult = {
  workspaces: number;
  matched: number;
  added: number;
  updated: number;
  removed: number;
  unmatched: number;
};

export type BitrixUser = {
  bitrix_id: number;
  email: string;
  name: string;
  last_name: string;
  active: boolean;
  department_ids: number[];
};

export function fetchBitrixStatus() {
  return apiFetch<{ data: BitrixStatus }>("/admin/bitrix");
}

export function saveBitrixWebhook(webhookURL: string) {
  return apiFetch<{ data: BitrixStatus }>("/admin/bitrix", {
    method: "PUT",
    body: JSON.stringify({ webhook_url: webhookURL }),
  });
}

export function disconnectBitrix() {
  return apiFetch<{ data: BitrixStatus }>("/admin/bitrix", {
    method: "PUT",
    body: JSON.stringify({ disconnect: true }),
  });
}

export function testBitrix() {
  return apiFetch<{ data: { ok: boolean } }>("/admin/bitrix/test", { method: "POST" });
}

export function syncBitrix() {
  return apiFetch<{
    data: { departments: number; users: number; memberships?: MembershipApplyResult };
  }>("/admin/bitrix/sync", {
    method: "POST",
  });
}

export function fetchBitrixDepartments() {
  return apiFetch<{ data: { departments: BitrixDepartment[] } }>("/admin/bitrix/departments");
}

export function fetchBitrixUsers(q = "") {
  const qs = q.trim() ? `?q=${encodeURIComponent(q.trim())}` : "";
  return apiFetch<{ data: { users: BitrixUser[] } }>(`/admin/bitrix/users${qs}`);
}

export function fetchAdminWorkspaces() {
  return apiFetch<{ data: { workspaces: AdminWorkspace[] } }>("/admin/workspaces");
}

export function createWorkspaceFromDepartment(bitrixDepartmentID: number, includeDescendants = true, name = "") {
  return apiFetch<{ data: AdminWorkspace }>("/admin/workspaces", {
    method: "POST",
    body: JSON.stringify({
      bitrix_department_id: bitrixDepartmentID,
      include_descendants: includeDescendants,
      name,
    }),
  });
}

export function applyWorkspaceMemberships() {
  return apiFetch<{ data: MembershipApplyResult }>("/admin/workspaces/apply", { method: "POST" });
}

export function unlinkWorkspaceDepartment(workspaceID: string, deptID: number) {
  return apiFetch<{ data: { ok: boolean } }>(`/admin/workspaces/${workspaceID}/bitrix/${deptID}`, {
    method: "DELETE",
  });
}

export function linkWorkspaceDepartment(workspaceID: string, deptID: number, includeDescendants: boolean) {
  return apiFetch<{ data: AdminWorkspace }>(`/admin/workspaces/${workspaceID}/bitrix`, {
    method: "POST",
    body: JSON.stringify({
      bitrix_department_id: deptID,
      include_descendants: includeDescendants,
    }),
  });
}

export type StorageAdminView = {
  endpoint: string;
  bucket: string;
  project_id?: string;
  region: string;
  access_key: string;
  secret_key_set: boolean;
  secret_key_hint?: string;
  use_ssl: boolean;
  path_style: boolean;
  enabled: boolean;
  cors_origins: string[];
  cors_xml: string;
  updated_at?: string;
};

export type StorageAdminUpdateRequest = {
  endpoint: string;
  bucket: string;
  project_id: string;
  region: string;
  access_key: string;
  secret_key?: string;
  use_ssl: boolean;
  path_style: boolean;
  enabled: boolean;
};

export type StorageTestResult = {
  ok: boolean;
  message: string;
};

export function fetchAdminStorageSettings() {
  return apiFetch<{ data: StorageAdminView }>("/admin/storage-settings");
}

export function updateAdminStorageSettings(payload: StorageAdminUpdateRequest) {
  return apiFetch<{ data: StorageAdminView }>("/admin/storage-settings", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function testAdminStorageConnection() {
  return apiFetch<{ data: StorageTestResult }>("/admin/storage-settings/test", {
    method: "POST",
  });
}

export type OutboundProxySettings = {
  proxy_enabled: boolean;
  proxy_active_url: string;
  proxy_urls: string[];
  active_masked?: string;
};

export type OutboundProxyTestResult = {
  ok: boolean;
  message: string;
  hub: string;
  cdn: string;
};

export function fetchAdminOutboundProxy() {
  return apiFetch<{ data: OutboundProxySettings }>("/admin/outbound-proxy");
}

export function saveAdminOutboundProxy(payload: {
  proxy_enabled: boolean;
  proxy_active_url: string;
  proxy_urls: string[];
}) {
  return apiFetch<{ data: OutboundProxySettings }>("/admin/outbound-proxy", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function testAdminOutboundProxy() {
  return apiFetch<{ data: OutboundProxyTestResult }>("/admin/outbound-proxy/test", {
    method: "POST",
  });
}

export function setAdminUserStudioAccess(userID: string, studioAccess: boolean) {
  return apiFetch(`/admin/users/${userID}/studio`, {
    method: "POST",
    body: JSON.stringify({ studio_access: studioAccess }),
  });
}

export type UILocaleOption = { code: string; label: string };

export function fetchAdminUILocale() {
  return apiFetch<{ data: { locale: string; locales: UILocaleOption[] } }>("/admin/ui-locale");
}

export function saveAdminUILocale(locale: string) {
  return apiFetch<{ data: { locale: string; locales: UILocaleOption[] } }>("/admin/ui-locale", {
    method: "PUT",
    body: JSON.stringify({ locale }),
  });
}

export type TrainingRequest = {
  id: string;
  workspace_id: string;
  requested_by_user_id: string;
  requester_email?: string;
  title: string;
  purpose: string;
  dataset_note: string;
  base_model: string;
  status: string;
  review_note?: string;
  created_at: string;
  updated_at: string;
};

export function fetchTrainingRequests() {
  return apiFetch<{ data: { requests: TrainingRequest[]; total: number } }>("/training-requests");
}

export function createTrainingRequest(payload: {
  title: string;
  purpose: string;
  dataset_note: string;
  base_model: string;
}) {
  return apiFetch<{ data: TrainingRequest }>("/training-requests", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function submitTrainingRequest(id: string) {
  return apiFetch<{ data: TrainingRequest }>(`/training-requests/${id}/submit`, { method: "POST" });
}

export function cancelTrainingRequest(id: string) {
  return apiFetch<{ data: TrainingRequest }>(`/training-requests/${id}/cancel`, { method: "POST" });
}

export function fetchAdminTrainingRequests(status?: string) {
  const qs = status ? `?status=${encodeURIComponent(status)}` : "";
  return apiFetch<{ data: { requests: TrainingRequest[]; total: number } }>(
    `/admin/training-requests${qs}`,
  );
}

export function approveAdminTrainingRequest(id: string, note = "") {
  return apiFetch<{ data: TrainingRequest }>(`/admin/training-requests/${id}/approve`, {
    method: "POST",
    body: JSON.stringify({ note }),
  });
}

export function rejectAdminTrainingRequest(id: string, note = "") {
  return apiFetch<{ data: TrainingRequest }>(`/admin/training-requests/${id}/reject`, {
    method: "POST",
    body: JSON.stringify({ note }),
  });
}
