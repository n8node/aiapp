"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { Database, FolderOpen, Plus, Settings2, Trash2, Zap } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/layout/EmptyState";
import {
  createKnowledgeBase,
  deleteKnowledgeBase,
  listKnowledgeBases,
  patchKnowledgeBase,
  vectorizeKnowledgeBase,
  type KnowledgeBase,
} from "@/lib/knowledge-api";
import { listAllFolders, listFiles, type WorkspaceFile, type WorkspaceFolder } from "@/lib/files-api";
import { isIngestibleFile } from "@/lib/file-view";
import { ApiError } from "@/lib/api";

function folderPath(folders: WorkspaceFolder[], id: string | null | undefined) {
  if (!id) return "";
  const byId = new Map(folders.map((f) => [f.id, f]));
  const parts: string[] = [];
  let cur: WorkspaceFolder | undefined = byId.get(id);
  const guard = new Set<string>();
  while (cur && !guard.has(cur.id)) {
    guard.add(cur.id);
    parts.unshift(cur.name);
    cur = cur.parent_id ? byId.get(cur.parent_id) : undefined;
  }
  return parts.join(" / ");
}

export function KnowledgeListPage() {
  const [items, setItems] = useState<KnowledgeBase[]>([]);
  const [files, setFiles] = useState<WorkspaceFile[]>([]);
  const [folders, setFolders] = useState<WorkspaceFolder[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [source, setSource] = useState<"folder" | "files">("files");
  const [folderId, setFolderId] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [settings, setSettings] = useState<KnowledgeBase | null>(null);
  const [chunkSize, setChunkSize] = useState(500);
  const [chunkOverlap, setChunkOverlap] = useState(50);
  const [topK, setTopK] = useState(8);
  const [threshold, setThreshold] = useState(0.3);

  async function refresh() {
    const res = await listKnowledgeBases();
    setItems(res.data.knowledge_bases ?? []);
  }

  useEffect(() => {
    void refresh().catch((e) => setError(e instanceof Error ? e.message : "Ошибка загрузки"));
    void listFiles("my-files")
      .then((r) => setFiles((r.files ?? []).filter(isIngestibleFile)))
      .catch(() => undefined);
    void listAllFolders()
      .then((r) => setFolders(r.folders ?? []))
      .catch(() => undefined);
  }, []);

  const inflight = items.some((k) => k.indexed_count < k.file_count && k.file_count > 0);
  useEffect(() => {
    if (!inflight) return;
    const t = setInterval(() => void refresh().catch(() => undefined), 3000);
    return () => clearInterval(t);
  }, [inflight]);

  const folderOptions = useMemo(
    () =>
      folders
        .map((f) => ({ id: f.id, label: folderPath(folders, f.id) || f.name }))
        .sort((a, b) => a.label.localeCompare(b.label, "ru")),
    [folders],
  );

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await createKnowledgeBase({
        name,
        folder_id: source === "folder" && folderId ? folderId : null,
        file_ids: source === "files" ? [...selected] : undefined,
      });
      setName("");
      setSelected(new Set());
      setFolderId("");
      setCreateOpen(false);
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось создать коллекцию");
    } finally {
      setBusy(false);
    }
  }

  async function onVectorize(id: string) {
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

  async function onClear(id: string) {
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

  async function onSaveSettings(e: React.FormEvent) {
    e.preventDefault();
    if (!settings) return;
    setBusy(true);
    try {
      await patchKnowledgeBase(settings.id, {
        chunk_size: chunkSize,
        chunk_overlap: chunkOverlap,
        top_k: topK,
        similarity_threshold: threshold,
      });
      setSettings(null);
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось сохранить настройки");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="RAG-коллекции"
        description="Векторные коллекции пространства: файлы с диска индексируются и подключаются к чату как контекст."
        actions={
          <button
            type="button"
            onClick={() => setCreateOpen(true)}
            className="inline-flex items-center gap-1.5 rounded-lg bg-accent px-3 py-2 text-sm text-white"
          >
            <Plus className="h-4 w-4" /> Создать коллекцию
          </button>
        }
      />
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      {items.length === 0 ? (
        <EmptyState
          title="Пока нет коллекций"
          description="Создайте коллекцию из папки или файлов, затем запустите векторизацию."
          action={
            <button type="button" onClick={() => setCreateOpen(true)} className="rounded-lg bg-accent px-3 py-2 text-sm text-white">
              Создать коллекцию
            </button>
          }
        />
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {items.map((k) => (
            <li key={k.id} className="rounded-xl border border-border bg-surface p-4 shadow-sm">
              <div className="flex items-start justify-between gap-2">
                <Link href={`/rag/${k.id}`} className="min-w-0">
                  <p className="truncate font-medium">{k.name}</p>
                  <p className="mt-1 text-xs text-muted">
                    {k.indexed_count}/{k.file_count} в индексе
                    {k.folder_id ? ` · ${folderPath(folders, k.folder_id) || "папка"}` : ""}
                  </p>
                </Link>
                <Database className="h-4 w-4 shrink-0 text-muted" />
              </div>
              <div className="mt-3 flex flex-wrap gap-1.5">
                <button
                  type="button"
                  disabled={busy || k.file_count === 0}
                  onClick={() => void onVectorize(k.id)}
                  className="inline-flex items-center gap-1 rounded-lg bg-accent px-2.5 py-1.5 text-xs text-white disabled:opacity-50"
                >
                  <Zap className="h-3.5 w-3.5" /> Векторизовать
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setSettings(k);
                    setChunkSize(k.chunk_size || 500);
                    setChunkOverlap(k.chunk_overlap || 50);
                    setTopK(k.top_k || 8);
                    setThreshold(k.similarity_threshold || 0.3);
                  }}
                  className="inline-flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs"
                >
                  <Settings2 className="h-3.5 w-3.5" /> Настройки
                </button>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void onClear(k.id)}
                  className="inline-flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs"
                >
                  Очистить индекс
                </button>
                <button
                  type="button"
                  disabled={busy}
                  onClick={() =>
                    void deleteKnowledgeBase(k.id)
                      .then(() => refresh())
                      .catch((e) => setError(e instanceof Error ? e.message : "Ошибка"))
                  }
                  className="inline-flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs text-red-600"
                >
                  <Trash2 className="h-3.5 w-3.5" /> Удалить
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      {createOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4">
          <form onSubmit={(e) => void onCreate(e)} className="w-full max-w-lg rounded-xl border border-border bg-surface p-5 shadow-sm">
            <p className="text-base font-semibold">Новая коллекция</p>
            <input
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Название"
              className="mt-3 w-full rounded-lg border border-border px-3 py-2 text-sm"
            />
            <div className="mt-3 flex gap-2 text-sm">
              <button
                type="button"
                onClick={() => setSource("folder")}
                className={`rounded-lg border px-3 py-1.5 ${source === "folder" ? "border-accent bg-zinc-100 font-medium" : "border-border"}`}
              >
                Из папки
              </button>
              <button
                type="button"
                onClick={() => setSource("files")}
                className={`rounded-lg border px-3 py-1.5 ${source === "files" ? "border-accent bg-zinc-100 font-medium" : "border-border"}`}
              >
                Из файлов
              </button>
            </div>
            {source === "folder" ? (
              <label className="mt-3 block text-sm">
                <span className="mb-1 flex items-center gap-1 text-muted">
                  <FolderOpen className="h-3.5 w-3.5" /> Папка
                </span>
                <select
                  value={folderId}
                  onChange={(e) => setFolderId(e.target.value)}
                  className="w-full rounded-lg border border-border px-3 py-2 text-sm"
                >
                  <option value="">Выберите папку</option>
                  {folderOptions.map((f) => (
                    <option key={f.id} value={f.id}>
                      {f.label}
                    </option>
                  ))}
                </select>
              </label>
            ) : files.length > 0 ? (
              <ul className="mt-3 max-h-48 space-y-1 overflow-auto text-sm">
                {files.map((f) => (
                  <li key={f.id}>
                    <label className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={selected.has(f.id)}
                        onChange={() => {
                          const next = new Set(selected);
                          if (next.has(f.id)) next.delete(f.id);
                          else next.add(f.id);
                          setSelected(next);
                        }}
                      />
                      {f.name}
                    </label>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="mt-3 text-xs text-muted">Сначала загрузите файлы на диск.</p>
            )}
            <div className="mt-4 flex justify-end gap-2">
              <button type="button" onClick={() => setCreateOpen(false)} className="rounded-lg border border-border px-3 py-2 text-sm">
                Отмена
              </button>
              <button type="submit" disabled={busy} className="rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">
                Создать
              </button>
            </div>
          </form>
        </div>
      ) : null}

      {settings ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4">
          <form onSubmit={(e) => void onSaveSettings(e)} className="w-full max-w-md rounded-xl border border-border bg-surface p-5 shadow-sm">
            <p className="text-base font-semibold">Настройки индекса · {settings.name}</p>
            <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
              <label>
                Размер чанка
                <input type="number" min={64} max={4000} value={chunkSize} onChange={(e) => setChunkSize(Number(e.target.value))} className="mt-1 w-full rounded-lg border border-border px-2 py-1.5" />
              </label>
              <label>
                Перекрытие
                <input type="number" min={0} max={3999} value={chunkOverlap} onChange={(e) => setChunkOverlap(Number(e.target.value))} className="mt-1 w-full rounded-lg border border-border px-2 py-1.5" />
              </label>
              <label>
                Top-K
                <input type="number" min={1} max={50} value={topK} onChange={(e) => setTopK(Number(e.target.value))} className="mt-1 w-full rounded-lg border border-border px-2 py-1.5" />
              </label>
              <label>
                Порог сходства
                <input type="number" min={0} max={1} step={0.05} value={threshold} onChange={(e) => setThreshold(Number(e.target.value))} className="mt-1 w-full rounded-lg border border-border px-2 py-1.5" />
              </label>
            </div>
            <p className="mt-2 text-xs text-muted">После изменения размера чанка запустите векторизацию заново.</p>
            <div className="mt-4 flex justify-end gap-2">
              <button type="button" onClick={() => setSettings(null)} className="rounded-lg border border-border px-3 py-2 text-sm">
                Отмена
              </button>
              <button type="submit" disabled={busy} className="rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">
                Сохранить
              </button>
            </div>
          </form>
        </div>
      ) : null}
    </div>
  );
}
