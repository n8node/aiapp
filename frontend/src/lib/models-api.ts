import { apiFetch } from "@/lib/api";

export type MLModel = {
  id: string;
  slug: string;
  display_name: string;
  purpose: string;
  source_type: string;
  source_ref: string;
  status: string;
  license?: string | null;
  dimensions?: number | null;
  gpu_device?: number | null;
  gateway_alias?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
};

export type StudioStatus = { reachable: boolean; path: string };
export type GatewayStatus = { reachable: boolean; embedding_model: string; dimensions: number };

export function listAdminModels() {
  return apiFetch<{ data: { models: MLModel[] } }>("/admin/models");
}

export function createAdminModel(body: {
  slug: string;
  display_name: string;
  purpose: string;
  source_type: string;
  source_ref: string;
  license?: string;
  notes?: string;
  gpu_device?: number | null;
}) {
  return apiFetch<{ data: MLModel }>("/admin/models", { method: "POST", body: JSON.stringify(body) });
}

export function actionAdminModel(id: string, action: string) {
  return apiFetch<{ data: MLModel }>(`/admin/models/${encodeURIComponent(id)}/action`, {
    method: "POST",
    body: JSON.stringify({ action }),
  });
}

export function fetchModelRuntime() {
  return apiFetch<{ data: { studio: StudioStatus; gateway: GatewayStatus } }>("/admin/models/runtime");
}

export const PURPOSE_LABEL: Record<string, string> = {
  embeddings: "Эмбеддинги",
  chat: "Чат",
  ocr: "OCR",
  rerank: "Реранк",
};

export const STATUS_LABEL: Record<string, string> = {
  quarantine: "Карантин",
  approved: "Одобрена",
  rejected: "Отклонена",
  deployed: "В шлюзе",
  disabled: "Отключена",
};
