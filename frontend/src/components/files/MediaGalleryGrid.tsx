"use client";

import { cn } from "@/lib/cn";
import type { WorkspaceFile } from "@/lib/files-api";
import { FileIcon } from "@/components/files/FileIcon";

export type MediaGridMode = "compact" | "large";

export function MediaGalleryGrid({
  files,
  mode,
  selected,
  onOpen,
  onToggleSelect,
}: {
  files: WorkspaceFile[];
  mode: MediaGridMode;
  selected: Set<string>;
  onOpen: (file: WorkspaceFile) => void;
  onToggleSelect: (id: string) => void;
}) {
  const gridClass =
    mode === "compact"
      ? "grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7"
      : "grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5";

  return (
    <div className={gridClass}>
      {files.map((f) => {
        const isSelected = selected.has(f.id);
        return (
          <div key={f.id} className="group relative">
            <button type="button" onClick={() => onOpen(f)} className="block w-full text-left">
              <FileIcon
                fileId={f.id}
                name={f.name}
                mimeType={f.mime_type}
                rounded="md"
                className={mode === "large" ? "h-36 w-full" : "h-24 w-full"}
              />
              <p className={cn("mt-1.5 truncate", mode === "large" ? "text-sm font-medium" : "text-[11px] text-muted")}>
                {f.name}
              </p>
            </button>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onToggleSelect(f.id);
              }}
              className={cn(
                "absolute left-2 top-2 flex h-5 w-5 items-center justify-center rounded-full border-2 bg-surface/90 shadow-sm",
                isSelected ? "border-accent bg-accent" : "border-white/90 hover:border-accent/70",
              )}
              aria-label="Выбрать файл"
            >
              {isSelected ? <span className="h-2 w-2 rounded-full bg-white" /> : null}
            </button>
          </div>
        );
      })}
    </div>
  );
}
