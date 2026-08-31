"use client";

import {
  Copy,
  Download,
  FileAudio,
  FileText,
  FileVideo,
  ImageIcon,
  Pencil,
  Trash2,
  X,
} from "lucide-react";
import { cn } from "@/lib/cn";
import { fileContentURL, formatBytes, type WorkspaceFile } from "@/lib/files-api";

type Props = {
  file: WorkspaceFile;
  onClose: () => void;
  onDownload: () => void;
  onRename: () => void;
  onCopy: () => void;
  onDelete: () => void;
};

function formatFullDate(iso: string) {
  return new Date(iso).toLocaleString("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function isVideoMime(mime: string, name: string) {
  if (mime.startsWith("video/")) return true;
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  return ["mp4", "webm", "mov", "mkv"].includes(ext);
}

function isAudioMime(mime: string, name: string) {
  if (mime.startsWith("audio/")) return true;
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  return ["mp3", "wav", "m4a", "ogg"].includes(ext);
}

export function FileDetailPanel({ file, onClose, onDownload, onRename, onCopy, onDelete }: Props) {
  const previewSrc = fileContentURL(file.id, "inline");
  const isImage = file.mime_type.startsWith("image/");
  const isVideo = isVideoMime(file.mime_type, file.name);
  const isAudio = isAudioMime(file.mime_type, file.name);
  const FallbackIcon = isVideo ? FileVideo : isAudio ? FileAudio : isImage ? ImageIcon : FileText;

  return (
    <>
      <button
        type="button"
        className="fixed inset-0 z-40 bg-black/30 xl:hidden"
        aria-label="Закрыть сведения"
        onClick={onClose}
      />
      <aside className="fixed inset-y-0 right-0 z-50 flex h-full w-[min(100%,20rem)] flex-col overflow-hidden border-l border-border bg-surface shadow-sm xl:sticky xl:top-4 xl:z-auto xl:h-fit xl:max-h-[calc(100vh-5rem)] xl:w-80 xl:rounded-xl xl:border">
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold">Сведения о файле</h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1 text-muted hover:bg-zinc-100"
            aria-label="Закрыть"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="overflow-y-auto px-4 py-4">
          <div
            className={cn(
              "overflow-hidden rounded-xl border border-border bg-zinc-50",
              isAudio ? "p-6" : "aspect-[4/3]",
            )}
          >
            {isImage ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={previewSrc}
                alt=""
                loading="lazy"
                decoding="async"
                className="h-full w-full object-contain"
              />
            ) : isVideo ? (
              <video src={previewSrc} controls className="h-full w-full bg-black object-contain" />
            ) : isAudio ? (
              <div className="flex flex-col items-center gap-4">
                <FileAudio className="h-12 w-12 text-accent" />
                <audio src={previewSrc} controls className="w-full" />
              </div>
            ) : (
              <div className="flex h-full min-h-[10rem] flex-col items-center justify-center gap-2 text-muted">
                <FallbackIcon className="h-12 w-12" />
              </div>
            )}
          </div>

          <div className="mt-4 space-y-3">
            <div>
              <p className="text-xs font-semibold uppercase tracking-wide text-muted">Имя</p>
              <p className="mt-1 break-all text-sm font-medium">{file.name}</p>
            </div>
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-xs text-muted">Размер</p>
                <p className="font-medium">{formatBytes(file.size)}</p>
              </div>
              <div>
                <p className="text-xs text-muted">Тип</p>
                <p className="font-medium">{file.mime_type || "—"}</p>
              </div>
              <div className="col-span-2">
                <p className="text-xs text-muted">Загружен</p>
                <p className="font-medium">{formatFullDate(file.created_at)}</p>
              </div>
            </div>
          </div>

          <div className="mt-4 flex flex-wrap gap-2 border-t border-border pt-4">
            <button
              type="button"
              onClick={onDownload}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-zinc-50"
            >
              <Download className="h-4 w-4" />
              Скачать
            </button>
            <button
              type="button"
              onClick={onRename}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-zinc-50"
            >
              <Pencil className="h-4 w-4" />
              Переименовать
            </button>
            <button
              type="button"
              onClick={onCopy}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm hover:bg-zinc-50"
            >
              <Copy className="h-4 w-4" />
              Копировать
            </button>
            <button
              type="button"
              onClick={onDelete}
              className="inline-flex items-center gap-1.5 rounded-lg border border-red-200 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50"
            >
              <Trash2 className="h-4 w-4" />
              Удалить
            </button>
          </div>
        </div>
      </aside>
    </>
  );
}
