import { ApiError, apiFetch } from "@/lib/api";
import { apiPath } from "@/lib/urls";

export type WorkspaceFile = {
  id: string;
  workspace_id: string;
  folder_id: string | null;
  name: string;
  mime_type: string;
  size: number;
  deleted_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type WorkspaceFolder = {
  id: string;
  workspace_id: string;
  parent_id: string | null;
  name: string;
  files_count?: number;
  deleted_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type FolderBreadcrumb = {
  id: string | null;
  name: string;
};

export type FilesSection = "my-files" | "recent" | "photos" | "videos" | "trash";

export type UploadJob = {
  id: string;
  name: string;
  size: number;
  progress: number;
  status: "pending" | "uploading" | "completed" | "error" | "cancelled";
  error?: string;
};

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 Б";
  const units = ["Б", "КБ", "МБ", "ГБ", "ТБ"];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i += 1;
  }
  return `${n >= 10 || i === 0 ? n.toFixed(i === 0 ? 0 : 1) : n.toFixed(1)} ${units[i]}`;
}

async function diskData<T>(path: string, init?: RequestInit): Promise<T> {
  const body = await apiFetch<{ data: T }>(path, init);
  return body.data;
}

export function listFiles(section: FilesSection, folderId?: string | null) {
  const params = new URLSearchParams({ section });
  if (folderId) params.set("folder_id", folderId);
  return diskData<{ files: WorkspaceFile[] }>(`/disk/files?${params}`);
}

export function getFile(fileId: string) {
  return diskData<WorkspaceFile>(`/disk/files/${encodeURIComponent(fileId)}`);
}

export function listFolders(parentId?: string | null) {
  const params = new URLSearchParams();
  if (parentId) params.set("parent_id", parentId);
  const q = params.toString();
  return diskData<{ folders: WorkspaceFolder[] }>(`/disk/folders${q ? `?${q}` : ""}`);
}

export function listAllFolders() {
  return diskData<{ folders: WorkspaceFolder[] }>("/disk/folders?scope=all");
}

export function fetchFolderBreadcrumbs(folderId: string) {
  return diskData<{ breadcrumbs: FolderBreadcrumb[] }>(
    `/disk/folders/${encodeURIComponent(folderId)}/breadcrumbs`,
  );
}

export function initUpload(input: {
  name: string;
  size: number;
  mime_type: string;
  folder_id?: string | null;
}) {
  return diskData<{
    upload_url: string;
    upload_headers: Record<string, string>;
    upload_session_token: string;
  }>("/disk/files/upload/init", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function completeUpload(uploadSessionToken: string) {
  return diskData<WorkspaceFile>("/disk/files/upload/complete", {
    method: "POST",
    body: JSON.stringify({ upload_session_token: uploadSessionToken }),
  });
}

export function putWithProgress(
  url: string,
  file: File,
  headers: Record<string, string>,
  onProgress: (pct: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", url);
    for (const [key, value] of Object.entries(headers)) {
      if (isUnsafeUploadHeader(key)) continue;
      try {
        xhr.setRequestHeader(key, value);
      } catch {
        /* browsers reject Host / Content-Length */
      }
    }
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(Math.round((e.loaded / e.total) * 100));
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) resolve();
      else reject(new Error(uploadFailureMessage(xhr.status)));
    };
    xhr.onerror = () => reject(new Error(uploadFailureMessage(0)));
    xhr.onabort = () => reject(new DOMException("Загрузка отменена", "AbortError"));
    const onAbort = () => xhr.abort();
    if (signal) {
      if (signal.aborted) {
        xhr.abort();
        return;
      }
      signal.addEventListener("abort", onAbort, { once: true });
    }
    xhr.send(file);
  });
}

function isUnsafeUploadHeader(name: string) {
  return /^(host|content-length|connection|cookie|cookie2|origin|referer|date|via|keep-alive|te|trailer|transfer-encoding|upgrade|dnt|expect)$/i.test(
    name,
  );
}

function uploadFailureMessage(status: number) {
  if (status === 0) {
    return "Браузер не смог отправить файл в S3. Обычно это CORS: в админке нажмите «Проверка соединения».";
  }
  if (status === 403) {
    return "S3 отклонил загрузку (нет прав или неверная подпись).";
  }
  return `Не удалось загрузить файл в хранилище (${status})`;
}

export async function uploadFile(
  file: File,
  folderId: string | null,
  onProgress: (pct: number) => void,
  signal?: AbortSignal,
) {
  const mimeType = file.type || "application/octet-stream";
  const init = await initUpload({
    name: file.name,
    size: file.size,
    mime_type: mimeType,
    folder_id: folderId,
  });
  await putWithProgress(init.upload_url, file, init.upload_headers ?? {}, onProgress, signal);
  return completeUpload(init.upload_session_token);
}

export function createFolder(name: string, parentId?: string | null) {
  return diskData<WorkspaceFolder>("/disk/folders", {
    method: "POST",
    body: JSON.stringify({ name, parent_id: parentId ?? null }),
  });
}

export function renameFile(id: string, name: string) {
  return diskData<WorkspaceFile>(`/disk/files/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ name }),
  });
}

export function renameFolder(id: string, name: string) {
  return diskData<WorkspaceFolder>(`/disk/folders/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ name }),
  });
}

export function copyFile(id: string, folderId: string | null) {
  return diskData<WorkspaceFile>(`/disk/files/${id}/copy`, {
    method: "POST",
    body: JSON.stringify({ folder_id: folderId }),
  });
}

export function deleteFile(id: string) {
  return diskData<{ ok: boolean }>(`/disk/files/${id}`, { method: "DELETE" });
}

export function deleteFolder(id: string) {
  return diskData<{ ok: boolean }>(`/disk/folders/${id}`, { method: "DELETE" });
}

export function fileContentURL(id: string, disposition: "inline" | "attachment" = "attachment") {
  const params = new URLSearchParams();
  params.set("disposition", disposition);
  return apiPath(`/disk/files/${encodeURIComponent(id)}/download?${params}`);
}

export function openFile(id: string, disposition: "inline" | "attachment" = "attachment") {
  window.open(fileContentURL(id, disposition), "_blank", "noopener,noreferrer");
}

export function bulkFiles(ids: string[], action: "delete" | "move" | "copy", folderId?: string | null) {
  return diskData<{ ok: number; errors: { id: string; message: string }[] }>("/disk/files/bulk", {
    method: "POST",
    body: JSON.stringify({ ids, action, folder_id: folderId ?? null }),
  });
}

export function bulkFolders(ids: string[], action: "delete" | "move", folderId?: string | null) {
  return diskData<{ ok: number; errors: { id: string; message: string }[] }>("/disk/folders/bulk", {
    method: "POST",
    body: JSON.stringify({ ids, action, folder_id: folderId ?? null }),
  });
}

export function listTrash() {
  return diskData<{ files: WorkspaceFile[]; folders: WorkspaceFolder[] }>("/disk/trash");
}

export function restoreTrash(fileIds: string[], folderIds: string[]) {
  return diskData<{ ok: boolean }>("/disk/trash/restore", {
    method: "POST",
    body: JSON.stringify({ file_ids: fileIds, folder_ids: folderIds }),
  });
}

export function emptyTrash() {
  return diskData<{ ok: boolean }>("/disk/trash/empty", { method: "POST" });
}

export function permanentDeleteTrash(id: string, kind: "file" | "folder") {
  return diskData<{ ok: boolean }>(`/disk/trash/${id}?kind=${kind}`, { method: "DELETE" });
}

export function isDiskApiError(e: unknown, code: string) {
  return e instanceof ApiError && e.code === code;
}
