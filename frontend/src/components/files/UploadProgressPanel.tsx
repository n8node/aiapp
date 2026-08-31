"use client";

import { AlertCircle, Check, Loader2, StopCircle, X } from "lucide-react";
import { cn } from "@/lib/cn";
import { formatBytes, type UploadJob } from "@/lib/files-api";

export function UploadProgressPanel({
  jobs,
  onCancel,
  onDismiss,
}: {
  jobs: UploadJob[];
  onCancel?: () => void;
  onDismiss?: () => void;
}) {
  if (jobs.length === 0) return null;

  const completedCount = jobs.filter((f) => f.status === "completed").length;
  const errorCount = jobs.filter((f) => f.status === "error").length;
  const cancelledCount = jobs.filter((f) => f.status === "cancelled").length;
  const totalCount = jobs.length;
  const isAllDone = completedCount + errorCount + cancelledCount === totalCount;

  const totalSize = jobs.reduce((sum, f) => sum + f.size, 0);
  const uploadedSize = jobs.reduce((sum, f) => {
    if (f.status === "completed") return sum + f.size;
    if (f.status === "uploading") return sum + (f.size * f.progress) / 100;
    return sum;
  }, 0);
  const overallProgress = totalSize > 0 ? Math.round((uploadedSize / totalSize) * 100) : 0;

  return (
    <div className="mb-4 overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
      <div className="flex items-center justify-between border-b border-border bg-zinc-50/80 px-4 py-3">
        <div className="flex items-center gap-3">
          {!isAllDone ? (
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-accent/10">
              <Loader2 className="h-4 w-4 animate-spin text-accent" />
            </div>
          ) : errorCount > 0 ? (
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-amber-100">
              <AlertCircle className="h-4 w-4 text-amber-600" />
            </div>
          ) : (
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-green-100">
              <Check className="h-4 w-4 text-green-600" />
            </div>
          )}
          <div>
            <p className="text-sm font-medium">
              {!isAllDone
                ? `Загрузка файлов (${completedCount} из ${totalCount})`
                : errorCount > 0
                  ? "Загружено с ошибками"
                  : "Загрузка завершена"}
            </p>
            <p className="text-xs text-muted">
              {formatBytes(uploadedSize)} из {formatBytes(totalSize)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {!isAllDone && onCancel ? (
            <button
              type="button"
              onClick={onCancel}
              className="inline-flex items-center gap-1.5 rounded-lg border border-amber-200 px-2.5 py-1.5 text-xs font-medium text-amber-700 hover:bg-amber-50"
            >
              <StopCircle className="h-3.5 w-3.5" />
              Отменить
            </button>
          ) : null}
          {isAllDone && onDismiss ? (
            <button
              type="button"
              onClick={onDismiss}
              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-muted hover:bg-zinc-100"
              title="Закрыть"
            >
              {errorCount > 0 ? "Убрать" : null}
              <X className="h-4 w-4" />
            </button>
          ) : null}
        </div>
      </div>

      {!isAllDone ? (
        <div className="px-4 py-2">
          <div className="relative h-2 overflow-hidden rounded-full bg-zinc-100">
            <div
              className="absolute left-0 top-0 h-full rounded-full bg-accent transition-all duration-300"
              style={{ width: `${overallProgress}%` }}
            />
          </div>
          <p className="mt-1 text-right text-xs text-muted">{overallProgress}%</p>
        </div>
      ) : null}

      <div className="max-h-64 overflow-y-auto">
        {jobs.map((file, index) => (
          <div
            key={file.id}
            className={cn(
              "flex items-center gap-3 px-4 py-2.5",
              index < jobs.length - 1 && "border-b border-border/60",
            )}
          >
            <div className="flex h-6 w-6 shrink-0 items-center justify-center">
              {file.status === "completed" ? (
                <div className="flex h-6 w-6 items-center justify-center rounded-full bg-green-100">
                  <Check className="h-3.5 w-3.5 text-green-600" />
                </div>
              ) : null}
              {file.status === "uploading" ? <Loader2 className="h-5 w-5 animate-spin text-accent" /> : null}
              {file.status === "pending" || file.status === "error" ? (
                <div className="h-5 w-5 rounded-full border-2 border-zinc-200" />
              ) : null}
              {file.status === "cancelled" ? (
                <div className="h-5 w-5 rounded-full border-2 border-zinc-300 bg-zinc-100" />
              ) : null}
            </div>
            <p className="min-w-0 flex-1 truncate text-sm">{file.name}</p>
            {file.status === "error" && file.error ? (
              <p className="hidden max-w-[40%] truncate text-xs text-amber-700 sm:block" title={file.error}>
                {file.error}
              </p>
            ) : null}
            <span className="shrink-0 text-xs text-muted">
              {file.status === "uploading"
                ? `${file.progress}%`
                : file.status === "completed"
                  ? "100%"
                  : formatBytes(file.size)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
