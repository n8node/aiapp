"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/layout/EmptyState";
import { createKnowledgeBase, listKnowledgeBases, type KnowledgeBase } from "@/lib/knowledge-api";
import { listFiles, type WorkspaceFile } from "@/lib/files-api";
import { isIngestibleFile } from "@/lib/file-view";
import { ApiError } from "@/lib/api";

export function KnowledgeListPage() {
  const [items, setItems] = useState<KnowledgeBase[]>([]);
  const [files, setFiles] = useState<WorkspaceFile[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function refresh() {
    const res = await listKnowledgeBases();
    setItems(res.data.knowledge_bases ?? []);
  }

  useEffect(() => {
    void refresh().catch((e) => setError(e instanceof Error ? e.message : "Ошибка загрузки"));
    void listFiles("my-files")
      .then((r) => setFiles((r.files ?? []).filter(isIngestibleFile)))
      .catch(() => undefined);
  }, []);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await createKnowledgeBase({ name, file_ids: [...selected] });
      setName("");
      setSelected(new Set());
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось создать коллекцию");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Базы знаний"
        description="RAG-коллекции пространства: файлы с диска превращаются в векторы через внутренний шлюз моделей."
      />
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <form onSubmit={(e) => void onCreate(e)} className="mb-6 rounded-xl border border-border bg-surface p-4 shadow-sm">
        <p className="text-sm font-semibold">Новая коллекция</p>
        <div className="mt-3 flex flex-wrap gap-2">
          <input
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Название"
            className="min-w-[12rem] flex-1 rounded-lg border border-border px-3 py-2 text-sm"
          />
          <button type="submit" disabled={busy} className="rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">
            Создать
          </button>
        </div>
        {files.length > 0 ? (
          <ul className="mt-3 max-h-40 space-y-1 overflow-auto text-sm">
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
          <p className="mt-2 text-xs text-muted">Файлы можно добавить позже. Сначала загрузите их на диск.</p>
        )}
      </form>

      {items.length === 0 ? (
        <EmptyState title="Пока нет коллекций" description="Создайте базу знаний и отправьте файлы на векторизацию." />
      ) : (
        <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
          {items.map((k) => (
            <li key={k.id}>
              <Link href={`/knowledge/${k.id}`} className="flex items-center justify-between px-4 py-3 hover:bg-zinc-50">
                <span className="font-medium">{k.name}</span>
                <span className="text-sm text-muted">
                  {k.indexed_count}/{k.file_count} в индексе
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
