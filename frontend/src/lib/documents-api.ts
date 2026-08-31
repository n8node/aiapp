import { apiFetch } from "@/lib/api";

export type DocumentStatus =
  | "queued"
  | "running"
  | "awaiting_review"
  | "published"
  | "failed"
  | "rejected";

export type DocumentVersion = {
  id: string;
  document_id: string;
  workspace_id: string;
  version_n: number;
  disk_file_id?: string | null;
  original_name: string;
  mime_type: string;
  size: number;
  content_hash: string;
  status: DocumentStatus;
  engine?: string | null;
  page_count?: number | null;
  char_count?: number | null;
  confidence?: number | null;
  warnings: string[];
  extracted_text?: string | null;
  error_code?: string | null;
  reviewed_by_user_id?: string | null;
  reviewed_at?: string | null;
  review_note?: string | null;
  created_at: string;
  updated_at: string;
};

export type DocumentCard = {
  id: string;
  workspace_id: string;
  title: string;
  created_by_user_id?: string | null;
  current_version_id?: string | null;
  created_at: string;
  updated_at: string;
  current_version?: DocumentVersion | null;
};

export type DocumentFromFileResult = {
  document: DocumentCard;
  job_id?: string | null;
  created: boolean;
};

const STATUS_LABEL: Record<DocumentStatus, string> = {
  queued: "В очереди",
  running: "Обработка",
  awaiting_review: "На проверке",
  published: "Опубликован",
  failed: "Ошибка",
  rejected: "Отклонён",
};

export function documentStatusLabel(status?: string | null): string {
  if (status && status in STATUS_LABEL) return STATUS_LABEL[status as DocumentStatus];
  return status || "—";
}

export function documentInFlight(status?: string | null): boolean {
  return status === "queued" || status === "running";
}

async function docData<T>(path: string, init?: RequestInit): Promise<T> {
  const body = await apiFetch<{ data: T }>(path, init);
  return body.data;
}

export function listDocuments() {
  return docData<{ documents: DocumentCard[] }>("/documents");
}

export function getDocument(id: string) {
  return docData<DocumentCard>(`/documents/${encodeURIComponent(id)}`);
}

export function createDocumentFromFile(fileId: string) {
  return docData<DocumentFromFileResult>("/documents/from-file", {
    method: "POST",
    body: JSON.stringify({ file_id: fileId }),
  });
}

export function reviewDocument(documentId: string, versionId: string, action: "approve" | "reject", note?: string) {
  return docData<DocumentCard>(
    `/documents/${encodeURIComponent(documentId)}/versions/${encodeURIComponent(versionId)}/review`,
    {
      method: "POST",
      body: JSON.stringify({ action, note: note || undefined }),
    },
  );
}

export function retryDocument(documentId: string, versionId: string) {
  return docData<DocumentFromFileResult>(
    `/documents/${encodeURIComponent(documentId)}/versions/${encodeURIComponent(versionId)}/retry`,
    { method: "POST" },
  );
}
