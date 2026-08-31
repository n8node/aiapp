"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import {
  documentInFlight,
  documentStatusLabel,
  getDocument,
  reviewDocument,
  retryDocument,
  type DocumentCard,
} from "@/lib/documents-api";
import { formatBytes } from "@/lib/files-api";
import { cn } from "@/lib/cn";

const WARNING_LABEL: Record<string, string> = {
  ocr_required: "Нужно распознавание (OCR) — текст пока пустой",
  unsupported_type: "Этот формат пока извлекается ограниченно",
  empty_text: "Текст не найден",
  page_limit: "Обработана только часть страниц",
  sheet_limit: "Обработана только часть листов",
  row_limit: "Обработана только часть строк",
  page_error: "Часть страниц не удалось прочитать",
  decode_failed: "Не удалось прочитать кодировку",
  rtf_unparsed: "RTF разобран как обычный текст",
};

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

const ERROR_LABEL: Record<string, string> = {
  extract_unavailable: "Сервис извлечения недоступен",
  extract_timeout: "Извлечение превысило время ожидания",
  extract_failed: "Не удалось извлечь текст",
  object_too_large: "Файл слишком большой",
};

export function DocumentReviewPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const [doc, setDoc] = useState<DocumentCard | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");

  async function refresh() {
    const next = await getDocument(id);
    setDoc(next);
  }

  useEffect(() => {
    let cancelled = false;
    refresh().catch((e) => {
      if (!cancelled) setError(e instanceof Error ? e.message : "Документ не найден");
    });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const status = doc?.current_version?.status;
  useEffect(() => {
    if (!documentInFlight(status)) return;
    const t = setInterval(() => {
      void refresh().catch(() => undefined);
    }, 2000);
    return () => clearInterval(t);
  }, [status, id]);

  const v = doc?.current_version;
  const canReview = v?.status === "awaiting_review";
  const canRetry = v?.status === "failed" || v?.status === "rejected";

  async function onReview(action: "approve" | "reject") {
    if (!doc || !v) return;
    setBusy(true);
    setError(null);
    try {
      const next = await reviewDocument(doc.id, v.id, action, note);
      setDoc(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Не удалось сохранить решение");
    } finally {
      setBusy(false);
    }
  }

  async function onRetry() {
    if (!doc || !v) return;
    setBusy(true);
    setError(null);
    try {
      const next = await retryDocument(doc.id, v.id);
      setDoc(next.document);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Не удалось повторить обработку");
    } finally {
      setBusy(false);
    }
  }

  if (error && !doc) {
    return (
      <div>
        <PageHeader title="Документ" crumbs={[{ label: "Главная", href: "/dashboard" }, { label: "Документы", href: "/documents" }, { label: "Документ" }]} />
        <p className="text-sm text-red-600">{error}</p>
      </div>
    );
  }

  if (!doc || !v) {
    return (
      <div>
        <PageHeader title="Документ" crumbs={[{ label: "Главная", href: "/dashboard" }, { label: "Документы", href: "/documents" }, { label: "Документ" }]} />
        <p className="text-sm text-muted">Загрузка…</p>
      </div>
    );
  }

  return (
    <div>
      <PageHeader
        title={doc.title}
        crumbs={[
          { label: "Главная", href: "/dashboard" },
          { label: "Документы", href: "/documents" },
          { label: doc.title },
        ]}
        description="Проверьте извлечённый текст. Публикация пока только фиксирует проверку — поиск по базе появится позже."
      />
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <div className="mb-4 rounded-xl border border-border bg-surface p-4 shadow-sm">
        <dl className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          <div>
            <dt className="text-xs text-muted">Статус</dt>
            <dd className={cn("font-medium", statusClass(v.status))}>{documentStatusLabel(v.status)}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted">Версия</dt>
            <dd className="font-medium">{v.version_n}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted">Размер</dt>
            <dd className="font-medium">{formatBytes(v.size)}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted">Страниц</dt>
            <dd className="font-medium">{v.page_count ?? "—"}</dd>
          </div>
          <div className="col-span-2">
            <dt className="text-xs text-muted">Исходный файл</dt>
            <dd className="break-all font-medium">{v.original_name}</dd>
          </div>
          <div className="col-span-2">
            <dt className="text-xs text-muted">SHA-256</dt>
            <dd className="break-all font-mono text-xs">{v.content_hash || "—"}</dd>
          </div>
        </dl>
        {v.error_code ? (
          <p className="mt-3 text-sm text-red-600">{ERROR_LABEL[v.error_code] ?? "Ошибка обработки"}</p>
        ) : null}
        {v.warnings?.length ? (
          <ul className="mt-3 list-disc space-y-1 pl-5 text-sm text-muted">
            {v.warnings.map((w) => (
              <li key={w}>{WARNING_LABEL[w] ?? w}</li>
            ))}
          </ul>
        ) : null}
        {v.disk_file_id ? (
          <p className="mt-3 text-sm">
            <Link href="/files" className="text-accent hover:underline">
              Открыть диск
            </Link>
          </p>
        ) : null}
      </div>

      <div className="rounded-xl border border-border bg-surface p-4 shadow-sm">
        <h2 className="text-sm font-semibold">Извлечённый текст</h2>
        {documentInFlight(v.status) ? (
          <p className="mt-3 text-sm text-muted">Идёт обработка…</p>
        ) : (
          <pre className="mt-3 max-h-[32rem] overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border bg-zinc-50 p-3 text-sm">
            {v.extracted_text?.trim() ? v.extracted_text : "Текст пуст"}
          </pre>
        )}
      </div>

      {canReview ? (
        <div className="mt-4 rounded-xl border border-border bg-surface p-4 shadow-sm">
          <label className="block text-sm font-medium">Комментарий проверки</label>
          <textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            maxLength={500}
            rows={3}
            className="mt-2 w-full rounded-lg border border-border px-3 py-2 text-sm"
          />
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => void onReview("approve")}
              className="rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50"
            >
              Одобрить
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => void onReview("reject")}
              className="rounded-lg border border-border bg-surface px-3 py-2 text-sm disabled:opacity-50"
            >
              Отклонить
            </button>
          </div>
        </div>
      ) : null}

      {canRetry ? (
        <div className="mt-4">
          <button
            type="button"
            disabled={busy}
            onClick={() => void onRetry()}
            className="rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50"
          >
            Повторить обработку
          </button>
        </div>
      ) : null}
    </div>
  );
}
