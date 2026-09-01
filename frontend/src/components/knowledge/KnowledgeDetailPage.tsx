"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import {
  KB_FILE_STATUS,
  deleteKnowledgeBase,
  getKnowledgeBase,
  patchKnowledgeBase,
  searchKnowledgeBase,
  vectorizeKnowledgeBase,
  type KnowledgeBase,
  type KnowledgeHit,
} from "@/lib/knowledge-api";
import { listFiles, type WorkspaceFile } from "@/lib/files-api";
import { isIngestibleFile } from "@/lib/file-view";
import { ApiError } from "@/lib/api";

export function KnowledgeDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const [kb, setKb] = useState<KnowledgeBase | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<KnowledgeHit[]>([]);
  const [busy, setBusy] = useState(false);
  const [files, setFiles] = useState<WorkspaceFile[]>([]);
  const [addIds, setAddIds] = useState<Set<string>>(new Set());

  async function refresh() {
    const res = await getKnowledgeBase(id);
    setKb(res.data);
  }

  useEffect(() => {
    void refresh().catch((e) => setError(e instanceof Error ? e.message : "Не найдено"));
    void listFiles("my-files")
      .then((r) => setFiles((r.files ?? []).filter(isIngestibleFile)))
      .catch(() => undefined);
  }, [id]);

  const attached = new Set((kb?.files ?? []).map((f) => f.file_id));
  const inflight =
    (kb?.files ?? []).some((f) => f.status === "pending") ||
    kb?.latest_job?.status === "queued" ||
    kb?.latest_job?.status === "running";

  useEffect(() => {
    if (!inflight) return;
    const t = setInterval(() => void refresh().catch(() => undefined), 2500);
    return () => clearInterval(t);
  }, [inflight, id]);

  async function onVectorize() {
    setBusy(true);
    setError(null);
    try {
      await vectorizeKnowledgeBase(id);
      await refresh();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось запустить векторизацию");
    } finally {
      setBusy(false);
    }
  }

  async function onSearch(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await searchKnowledgeBase(id, query, kb?.top_k ?? 10, kb?.similarity_threshold ?? 0.3);
      setHits(res.data.hits ?? []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Поиск не удался");
    } finally {
      setBusy(false);
    }
  }

  async function onAdd() {
    setBusy(true);
    try {
      await patchKnowledgeBase(id, { file_ids: [...addIds] });
      setAddIds(new Set());
      await refresh();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось добавить файлы");
    } finally {
      setBusy(false);
    }
  }

  async function onRemove(fileId: string) {
    setBusy(true);
    try {
      await patchKnowledgeBase(id, { remove_file_ids: [fileId] });
      await refresh();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось удалить файл");
    } finally {
      setBusy(false);
    }
  }

  async function onClear() {
    setBusy(true);
    try {
      await patchKnowledgeBase(id, { clear_vectors: true });
      await refresh();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось очистить индекс");
    } finally {
      setBusy(false);
    }
  }

  if (!kb) {
    return (
      <div>
        <PageHeader
          title="RAG-коллекция"
          crumbs={[
            { label: "Главная", href: "/dashboard" },
            { label: "RAG-коллекции", href: "/rag" },
            { label: "Коллекция" },
          ]}
        />
        <p className="text-sm text-muted">{error || "Загрузка…"}</p>
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title={kb.name}
        crumbs={[
          { label: "Главная", href: "/dashboard" },
          { label: "RAG-коллекции", href: "/rag" },
          { label: kb.name },
        ]}
        actions={
          <button
            type="button"
            onClick={() =>
              void deleteKnowledgeBase(id)
                .then(() => router.push("/rag"))
                .catch((e) => setError(e instanceof Error ? e.message : "Ошибка"))
            }
            className="rounded-lg border border-border px-3 py-2 text-sm text-red-600"
          >
            Удалить
          </button>
        }
      />
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <div className="mb-4 flex flex-wrap gap-2">
        <button type="button" disabled={busy} onClick={() => void onVectorize()} className="rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">
          Векторизовать
        </button>
        <button type="button" disabled={busy} onClick={() => void onClear()} className="rounded-lg border border-border px-3 py-2 text-sm">
          Очистить индекс
        </button>
        <span className="self-center text-sm text-muted">
          {kb.indexed_count} из {kb.file_count} файлов в индексе
          {kb.latest_job ? ` · ${kb.latest_job.status === "running" || kb.latest_job.status === "queued" ? "идёт векторизация" : kb.latest_job.status}` : ""}
        </span>
      </div>

      <div className="mb-4 rounded-xl border border-border bg-surface p-4 shadow-sm">
        <h2 className="text-sm font-semibold">Файлы</h2>
        <ul className="mt-2 divide-y divide-border text-sm">
          {(kb.files ?? []).map((f) => (
            <li key={f.file_id} className="flex items-center justify-between gap-2 py-2">
              <span className="min-w-0 truncate">{f.name || f.file_id}</span>
              <span className="flex items-center gap-2">
                <span className="text-muted">{KB_FILE_STATUS[f.status] ?? f.status}</span>
                <button type="button" disabled={busy} onClick={() => void onRemove(f.file_id)} className="text-xs text-red-600">
                  Убрать
                </button>
              </span>
            </li>
          ))}
        </ul>
        <div className="mt-3">
          <p className="text-xs text-muted">Добавить с диска (корень)</p>
          <ul className="mt-1 max-h-32 overflow-auto">
            {files
              .filter((f) => !attached.has(f.id))
              .map((f) => (
                <li key={f.id}>
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={addIds.has(f.id)}
                      onChange={() => {
                        const next = new Set(addIds);
                        if (next.has(f.id)) next.delete(f.id);
                        else next.add(f.id);
                        setAddIds(next);
                      }}
                    />
                    {f.name}
                  </label>
                </li>
              ))}
          </ul>
          {addIds.size > 0 ? (
            <button type="button" disabled={busy} onClick={() => void onAdd()} className="mt-2 rounded-lg border border-border px-3 py-1.5 text-sm">
              Добавить выбранные
            </button>
          ) : null}
        </div>
      </div>

      <form onSubmit={(e) => void onSearch(e)} className="rounded-xl border border-border bg-surface p-4 shadow-sm">
        <h2 className="text-sm font-semibold">Проверка поиска</h2>
        <div className="mt-2 flex gap-2">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="flex-1 rounded-lg border border-border px-3 py-2 text-sm"
            placeholder="Запрос"
          />
          <button type="submit" disabled={busy} className="rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">
            Искать
          </button>
        </div>
        <ul className="mt-3 space-y-2">
          {hits.map((h, i) => (
            <li key={`${h.file_id}-${h.chunk_index}-${i}`} className="rounded-lg border border-border p-3 text-sm">
              <p className="text-xs text-muted">
                {h.file_name} · сходство {(h.similarity * 100).toFixed(0)}%
              </p>
              <p className="mt-1 whitespace-pre-wrap">{h.content}</p>
            </li>
          ))}
        </ul>
      </form>
    </div>
  );
}
