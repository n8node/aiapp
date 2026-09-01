import { apiFetch } from "@/lib/api";

export type KnowledgeFile = {
  file_id: string;
  name?: string;
  content_hash: string;
  chunk_count: number;
  status: string;
  error_code?: string | null;
};

export type VectorizeJob = {
  id: string;
  knowledge_base_id: string;
  status: string;
  processed: number;
  total: number;
  last_error?: string | null;
};

export type KnowledgeBase = {
  id: string;
  workspace_id: string;
  name: string;
  chunk_size: number;
  chunk_overlap: number;
  folder_id?: string | null;
  similarity_threshold: number;
  top_k: number;
  file_count: number;
  indexed_count: number;
  created_at: string;
  updated_at: string;
  files?: KnowledgeFile[];
  latest_job?: VectorizeJob | null;
};

export type KnowledgeHit = {
  file_id?: string | null;
  file_name?: string;
  chunk_index: number;
  content: string;
  similarity: number;
};

export type KnowledgePatch = {
  name?: string;
  file_ids?: string[];
  remove_file_ids?: string[];
  folder_id?: string | null;
  chunk_size?: number;
  chunk_overlap?: number;
  similarity_threshold?: number;
  top_k?: number;
  clear_vectors?: boolean;
};

export function listKnowledgeBases() {
  return apiFetch<{ data: { knowledge_bases: KnowledgeBase[] } }>("/knowledge-bases");
}

export function createKnowledgeBase(body: { name: string; file_ids?: string[]; folder_id?: string | null }) {
  return apiFetch<{ data: KnowledgeBase }>("/knowledge-bases", { method: "POST", body: JSON.stringify(body) });
}

export function getKnowledgeBase(id: string) {
  return apiFetch<{ data: KnowledgeBase }>(`/knowledge-bases/${encodeURIComponent(id)}`);
}

export function patchKnowledgeBase(id: string, body: KnowledgePatch) {
  return apiFetch<{ data: KnowledgeBase }>(`/knowledge-bases/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export function deleteKnowledgeBase(id: string) {
  return apiFetch<{ data: { ok: boolean } }>(`/knowledge-bases/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function vectorizeKnowledgeBase(id: string) {
  return apiFetch<{ data: { knowledge_base: KnowledgeBase; job_id: string } }>(
    `/knowledge-bases/${encodeURIComponent(id)}/vectorize`,
    { method: "POST" },
  );
}

export function searchKnowledgeBase(id: string, query: string, limit = 10, threshold = 0.3) {
  return apiFetch<{ data: { hits: KnowledgeHit[] } }>(`/knowledge-bases/${encodeURIComponent(id)}/search`, {
    method: "POST",
    body: JSON.stringify({ query, limit, threshold }),
  });
}

export const KB_FILE_STATUS: Record<string, string> = {
  pending: "Ожидает",
  indexed: "В индексе",
  failed: "Ошибка",
  skipped: "Пропущен",
};
