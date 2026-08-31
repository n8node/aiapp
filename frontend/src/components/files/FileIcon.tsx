"use client";

import { FileAudio, FileText, FileVideo, ImageIcon } from "lucide-react";
import { cn } from "@/lib/cn";
import { fileContentURL } from "@/lib/files-api";

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
  const Icon = iconForMime(mimeType, name);
  const isImage = mimeType.startsWith("image/");

  return (
    <div
      className={cn(
        "relative h-11 w-11 shrink-0 overflow-hidden border border-border bg-zinc-100",
        rounded === "full" ? "rounded-full" : "rounded-md",
        className,
      )}
      title={name}
    >
      {isImage && fileId ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={fileContentURL(fileId, "inline")}
          alt=""
          loading="lazy"
          decoding="async"
          className="h-full w-full object-cover"
        />
      ) : (
        <div className="flex h-full w-full items-center justify-center">
          <Icon className="h-5 w-5 text-muted" />
        </div>
      )}
    </div>
  );
}
