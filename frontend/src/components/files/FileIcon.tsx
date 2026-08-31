"use client";

import { useEffect, useState } from "react";
import { FileAudio, FileText, FileVideo, ImageIcon } from "lucide-react";
import { cn } from "@/lib/cn";
import { downloadFile } from "@/lib/files-api";

function iconForMime(mime: string, name?: string) {
  const ext = name?.split(".").pop()?.toLowerCase() ?? "";
  if (mime.startsWith("video/") || ["mp4", "webm", "mov", "mkv"].includes(ext)) return FileVideo;
  if (mime.startsWith("audio/") || ["mp3", "wav", "m4a", "ogg"].includes(ext)) return FileAudio;
  if (mime.startsWith("image/") || ["jpg", "jpeg", "png", "gif", "webp", "tif", "tiff"].includes(ext)) {
    return ImageIcon;
  }
  return FileText;
}

export function FileIcon({
  fileId,
  name,
  mimeType,
  className,
  rounded = "full",
}: {
  fileId?: string;
  name: string;
  mimeType: string;
  className?: string;
  rounded?: "full" | "md";
}) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const isImage = mimeType.startsWith("image/");
  const Icon = iconForMime(mimeType, name);

  useEffect(() => {
    if (!isImage || !fileId) return;
    let cancelled = false;
    void downloadFile(fileId, "inline")
      .then(({ url }) => {
        if (!cancelled) setPreviewUrl(url);
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [fileId, isImage]);

  return (
    <div
      className={cn(
        "relative h-11 w-11 shrink-0 overflow-hidden border border-border bg-zinc-100",
        rounded === "full" ? "rounded-full" : "rounded-md",
        className,
      )}
      title={name}
    >
      {isImage && previewUrl && !failed ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={previewUrl} alt="" loading="lazy" decoding="async" className="h-full w-full object-cover" />
      ) : (
        <div className="flex h-full w-full items-center justify-center">
          <Icon className="h-5 w-5 text-muted" />
        </div>
      )}
    </div>
  );
}
