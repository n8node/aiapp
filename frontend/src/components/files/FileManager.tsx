"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type ComponentType } from "react";
import {
  Clock,
  Copy,
  Download,
  Folder,
  FolderInput,
  FolderPlus,
  Grid3x3,
  ImageIcon,
  LayoutGrid,
  Pencil,
  RotateCcw,
  Trash2,
  Upload,
  Video,
} from "lucide-react";
import { PageHeader, type Crumb } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/layout/EmptyState";
import { FileIcon } from "@/components/files/FileIcon";
import { MoveTargetDialog, type MoveTarget } from "@/components/files/MoveTargetDialog";
import { MediaGalleryGrid, type MediaGridMode } from "@/components/files/MediaGalleryGrid";
import { UploadProgressPanel } from "@/components/files/UploadProgressPanel";
import { cn } from "@/lib/cn";
import {
  type FilesSection,
  type FolderBreadcrumb,
  type UploadJob,
  type WorkspaceFile,
  type WorkspaceFolder,
  bulkFiles,
  bulkFolders,
  copyFile,
  createFolder,
  deleteFile,
  emptyTrash,
  fetchFolderBreadcrumbs,
  formatBytes,
  isDiskApiError,
  listFiles,
  listFolders,
  listTrash,
  openFile,
  permanentDeleteTrash,
  renameFile,
  renameFolder,
  restoreTrash,
  uploadFile,
} from "@/lib/files-api";

const SECTIONS: { id: FilesSection; label: string; icon: ComponentType<{ className?: string }> }[] = [
  { id: "my-files", label: "Файлы", icon: Folder },
  { id: "recent", label: "Недавние", icon: Clock },
  { id: "photos", label: "Фото", icon: ImageIcon },
  { id: "videos", label: "Видео", icon: Video },
  { id: "trash", label: "Корзина", icon: Trash2 },
];

function groupByDate(items: WorkspaceFile[]) {
  const map = new Map<string, WorkspaceFile[]>();
  for (const f of items) {
    const d = new Date(f.created_at);
    const key = d.toLocaleDateString("ru-RU", { weekday: "long", day: "numeric", month: "long" });
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(f);
  }
  return [...map.entries()];
}

function formatFileTime(iso: string) {
  const d = new Date(iso);
  const now = new Date();
  const sameDay =
    d.getDate() === now.getDate() && d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
  const time = d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  if (sameDay) return `Сегодня ${time}`;
  return d.toLocaleDateString("ru-RU", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
}

export function FileManager({
  workspace,
}: {
  workspace: { id: string; name: string } | null;
}) {
  const [section, setSection] = useState<FilesSection>("my-files");
  const [folderId, setFolderId] = useState<string | null>(null);
  const [folderTrail, setFolderTrail] = useState<FolderBreadcrumb[]>([]);
  const [files, setFiles] = useState<WorkspaceFile[]>([]);
  const [folders, setFolders] = useState<WorkspaceFolder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [storageMissing, setStorageMissing] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [selectedKinds, setSelectedKinds] = useState<Map<string, "file" | "folder">>(new Map());
  const [uploadJobs, setUploadJobs] = useState<UploadJob[]>([]);
  const [moveDialog, setMoveDialog] = useState<{ mode: "move" | "copy" } | null>(null);
  const [mediaGridMode, setMediaGridMode] = useState<MediaGridMode>("compact");
  const [dragging, setDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const abortRef = useRef(new Map<string, AbortController>());
  const dragDepth = useRef(0);
  const uploadLock = useRef(false);

  const refresh = useCallback(async () => {
    if (!workspace) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    setStorageMissing(false);
    try {
      if (section === "trash") {
        const tr = await listTrash();
        setFiles(tr.files);
        setFolders(tr.folders);
      } else {
        const [fRes, foRes] = await Promise.all([
          listFiles(section, section === "my-files" ? folderId : null),
          section === "my-files" ? listFolders(folderId) : Promise.resolve({ folders: [] as WorkspaceFolder[] }),
        ]);
        setFiles(fRes.files);
        setFolders(foRes.folders);
      }
      setSelected(new Set());
      setSelectedKinds(new Map());
    } catch (e) {
      if (isDiskApiError(e, "storage_not_configured")) {
        setStorageMissing(true);
        setFiles([]);
        setFolders([]);
      } else {
        setError(e instanceof Error ? e.message : "Ошибка загрузки");
      }
    } finally {
      setLoading(false);
    }
  }, [section, folderId, workspace]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    setFolderId(null);
    setFolderTrail([]);
    setSelected(new Set());
    setSelectedKinds(new Map());
    setUploadJobs([]);
  }, [workspace?.id]);

  useEffect(() => {
    if (section !== "my-files" || !folderId) {
      setFolderTrail([]);
      return;
    }
    let cancelled = false;
    void fetchFolderBreadcrumbs(folderId)
      .then((data) => {
        if (!cancelled) setFolderTrail(data.breadcrumbs);
      })
      .catch(() => {
        if (!cancelled) setFolderTrail([]);
      });
    return () => {
      cancelled = true;
    };
  }, [section, folderId]);

  const toggleSelect = (id: string, kind: "file" | "folder") => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
    setSelectedKinds((prev) => {
      const next = new Map(prev);
      if (next.has(id)) next.delete(id);
      else next.set(id, kind);
      return next;
    });
  };

  const selectedSize = useMemo(() => {
    let sum = 0;
    for (const f of files) {
      if (selected.has(f.id)) sum += f.size;
    }
    return sum;
  }, [files, selected]);

  const runUploads = useCallback(
    async (fileList: File[]) => {
      if (!fileList.length || !workspace || uploadLock.current || section === "trash") return;
      uploadLock.current = true;
      const jobs: UploadJob[] = fileList.map((file, i) => ({
        id: `${Date.now()}-${i}-${file.name}`,
        name: file.name,
        size: file.size,
        progress: 0,
        status: "pending",
      }));
      setUploadJobs(jobs);
      const targetFolder = section === "my-files" ? folderId : null;
      try {
        for (const [i, file] of fileList.entries()) {
          const job = jobs[i];
          const controller = new AbortController();
          abortRef.current.set(job.id, controller);
          setUploadJobs((prev) =>
            prev.map((j) => (j.id === job.id ? { ...j, status: "uploading" as const } : j)),
          );
          try {
            const created = await uploadFile(
              file,
              targetFolder,
              (pct) => {
                setUploadJobs((prev) => prev.map((j) => (j.id === job.id ? { ...j, progress: pct } : j)));
              },
              controller.signal,
            );
            setUploadJobs((prev) =>
              prev.map((j) => (j.id === job.id ? { ...j, status: "completed" as const, progress: 100 } : j)),
            );
            if (section === "my-files" && (created.folder_id ?? null) === targetFolder) {
              setFiles((prev) => (prev.some((f) => f.id === created.id) ? prev : [created, ...prev]));
            }
          } catch (e) {
            const aborted = e instanceof DOMException && e.name === "AbortError";
            setUploadJobs((prev) =>
              prev.map((j) =>
                j.id === job.id
                  ? {
                      ...j,
                      status: aborted ? "cancelled" : "error",
                      error: aborted ? undefined : e instanceof Error ? e.message : "Ошибка",
                    }
                  : j,
              ),
            );
            if (aborted) break;
          } finally {
            abortRef.current.delete(job.id);
          }
        }
      } finally {
        uploadLock.current = false;
      }
    },
    [workspace, section, folderId],
  );

  const handleUpload = (list: FileList | File[] | null) => {
    if (!list || list.length === 0) return;
    void runUploads([...list]);
  };

  const cancelUploads = () => {
    for (const c of abortRef.current.values()) c.abort();
  };

  const handleCreateFolder = async () => {
    const name = window.prompt("Имя папки");
    if (!name?.trim()) return;
    try {
      await createFolder(name.trim(), folderId);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка");
    }
  };

  const handleRename = async (kind: "file" | "folder", id: string, current: string) => {
    const name = window.prompt("Новое имя", current);
    if (!name?.trim() || name.trim() === current) return;
    try {
      if (kind === "file") await renameFile(id, name.trim());
      else await renameFolder(id, name.trim());
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка");
    }
  };

  const handleDeleteSelected = async () => {
    if (selected.size === 0) return;
    const fileIds = [...selected].filter((id) => selectedKinds.get(id) === "file");
    const folderIds = [...selected].filter((id) => selectedKinds.get(id) === "folder");
    if (section === "trash") {
      if (!window.confirm(`Удалить выбранное навсегда (${selected.size})?`)) return;
      try {
        for (const id of fileIds) await permanentDeleteTrash(id, "file");
        for (const id of folderIds) await permanentDeleteTrash(id, "folder");
        await refresh();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Ошибка удаления");
      }
      return;
    }
    if (!window.confirm(`Удалить выбранное (${selected.size})?`)) return;
    try {
      if (fileIds.length) await bulkFiles(fileIds, "delete");
      if (folderIds.length) await bulkFolders(folderIds, "delete");
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка удаления");
    }
  };

  const handleDownloadSelected = async () => {
    const fileIds = [...selected].filter((id) => selectedKinds.get(id) === "file");
    for (const id of fileIds) {
      openFile(id, "attachment");
    }
  };

  const handleMoveConfirm = async (target: MoveTarget) => {
    const fileIds = [...selected].filter((id) => selectedKinds.get(id) === "file");
    const folderIds = [...selected].filter((id) => selectedKinds.get(id) === "folder");
    const action = moveDialog?.mode ?? "move";
    if (action === "copy" && folderIds.length && !fileIds.length) {
      throw new Error("Копирование папок пока недоступно");
    }
    if (fileIds.length) await bulkFiles(fileIds, action, target.folderId);
    if (action === "move" && folderIds.length) await bulkFolders(folderIds, "move", target.folderId);
    setSelected(new Set());
    setSelectedKinds(new Map());
    await refresh();
  };

  const handleRestore = async (fileIds: string[], folderIds: string[]) => {
    try {
      await restoreTrash(fileIds, folderIds);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка восстановления");
    }
  };

  const handleRestoreSelected = async () => {
    const fileIds = [...selected].filter((id) => selectedKinds.get(id) === "file");
    const folderIds = [...selected].filter((id) => selectedKinds.get(id) === "folder");
    await handleRestore(fileIds, folderIds);
  };

  const grouped = section !== "trash" && section !== "my-files" ? groupByDate(files) : null;
  const isUploading = uploadJobs.some((j) => j.status === "pending" || j.status === "uploading");
  const isGallerySection = section === "photos" || section === "videos";

  const myFilesTitle = useMemo(() => {
    if (!folderId) return "Файлы";
    return folderTrail[folderTrail.length - 1]?.name ?? "…";
  }, [folderId, folderTrail]);

  const pageCrumbs = useMemo((): Crumb[] | undefined => {
    if (section !== "my-files") return undefined;
    const trail: Crumb[] = [{ label: "Главная", href: "/dashboard" }];
    if (!folderId) {
      trail.push({ label: "Файлы" });
      return trail;
    }
    for (const crumb of folderTrail) {
      trail.push({
        label: crumb.name,
        onClick: crumb.id ? () => setFolderId(crumb.id) : () => setFolderId(null),
      });
    }
    return trail;
  }, [section, folderId, folderTrail]);

  const headerActions = (
    <>
      {isGallerySection ? (
        <div className="flex rounded-lg border border-border p-0.5">
          <button
            type="button"
            title="Компактная сетка"
            onClick={() => setMediaGridMode("compact")}
            className={cn(
              "rounded-md p-2 transition-colors",
              mediaGridMode === "compact" ? "bg-accent/10 text-accent" : "text-muted hover:bg-zinc-50",
            )}
          >
            <Grid3x3 className="h-4 w-4" />
          </button>
          <button
            type="button"
            title="Крупная сетка"
            onClick={() => setMediaGridMode("large")}
            className={cn(
              "rounded-md p-2 transition-colors",
              mediaGridMode === "large" ? "bg-accent/10 text-accent" : "text-muted hover:bg-zinc-50",
            )}
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
        </div>
      ) : null}
      {section === "my-files" ? (
        <button
          type="button"
          onClick={() => void handleCreateFolder()}
          className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface px-3 py-2 text-sm hover:bg-zinc-50"
        >
          <FolderPlus className="h-4 w-4" />
          Новая папка
        </button>
      ) : section === "trash" ? (
        <button
          type="button"
          onClick={() => {
            if (window.confirm("Очистить корзину полностью? Файлы будут удалены из хранилища навсегда.")) {
              void emptyTrash().then(refresh);
            }
          }}
          className="rounded-lg border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
        >
          Очистить корзину
        </button>
      ) : null}
    </>
  );

  if (!workspace) {
    return (
      <EmptyState
        title="Нет пространства"
        description="Сначала выберите пространство отдела в меню слева — диск принадлежит отделу, а не личному кабинету."
      />
    );
  }

  return (
    <div
      className="relative flex min-h-0 gap-3 xl:gap-4"
      onDragEnter={(e) => {
        if (![...e.dataTransfer.types].includes("Files")) return;
        dragDepth.current += 1;
        setDragging(true);
      }}
      onDragLeave={() => {
        dragDepth.current = Math.max(0, dragDepth.current - 1);
        if (dragDepth.current === 0) setDragging(false);
      }}
      onDragOver={(e) => {
        if (![...e.dataTransfer.types].includes("Files")) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "copy";
      }}
      onDrop={(e) => {
        e.preventDefault();
        dragDepth.current = 0;
        setDragging(false);
        if (section === "trash" || storageMissing) return;
        handleUpload(e.dataTransfer.files);
      }}
    >
      {dragging && section !== "trash" && !storageMissing ? (
        <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center rounded-xl border-2 border-dashed border-accent bg-accent/5">
          <p className="rounded-lg bg-surface px-4 py-2 text-sm font-medium shadow-sm">
            Отпустите файлы, чтобы загрузить в пространство
          </p>
        </div>
      ) : null}

      <aside className="w-44 shrink-0 space-y-4 xl:w-48">
        <button
          type="button"
          disabled={section === "trash" || storageMissing}
          onClick={() => fileInputRef.current?.click()}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-accent px-3 py-2.5 text-sm font-medium text-white hover:bg-accent/90 disabled:opacity-60"
        >
          <Upload className="h-4 w-4" />
          {isUploading ? "Загрузка…" : "Загрузить"}
        </button>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(e) => {
            handleUpload(e.target.files);
            e.target.value = "";
          }}
        />

        <nav className="space-y-1 rounded-xl border border-border bg-surface p-2 shadow-sm">
          <p className="px-2 py-1 text-xs font-semibold uppercase tracking-wide text-muted">Диск</p>
          {SECTIONS.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              type="button"
              onClick={() => {
                setSection(id);
                if (id !== "my-files") setFolderId(null);
              }}
              className={cn(
                "flex w-full items-center gap-2 rounded-lg px-2 py-2 text-sm",
                section === id ? "bg-zinc-100 font-medium text-text" : "text-muted hover:bg-zinc-50",
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
            </button>
          ))}
        </nav>
      </aside>

      <div className="min-w-0 flex-1">
        <PageHeader
          title={section === "my-files" ? myFilesTitle : (SECTIONS.find((s) => s.id === section)?.label ?? "Файлы")}
          crumbs={pageCrumbs}
          description={
            section === "trash"
              ? "Удалённые файлы пространства. Очистка корзины удаляет их из хранилища навсегда."
              : `Файлы пространства «${workspace.name}». Доступны всем участникам отдела.`
          }
          actions={headerActions}
        />

        {section === "my-files" && folderId ? (
          <button type="button" onClick={() => setFolderId(null)} className="mb-3 text-sm text-accent hover:underline">
            ← Назад к корню
          </button>
        ) : null}

        {uploadJobs.length > 0 ? (
          <UploadProgressPanel
            jobs={uploadJobs}
            onCancel={cancelUploads}
            onDismiss={() => setUploadJobs([])}
          />
        ) : null}

        {error ? (
          <div className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>
        ) : null}

        {storageMissing ? (
          <EmptyState
            title="Хранилище не подключено"
            description="Администратор платформы должен указать бакет S3 в разделе «S3 — хранилище»."
          />
        ) : loading ? (
          <p className="text-sm text-muted">Загрузка…</p>
        ) : files.length === 0 && folders.length === 0 ? (
          <EmptyState
            title={section === "trash" ? "Корзина пуста" : "Нет файлов"}
            description={
              section === "trash"
                ? "Удалённые файлы и папки появятся здесь."
                : "Перетащите файлы сюда или нажмите «Загрузить». Можно создать папку."
            }
            action={
              section !== "trash" ? (
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="rounded-lg bg-accent px-3 py-2 text-sm font-medium text-white hover:bg-accent/90"
                >
                  Загрузить файлы
                </button>
              ) : undefined
            }
          />
        ) : (
          <div className="space-y-6">
            {section === "my-files" && folders.length > 0 ? (
              <section>
                <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
                  Папки ({folders.length})
                </h3>
                <div className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
                  {folders.map((fo) => (
                    <div key={fo.id} className="group flex items-center gap-3 px-3 py-2.5 hover:bg-zinc-50">
                      <button
                        type="button"
                        onClick={() => toggleSelect(fo.id, "folder")}
                        className={cn(
                          "flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2",
                          selected.has(fo.id) ? "border-accent bg-accent" : "border-zinc-300 hover:border-accent/50",
                        )}
                        aria-label="Выбрать папку"
                      >
                        {selected.has(fo.id) ? <span className="h-2 w-2 rounded-full bg-white" /> : null}
                      </button>
                      <button
                        type="button"
                        onClick={() => setFolderId(fo.id)}
                        className="flex min-w-0 flex-1 items-center gap-3 text-left"
                      >
                        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-border bg-amber-50">
                          <Folder className="h-5 w-5 text-amber-500" />
                        </div>
                        <div className="min-w-0">
                          <p className="truncate text-sm font-medium">{fo.name}</p>
                          <p className="text-xs text-muted">
                            Папка · Файлов: {fo.files_count ?? 0} · {formatFileTime(fo.created_at)}
                          </p>
                        </div>
                      </button>
                      <button
                        type="button"
                        title="Переименовать"
                        onClick={() => void handleRename("folder", fo.id, fo.name)}
                        className="rounded p-1 text-muted opacity-0 hover:bg-zinc-100 group-hover:opacity-100"
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
                </div>
              </section>
            ) : null}

            {section === "trash" && folders.length > 0 ? (
              <div className="space-y-2">
                {folders.map((fo) => (
                  <div
                    key={fo.id}
                    className="flex items-center gap-3 rounded-xl border border-border bg-surface px-3 py-2 shadow-sm"
                  >
                    <button
                      type="button"
                      onClick={() => toggleSelect(fo.id, "folder")}
                      className={cn(
                        "flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2",
                        selected.has(fo.id) ? "border-accent bg-accent" : "border-zinc-300",
                      )}
                      aria-label="Выбрать папку"
                    >
                      {selected.has(fo.id) ? <span className="h-2 w-2 rounded-full bg-white" /> : null}
                    </button>
                    <Folder className="h-5 w-5 text-amber-500" />
                    <span className="flex-1 truncate text-sm font-medium">{fo.name}</span>
                    <button
                      type="button"
                      title="Восстановить"
                      onClick={() => void handleRestore([], [fo.id])}
                      className="rounded p-1 hover:bg-zinc-100"
                    >
                      <RotateCcw className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      title="Удалить навсегда"
                      onClick={() => {
                        if (window.confirm("Удалить папку навсегда?")) {
                          void permanentDeleteTrash(fo.id, "folder").then(refresh);
                        }
                      }}
                      className="rounded p-1 text-red-500 hover:bg-red-50"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            ) : null}

            {(grouped ?? [["", files] as [string, WorkspaceFile[]]]).map(([label, groupFiles]) => (
              <section key={label || "all"}>
                {section === "my-files" && groupFiles.length > 0 ? (
                  <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
                    Файлы ({groupFiles.length})
                  </h3>
                ) : null}
                {label ? <h3 className="mb-2 text-sm font-medium capitalize text-muted">{label}</h3> : null}

                {isGallerySection ? (
                  <MediaGalleryGrid
                    files={groupFiles}
                    mode={mediaGridMode}
                    selected={selected}
                    onOpen={(f) => openFile(f.id, "inline")}
                    onToggleSelect={(id) => toggleSelect(id, "file")}
                  />
                ) : (
                  <div className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
                    {groupFiles.map((f) => (
                      <div key={f.id} className="group flex items-center gap-3 px-3 py-2.5 hover:bg-zinc-50">
                        <button
                          type="button"
                          onClick={() => toggleSelect(f.id, "file")}
                          className={cn(
                            "flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2",
                            selected.has(f.id) ? "border-accent bg-accent" : "border-zinc-300 hover:border-accent/50",
                          )}
                          aria-label="Выбрать файл"
                        >
                          {selected.has(f.id) ? <span className="h-2 w-2 rounded-full bg-white" /> : null}
                        </button>
                        <div className="flex min-w-0 flex-1 items-center gap-3">
                          <FileIcon fileId={f.id} name={f.name} mimeType={f.mime_type} />
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-sm font-medium">{f.name}</p>
                            <p className="text-xs text-muted">
                              {formatBytes(f.size)} · {formatFileTime(f.created_at)}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100">
                          {section !== "trash" ? (
                            <>
                              <button
                                type="button"
                                title="Скачать"
                                onClick={() => openFile(f.id, "attachment")}
                                className="rounded p-1.5 text-muted hover:bg-zinc-100"
                              >
                                <Download className="h-4 w-4" />
                              </button>
                              <button
                                type="button"
                                title="Переименовать"
                                onClick={() => void handleRename("file", f.id, f.name)}
                                className="rounded p-1.5 text-muted hover:bg-zinc-100"
                              >
                                <Pencil className="h-4 w-4" />
                              </button>
                              <button
                                type="button"
                                title="Копировать"
                                onClick={() => void copyFile(f.id, folderId).then(refresh)}
                                className="rounded p-1.5 text-muted hover:bg-zinc-100"
                              >
                                <Copy className="h-4 w-4" />
                              </button>
                              <button
                                type="button"
                                title="Удалить"
                                onClick={() => void deleteFile(f.id).then(refresh)}
                                className="rounded p-1.5 text-red-500 hover:bg-red-50"
                              >
                                <Trash2 className="h-4 w-4" />
                              </button>
                            </>
                          ) : (
                            <>
                              <button
                                type="button"
                                title="Восстановить"
                                onClick={() => void handleRestore([f.id], [])}
                                className="rounded p-1.5 text-muted hover:bg-zinc-100"
                              >
                                <RotateCcw className="h-4 w-4" />
                              </button>
                              <button
                                type="button"
                                title="Удалить навсегда"
                                onClick={() => {
                                  if (window.confirm("Удалить файл навсегда?")) {
                                    void permanentDeleteTrash(f.id, "file").then(refresh);
                                  }
                                }}
                                className="rounded p-1.5 text-red-500 hover:bg-red-50"
                              >
                                <Trash2 className="h-4 w-4" />
                              </button>
                            </>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </section>
            ))}
          </div>
        )}
      </div>

      {selected.size > 0 ? (
        <div className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2">
          <div className="flex flex-wrap items-center gap-2 rounded-xl border border-border bg-surface px-4 py-3 shadow-sm">
            <span className="text-sm font-medium">
              {selected.size} выбрано · {formatBytes(selectedSize)}
            </span>
            {section !== "trash" ? (
              <>
                <button
                  type="button"
                  onClick={() => void handleDownloadSelected()}
                  className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm hover:bg-zinc-100"
                >
                  <Download className="h-4 w-4" /> Скачать
                </button>
                <button
                  type="button"
                  onClick={() => setMoveDialog({ mode: "move" })}
                  className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm hover:bg-zinc-100"
                >
                  <FolderInput className="h-4 w-4" /> Переместить
                </button>
                <button
                  type="button"
                  onClick={() => setMoveDialog({ mode: "copy" })}
                  className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm hover:bg-zinc-100"
                >
                  <Copy className="h-4 w-4" /> Копировать
                </button>
                <button
                  type="button"
                  onClick={() => void handleDeleteSelected()}
                  className="inline-flex items-center gap-1 rounded-lg bg-red-600 px-2 py-1.5 text-sm text-white hover:bg-red-700"
                >
                  <Trash2 className="h-4 w-4" /> Удалить
                </button>
              </>
            ) : (
              <>
                <button
                  type="button"
                  onClick={() => void handleRestoreSelected()}
                  className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-sm hover:bg-zinc-100"
                >
                  <RotateCcw className="h-4 w-4" /> Восстановить
                </button>
                <button
                  type="button"
                  onClick={() => void handleDeleteSelected()}
                  className="inline-flex items-center gap-1 rounded-lg bg-red-600 px-2 py-1.5 text-sm text-white hover:bg-red-700"
                >
                  <Trash2 className="h-4 w-4" /> Удалить навсегда
                </button>
              </>
            )}
            <button
              type="button"
              onClick={() => {
                setSelected(new Set());
                setSelectedKinds(new Map());
              }}
              className="text-sm text-muted hover:text-text"
            >
              Снять
            </button>
          </div>
        </div>
      ) : null}

      {moveDialog ? (
        <MoveTargetDialog
          open
          mode={moveDialog.mode}
          workspaceName={workspace.name}
          onClose={() => setMoveDialog(null)}
          onConfirm={handleMoveConfirm}
        />
      ) : null}
    </div>
  );
}
