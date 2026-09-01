import { apiFetch } from "@/lib/api";
import { apiPath } from "@/lib/urls";

export type CatalogModel = {
  id: string;
  slug: string;
  display_name: string;
  purpose: string;
};

export type ChatCitation = {
  source: string;
  file_name?: string;
  content?: string;
  similarity?: number;
  url?: string;
};

export type ChatMessage = {
  id: string;
  thread_id: string;
  role: string;
  content: string;
  citations?: ChatCitation[];
  media_url?: string | null;
  media_kind?: string | null;
  created_at: string;
};

export type ChatThread = {
  id: string;
  workspace_id: string;
  user_id: string;
  model_id?: string | null;
  title: string;
  web_search: boolean;
  collection_ids: string[];
  created_at: string;
  updated_at: string;
  model?: CatalogModel | null;
};

export function listCatalogModels() {
  return apiFetch<{ data: { models: CatalogModel[] } }>("/models");
}

export function listChatThreads() {
  return apiFetch<{ data: { threads: ChatThread[] } }>("/chats");
}

export function createChatThread(body: { model_id?: string; collection_ids?: string[]; web_search?: boolean }) {
  return apiFetch<{ data: ChatThread }>("/chats", { method: "POST", body: JSON.stringify(body) });
}

export function getChatThread(id: string) {
  return apiFetch<{ data: { thread: ChatThread; messages: ChatMessage[] } }>(`/chats/${encodeURIComponent(id)}`);
}

export function patchChatThread(
  id: string,
  body: { model_id?: string; collection_ids?: string[]; web_search?: boolean; title?: string },
) {
  return apiFetch<{ data: ChatThread }>(`/chats/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export function deleteChatThread(id: string) {
  return apiFetch<{ data: { ok: boolean } }>(`/chats/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function generateChatMedia(id: string, kind: "image" | "video", prompt: string) {
  return apiFetch<{ data: ChatMessage }>(`/chats/${encodeURIComponent(id)}/media`, {
    method: "POST",
    body: JSON.stringify({ kind, prompt }),
  });
}

export type ChatStreamEvent = {
  type: "delta" | "done" | "error";
  text?: string;
  message?: ChatMessage;
  citations?: ChatCitation[];
};

export async function streamChatMessage(
  threadId: string,
  content: string,
  onEvent: (ev: ChatStreamEvent) => void,
): Promise<void> {
  const res = await fetch(apiPath(`/chats/${encodeURIComponent(threadId)}/messages`), {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const err = body.error ?? {};
    throw new Error(err.message || "Не удалось отправить сообщение");
  }
  if (!res.body) {
    throw new Error("Поток ответа недоступен");
  }
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    for (;;) {
      const idx = buf.indexOf("\n\n");
      if (idx < 0) break;
      const block = buf.slice(0, idx);
      buf = buf.slice(idx + 2);
      for (const line of block.split("\n")) {
        const trimmed = line.trim();
        if (!trimmed.startsWith("data:")) continue;
        const payload = trimmed.slice(5).trim();
        if (!payload) continue;
        try {
          onEvent(JSON.parse(payload) as ChatStreamEvent);
        } catch {
          /* ignore partial json */
        }
      }
    }
  }
}
