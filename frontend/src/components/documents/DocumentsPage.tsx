"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/layout/EmptyState";
import {
  documentInFlight,
  documentStatusLabel,
  listDocuments,
  type DocumentCard,
} from "@/lib/documents-api";
import { cn } from "@/lib/cn";

function statusClass(status?: string | null) {
  switch (status) {
    case "published":
      return "text-emerald-700";
    case "awaiting_review":
      return "text-amber-700";
    case "failed":
    case "rejected":
      return "text-red-600";
    default:
      return "text-muted";
  }
}

export function DocumentsPage() {
  const [items, setItems] = useState<DocumentCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    const res = await listDocuments();
    setItems(res.documents ?? []);
  }

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    refresh()
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : "Не удалось загрузить документы");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const inflight = items.some((d) => documentInFlight(d.current_version?.status));
  useEffect(() => {
    if (!inflight) return;
    const t = setInterval(() => {
      void refresh().catch(() => undefined);
    }, 2500);
    return () => clearInterval(t);
  }, [inflight]);

  return (
    <div>
      <PageHeader
        title="Документы"
        description="Карточки документов с версией, хешем и статусом проверки. Индексация для чата появится позже."
      />
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}
      {loading ? <p className="text-sm text-muted">Загрузка…</p> : null}
      {!loading && items.length === 0 ? (
        <EmptyState
          title="Пока нет документов"
          description="Откройте файл на диске и нажмите «Отправить в документы». После извлечения текста карточка появится здесь на проверке."
          action={
            <Link href="/files" className="inline-flex rounded-lg bg-accent px-3 py-2 text-sm text-white">
              Перейти к файлам
            </Link>
          }
        />
      ) : null}
      {items.length > 0 ? (
        <div className="overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
          <ul className="divide-y divide-border">
            {items.map((doc) => {
              const v = doc.current_version;
              return (
                <li key={doc.id}>
                  <Link
                    href={`/documents/${doc.id}`}
                    className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 hover:bg-zinc-50"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{doc.title}</p>
                      <p className="mt-0.5 truncate text-xs text-muted">
                        {v?.original_name ?? "—"}
                        {v?.version_n ? ` · версия ${v.version_n}` : ""}
                      </p>
                    </div>
                    <span className={cn("text-sm", statusClass(v?.status))}>{documentStatusLabel(v?.status)}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
