"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Globe, ImageIcon, Plus, Send, Trash2, Video } from "lucide-react";
import { EmptyState } from "@/components/layout/EmptyState";
import {
  createChatThread,
  deleteChatThread,
  generateChatMedia,
  getChatThread,
  listCatalogModels,
  listChatThreads,
  patchChatThread,
  streamChatMessage,
  type CatalogModel,
  type ChatMessage,
  type ChatThread,
} from "@/lib/chat-api";
import { listKnowledgeBases, type KnowledgeBase } from "@/lib/knowledge-api";
import { ApiError } from "@/lib/api";
import { cn } from "@/lib/cn";

export function ChatPage() {
  const [threads, setThreads] = useState<ChatThread[]>([]);
  const [models, setModels] = useState<CatalogModel[]>([]);
  const [collections, setCollections] = useState<KnowledgeBase[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [thread, setThread] = useState<ChatThread | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [streaming, setStreaming] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  const chatModels = models.filter((m) => m.purpose === "chat");
  const hasImage = models.some((m) => m.purpose === "image");
  const hasVideo = models.some((m) => m.purpose === "video");

  const loadLists = useCallback(async () => {
    const [t, m, k] = await Promise.all([listChatThreads(), listCatalogModels(), listKnowledgeBases()]);
    setThreads(t.data.threads ?? []);
    setModels(m.data.models ?? []);
    setCollections(k.data.knowledge_bases ?? []);
  }, []);

  useEffect(() => {
    void loadLists().catch((e) => setError(e instanceof Error ? e.message : "Ошибка загрузки"));
  }, [loadLists]);

  async function openThread(id: string) {
    setError(null);
    const res = await getChatThread(id);
    setActiveId(id);
    setThread(res.data.thread);
    setMessages(res.data.messages ?? []);
    setStreaming("");
  }

  async function onNew() {
    setBusy(true);
    setError(null);
    try {
      const modelId = chatModels[0]?.id;
      const created = await createChatThread({ model_id: modelId, collection_ids: [], web_search: false });
      await loadLists();
      await openThread(created.data.id);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось создать чат");
    } finally {
      setBusy(false);
    }
  }

  async function onSend(e?: React.FormEvent) {
    e?.preventDefault();
    const text = draft.trim();
    if (!text || !activeId || busy) return;
    setDraft("");
    setBusy(true);
    setError(null);
    setMessages((prev) => [
      ...prev,
      {
        id: "tmp-user",
        thread_id: activeId,
        role: "user",
        content: text,
        created_at: new Date().toISOString(),
      },
    ]);
    setStreaming("");
    try {
      await streamChatMessage(activeId, text, (ev) => {
        if (ev.type === "delta" && ev.text) {
          setStreaming((s) => s + ev.text);
        }
        if (ev.type === "done" && ev.message) {
          setStreaming("");
          setMessages((prev) => [...prev, ev.message!]);
        }
        if (ev.type === "error") {
          setError("Модель не ответила. Проверьте, что чат-модель развёрнута.");
        }
      });
      await loadLists();
      if (thread) {
        const res = await getChatThread(activeId);
        setThread(res.data.thread);
        setMessages(res.data.messages ?? []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отправить сообщение");
    } finally {
      setBusy(false);
      setStreaming("");
    }
  }

  async function onMedia(kind: "image" | "video") {
    if (!activeId || !draft.trim()) {
      setError("Введите описание для генерации");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const msg = await generateChatMedia(activeId, kind, draft.trim());
      setDraft("");
      setMessages((prev) => [...prev, msg.data]);
      await loadLists();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Генерация недоступна");
    } finally {
      setBusy(false);
    }
  }

  async function patchActive(body: Parameters<typeof patchChatThread>[1]) {
    if (!activeId) return;
    try {
      const res = await patchChatThread(activeId, body);
      setThread(res.data);
      await loadLists();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось сохранить настройки чата");
    }
  }

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streaming]);

  const selectedCollections = useMemo(() => new Set(thread?.collection_ids ?? []), [thread]);

  return (
    <div className="-mx-4 -my-6 flex h-[calc(100vh-0px)] min-h-[32rem] sm:-mx-6 lg:-mx-8">
      <aside className="flex w-60 shrink-0 flex-col border-r border-border bg-surface">
        <div className="flex items-center justify-between border-b border-border px-3 py-3">
          <p className="text-sm font-semibold">Чаты</p>
          <button type="button" disabled={busy} onClick={() => void onNew()} className="rounded-md p-1.5 hover:bg-zinc-100" aria-label="Новый чат">
            <Plus className="h-4 w-4" />
          </button>
        </div>
        <ul className="flex-1 overflow-auto p-2">
          {threads.map((t) => (
            <li key={t.id}>
              <button
                type="button"
                onClick={() => void openThread(t.id)}
                className={cn(
                  "flex w-full items-center justify-between gap-1 rounded-md px-2 py-2 text-left text-sm",
                  activeId === t.id ? "bg-zinc-100 font-medium" : "text-muted hover:bg-zinc-50 hover:text-text",
                )}
              >
                <span className="truncate">{t.title || "Новый чат"}</span>
                <span
                  role="button"
                  tabIndex={0}
                  className="rounded p-1 hover:bg-zinc-200"
                  onClick={(ev) => {
                    ev.stopPropagation();
                    void deleteChatThread(t.id).then(async () => {
                      if (activeId === t.id) {
                        setActiveId(null);
                        setThread(null);
                        setMessages([]);
                      }
                      await loadLists();
                    });
                  }}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </span>
              </button>
            </li>
          ))}
        </ul>
      </aside>
      <section className="flex min-w-0 flex-1 flex-col bg-bg">
        {error ? <p className="border-b border-border bg-surface px-4 py-2 text-sm text-red-600">{error}</p> : null}
        {!thread ? (
          <div className="flex flex-1 items-center justify-center p-6">
            <EmptyState
              title="Новый диалог"
              description="Каждый чат — отдельный контекст. Без RAG-коллекций модель отвечает своими знаниями и историей этой беседы."
              action={
                <button type="button" onClick={() => void onNew()} className="rounded-lg bg-accent px-3 py-2 text-sm text-white">
                  Начать чат
                </button>
              }
            />
          </div>
        ) : (
          <>
            <div className="flex flex-wrap items-center gap-2 border-b border-border bg-surface px-4 py-3">
              <select
                value={thread.model_id ?? ""}
                onChange={(e) => void patchActive({ model_id: e.target.value })}
                className="rounded-lg border border-border px-2 py-1.5 text-sm"
              >
                {chatModels.length === 0 ? <option value="">Нет развёрнутых моделей</option> : null}
                {chatModels.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.display_name}
                  </option>
                ))}
              </select>
              <label className="flex items-center gap-1.5 text-sm">
                <Globe className="h-4 w-4 text-muted" />
                Интернет
                <input
                  type="checkbox"
                  checked={thread.web_search}
                  onChange={(e) => void patchActive({ web_search: e.target.checked })}
                />
              </label>
              <details className="relative">
                <summary className="cursor-pointer list-none rounded-lg border border-border px-2 py-1.5 text-sm">
                  Коллекции ({selectedCollections.size})
                </summary>
                <div className="absolute z-10 mt-1 max-h-56 w-64 overflow-auto rounded-lg border border-border bg-surface p-2 shadow-sm">
                  {collections.length === 0 ? (
                    <p className="px-1 py-1 text-xs text-muted">Сначала создайте RAG-коллекцию</p>
                  ) : (
                    collections.map((c) => (
                      <label key={c.id} className="flex items-center gap-2 py-1 text-sm">
                        <input
                          type="checkbox"
                          checked={selectedCollections.has(c.id)}
                          onChange={() => {
                            const next = new Set(selectedCollections);
                            if (next.has(c.id)) next.delete(c.id);
                            else next.add(c.id);
                            void patchActive({ collection_ids: [...next] });
                          }}
                        />
                        <span className="truncate">{c.name}</span>
                      </label>
                    ))
                  )}
                </div>
              </details>
            </div>
            <div className="flex-1 space-y-3 overflow-auto px-4 py-4">
              {messages.map((m) => (
                <article
                  key={m.id}
                  className={cn(
                    "max-w-3xl rounded-xl border border-border px-3 py-2 text-sm shadow-sm",
                    m.role === "user" ? "ml-auto bg-zinc-100" : "bg-surface",
                  )}
                >
                  {m.content ? <p className="whitespace-pre-wrap">{m.content}</p> : null}
                  {m.media_url && m.media_kind === "image" ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={m.media_url} alt="" className="mt-2 max-h-80 rounded-lg" />
                  ) : null}
                  {m.media_url && m.media_kind === "video" ? (
                    <video src={m.media_url} controls className="mt-2 max-h-80 rounded-lg" />
                  ) : null}
                  {(m.citations ?? []).length > 0 ? (
                    <ul className="mt-2 space-y-1 border-t border-border pt-2 text-xs text-muted">
                      {(m.citations ?? []).map((c, i) => (
                        <li key={`${c.source}-${i}`}>
                          {c.source === "web" && c.url ? (
                            <a href={c.url} className="text-accent hover:underline" target="_blank" rel="noreferrer">
                              {c.file_name || c.url}
                            </a>
                          ) : (
                            <span>
                              {c.file_name}
                              {c.similarity != null ? ` · ${(c.similarity * 100).toFixed(0)}%` : ""}
                            </span>
                          )}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </article>
              ))}
              {streaming ? (
                <article className="max-w-3xl rounded-xl border border-border bg-surface px-3 py-2 text-sm shadow-sm">
                  <p className="whitespace-pre-wrap">{streaming}</p>
                </article>
              ) : null}
              <div ref={bottomRef} />
            </div>
            <form onSubmit={(e) => void onSend(e)} className="border-t border-border bg-surface p-3">
              <div className="flex flex-wrap gap-2">
                <textarea
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  rows={2}
                  placeholder="Сообщение…"
                  className="min-w-0 flex-1 rounded-lg border border-border px-3 py-2 text-sm"
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      void onSend();
                    }
                  }}
                />
                <div className="flex flex-col gap-1">
                  <button type="submit" disabled={busy || !draft.trim()} className="inline-flex items-center justify-center gap-1 rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">
                    <Send className="h-4 w-4" /> Отправить
                  </button>
                  {hasImage ? (
                    <button type="button" disabled={busy} onClick={() => void onMedia("image")} className="inline-flex items-center justify-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs">
                      <ImageIcon className="h-3.5 w-3.5" /> Картинка
                    </button>
                  ) : null}
                  {hasVideo ? (
                    <button type="button" disabled={busy} onClick={() => void onMedia("video")} className="inline-flex items-center justify-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs">
                      <Video className="h-3.5 w-3.5" /> Видео
                    </button>
                  ) : null}
                </div>
              </div>
            </form>
          </>
        )}
      </section>
    </div>
  );
}
