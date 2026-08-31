"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type ComponentType, type MouseEvent } from "react";
import {
  Check,
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
import { FileDetailPanel } from "@/components/files/FileDetailPanel";
import { MoveTargetDialog, type MoveTarget } from "@/components/files/MoveTargetDialog";
import { MediaGalleryGrid, type MediaGridMode } from "@/components/files/MediaGalleryGrid";
import { UploadProgressPanel } from "@/components/files/UploadProgressPanel";
import { NameDialog } from "@/components/files/NameDialog";
import { ConfirmDialog } from "@/components/files/ConfirmDialog";
import { FileToolbar } from "@/components/files/FileToolbar";
import { cn } from "@/lib/cn";
import {
  DEFAULT_VISIBLE_TABS,
  EMPTY_FILTERS,
  classifyFile,
  filterFiles,
  filterFolders,
  filtersActive,
  kindLabel,
  loadVisibleTabs,
  saveVisibleTabs,
  sortFiles,
  sortFolders,
  type FileKind,
  type FileViewFilters,
} from "@/lib/file-view";
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
  deleteFolder,
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

function SelectBox({
  on,
  label,
  onClick,
}: {
  on: boolean;
  label: string;
  onClick: (e: MouseEvent<HTMLButtonElement>) => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      className={cn(
        "flex h-4 w-4 shrink-0 items-center justify-center rounded border",
        on ? "border-accent bg-accent text-white" : "border-zinc-300 hover:border-accent/50",
      )}
    >
      {on ? <Check className="h-3 w-3" /> : null}
    </button>
  );
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
  const [previewFileId, setPreviewFileId] = useState<string | null>(null);
  const [mediaGridMode, setMediaGridMode] = useState<MediaGridMode>("compact");
  const [dragging, setDragging] = useState(false);
  const [filters, setFilters] = useState<FileViewFilters>(EMPTY_FILTERS);
  const [visibleTabs, setVisibleTabs] = useState<FileKind[]>(DEFAULT_VISIBLE_TABS);
  const [folderDialog, setFolderDialog] = useState(false);
  const [renameTarget, setRenameTarget] = useState<{ kind: "file" | "folder"; id: string; current: string } | null>(
    null,
  );
  const [confirm, setConfirm] = useState<
    | { mode: "delete-selected" }
    | { mode: "delete-file"; id: string }
    | { mode: "delete-folder"; id: string }
    | { mode: "empty-trash" }
    | { mode: "purge-selected" }
    | { mode: "purge-file"; id: string }
    | { mode: "purge-folder"; id: string }
    | null
  >(null);
  const [dialogBusy, setDialogBusy] = useState(false);
  const [dialogError, setDialogError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const abortRef = useRef(new Map<string, AbortController>());
  const dragDepth = useRef(0);
  const uploadLock = useRef(false);
  const lastClickedId = useRef<string | null>(null);

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
    setVisibleTabs(loadVisibleTabs());
  }, []);

  useEffect(() => {
    setFolderId(null);
    setFolderTrail([]);
    setSelected(new Set());
    setSelectedKinds(new Map());
    setUploadJobs([]);
    setPreviewFileId(null);
    setFilters(EMPTY_FILTERS);
  }, [workspace?.id]);

  useEffect(() => {
    setFilters(EMPTY_FILTERS);
    lastClickedId.current = null;
  }, [section]);

  useEffect(() => {
    if (loading) return;
    if (previewFileId && !files.some((f) => f.id === previewFileId)) {
      setPreviewFileId(null);
    }
  }, [loading, files, previewFileId]);

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

  const visibleFolders = useMemo(
    () => sortFolders(filterFolders(folders, filters), filters),
    [folders, filters],
  );
  const visibleFiles = useMemo(
    () => sortFiles(filterFiles(files, filters), filters),
    [files, filters],
  );
  const visibleIds = useMemo(
    () => [...visibleFolders.map((f) => f.id), ...visibleFiles.map((f) => f.id)],
    [visibleFolders, visibleFiles],
  );

  const applyRangeSelect = (id: string, kind: "file" | "folder", shiftKey: boolean) => {
    const prevClick = lastClickedId.current;
    lastClickedId.current = id;
    const kindOf = (itemId: string): "file" | "folder" =>
      visibleFolders.some((f) => f.id === itemId) ? "folder" : "file";
    if (shiftKey && prevClick && prevClick !== id) {
      const a = visibleIds.indexOf(prevClick);
      const b = visibleIds.indexOf(id);
      if (a >= 0 && b >= 0) {
        const [from, to] = a < b ? [a, b] : [b, a];
        const slice = visibleIds.slice(from, to + 1);
        setSelected((prev) => {
          const next = new Set(prev);
          for (const itemId of slice) next.add(itemId);
          return next;
        });
        setSelectedKinds((prev) => {
          const next = new Map(prev);
          for (const itemId of slice) next.set(itemId, kindOf(itemId));
          return next;
        });
        return;
      }
    }
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

  const allVisibleSelected = visibleIds.length > 0 && visibleIds.every((id) => selected.has(id));

  const toggleSelectAll = () => {
    if (allVisibleSelected) {
      setSelected(new Set());
      setSelectedKinds(new Map());
      return;
    }
    setSelected(new Set(visibleIds));
    const kinds = new Map<string, "file" | "folder">();
    for (const fo of visibleFolders) kinds.set(fo.id, "folder");
    for (const f of visibleFiles) kinds.set(f.id, "file");
    setSelectedKinds(kinds);
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

  const handleCreateFolder = async (name: string) => {
    setDialogBusy(true);
    setDialogError(null);
    try {
      await createFolder(name, folderId);
      setFolderDialog(false);
      await refresh();
    } catch (e) {
      setDialogError(e instanceof Error ? e.message : "Ошибка");
    } finally {
      setDialogBusy(false);
    }
  };

  const handleRenameConfirm = async (name: string) => {
    if (!renameTarget) return;
    setDialogBusy(true);
    setDialogError(null);
    try {
      if (renameTarget.kind === "file") await renameFile(renameTarget.id, name);
      else await renameFolder(renameTarget.id, name);
      setRenameTarget(null);
      await refresh();
    } catch (e) {
      setDialogError(e instanceof Error ? e.message : "Ошибка");
    } finally {
      setDialogBusy(false);
    }
  };

  const runConfirm = async () => {
    if (!confirm) return;
    setDialogBusy(true);
    setError(null);
    try {
      switch (confirm.mode) {
        case "delete-selected": {
          const fileIds = [...selected].filter((id) => selectedKinds.get(id) === "file");
          const folderIds = [...selected].filter((id) => selectedKinds.get(id) === "folder");
          if (fileIds.length) await bulkFiles(fileIds, "delete");
          if (folderIds.length) await bulkFolders(folderIds, "delete");
          break;
        }
        case "delete-file":
          await deleteFile(confirm.id);
          setPreviewFileId((cur) => (cur === confirm.id ? null : cur));
          break;
        case "delete-folder":
          await deleteFolder(confirm.id);
          break;
        case "empty-trash":
          await emptyTrash();
          break;
        case "purge-selected": {
          const fileIds = [...selected].filter((id) => selectedKinds.get(id) === "file");
          const folderIds = [...selected].filter((id) => selectedKinds.get(id) === "folder");
          for (const id of fileIds) await permanentDeleteTrash(id, "file");
          for (const id of folderIds) await permanentDeleteTrash(id, "folder");
          break;
        }
        case "purge-file":
          await permanentDeleteTrash(confirm.id, "file");
          break;
        case "purge-folder":
          await permanentDeleteTrash(confirm.id, "folder");
          break;
      }
      setConfirm(null);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка удаления");
      setConfirm(null);
    } finally {
      setDialogBusy(false);
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

  const grouped = section !== "trash" && section !== "my-files" ? groupByDate(visibleFiles) : null;
  const isUploading = uploadJobs.some((j) => j.status === "pending" || j.status === "uploading");
  const isGallerySection = section === "photos" || section === "videos";
  const previewFile = previewFileId ? (files.find((f) => f.id === previewFileId) ?? null) : null;
  const showDetail = Boolean(previewFile && section !== "trash");
  const hasVisible = visibleFiles.length > 0 || visibleFolders.length > 0;
  const hasAny = files.length > 0 || folders.length > 0;

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
          onClick={() => {
            setDialogError(null);
            setFolderDialog(true);
          }}
          className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface px-3 py-2 text-sm hover:bg-zinc-50"
        >
          <FolderPlus className="h-4 w-4" />
          Новая папка
        </button>
      ) : section === "trash" ? (
        <button
          type="button"
          onClick={() => setConfirm({ mode: "empty-trash" })}
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
                setPreviewFileId(null);
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

      <div className="flex min-w-0 flex-1 gap-3 xl:gap-4">
      <div className="min-w-0 flex-1">
        <PageHeader
          title={section === "my-files" ? myFilesTitle : (SECTIONS.find((s) => s.id === section)?.label ?? "Файлы")}
          crumbs={pageCrumbs}
          description={
            section === "trash"
              ? "Ранее удалённые объекты. Очистка корзины удаляет их из хранилища навсегда."
              : `Файлы пространства «${workspace.name}». Удаление сразу стирает объект в хранилище.`
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
        ) : !hasAny ? (
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
          <div className="space-y-4">
            <FileToolbar
              filters={filters}
              visibleTabs={visibleTabs}
              hideKindTabs={isGallerySection}
              onChange={setFilters}
              onVisibleTabsChange={(tabs) => {
                setVisibleTabs(tabs);
                saveVisibleTabs(tabs);
              }}
            />

            {hasVisible ? (
              <div className="flex items-center gap-2">
                <SelectBox on={allVisibleSelected} label="Выбрать все" onClick={() => toggleSelectAll()} />
                <button type="button" onClick={toggleSelectAll} className="text-sm text-muted hover:text-text">
                  {allVisibleSelected ? "Снять выделение" : `Выбрать все (${visibleIds.length})`}
                </button>
              </div>
            ) : null}

            {!hasVisible ? (
              <EmptyState
                title="Ничего не найдено"
                description="Сбросьте фильтры или выберите другую вкладку."
                action={
                  filtersActive(filters) ? (
                    <button
                      type="button"
                      onClick={() => setFilters(EMPTY_FILTERS)}
                      className="rounded-lg border border-border bg-surface px-3 py-2 text-sm hover:bg-zinc-50"
                    >
                      Сбросить фильтры
                    </button>
                  ) : undefined
                }
              />
            ) : (
          <div className="space-y-6">
            {section !== "photos" && section !== "videos" && visibleFolders.length > 0 ? (
              <section>
                <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted">
                  Папки ({visibleFolders.length})
                </h3>
                <div className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
                  {visibleFolders.map((fo) => (
                    <div key={fo.id} className="group flex items-center gap-3 px-3 py-2.5 hover:bg-zinc-50">
                      <SelectBox
                        on={selected.has(fo.id)}
                        label="Выбрать папку"
                        onClick={(e) => applyRangeSelect(fo.id, "folder", e.shiftKey)}
                      />
                      <button
                        type="button"
                        onClick={() => (section === "trash" ? undefined : setFolderId(fo.id))}
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
                      {section === "trash" ? (
                        <div className="flex items-center gap-1">
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
                            onClick={() => setConfirm({ mode: "purge-folder", id: fo.id })}
                            className="rounded p-1 text-red-500 hover:bg-red-50"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      ) : (
                        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100">
                          <button
                            type="button"
                            title="Переименовать"
                            onClick={() => {
                              setDialogError(null);
                              setRenameTarget({ kind: "folder", id: fo.id, current: fo.name });
                            }}
                            className="rounded p-1 text-muted hover:bg-zinc-100"
                          >
                            <Pencil className="h-4 w-4" />
                          </button>
                          <button
                            type="button"
                            title="Удалить папку"
                            onClick={() => setConfirm({ mode: "delete-folder", id: fo.id })}
                            className="rounded p-1 text-red-500 hover:bg-red-50"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </section>
            ) : null}

            {(grouped ?? [["", visibleFiles] as [string, WorkspaceFile[]]]).map(([label, groupFiles]) => (
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
                    onOpen={(f) => setPreviewFileId(f.id)}
                    onToggleSelect={(id, event) => applyRangeSelect(id, "file", Boolean(event?.shiftKey))}
                  />
                ) : groupFiles.length === 0 ? null : (
                  <div className="overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
                    <div className="hidden items-center gap-3 border-b border-border bg-zinc-50 px-3 py-2 text-xs font-semibold uppercase tracking-wide text-muted sm:flex">
                      <span className="w-4" />
                      <button type="button" className="min-w-0 flex-1 text-left" onClick={() => setFilters((f) => ({ ...f, sort: "name", dir: f.sort === "name" && f.dir === "asc" ? "desc" : "asc" }))}>
                        Имя
                      </button>
                      <button type="button" className="w-28 text-left" onClick={() => setFilters((f) => ({ ...f, sort: "type", dir: f.sort === "type" && f.dir === "asc" ? "desc" : "asc" }))}>
                        Тип
                      </button>
                      <button type="button" className="w-24 text-left" onClick={() => setFilters((f) => ({ ...f, sort: "size", dir: f.sort === "size" && f.dir === "asc" ? "desc" : "asc" }))}>
                        Размер
                      </button>
                      <button type="button" className="w-36 text-left" onClick={() => setFilters((f) => ({ ...f, sort: "date", dir: f.sort === "date" && f.dir === "asc" ? "desc" : "asc" }))}>
                        Дата
                      </button>
                      <span className="w-24" />
                    </div>
                    <div className="divide-y divide-border">
                    {groupFiles.map((f) => (
                      <div
                        key={f.id}
                        className={cn(
                          "group flex items-center gap-3 px-3 py-2.5 hover:bg-zinc-50",
                          previewFileId === f.id && "bg-accent/5",
                        )}
                      >
                        <SelectBox
                          on={selected.has(f.id)}
                          label="Выбрать файл"
                          onClick={(e) => applyRangeSelect(f.id, "file", e.shiftKey)}
                        />
                        <button
                          type="button"
                          onClick={() => section !== "trash" && setPreviewFileId(f.id)}
                          className="flex min-w-0 flex-1 items-center gap-3 text-left"
                        >
                          <FileIcon fileId={f.id} name={f.name} mimeType={f.mime_type} />
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-sm font-medium">{f.name}</p>
                            <p className="text-xs text-muted sm:hidden">
                              {formatBytes(f.size)} · {formatFileTime(f.created_at)}
                            </p>
                          </div>
                        </button>
                        <span className="hidden w-28 truncate text-xs text-muted sm:block">
                          {kindLabel(classifyFile(f))}
                        </span>
                        <span className="hidden w-24 text-xs text-muted sm:block">{formatBytes(f.size)}</span>
                        <span className="hidden w-36 text-xs text-muted lg:block">{formatFileTime(f.created_at)}</span>
                        <div className="flex w-24 items-center justify-end gap-1 opacity-0 group-hover:opacity-100">
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
                                onClick={() => {
                                  setDialogError(null);
                                  setRenameTarget({ kind: "file", id: f.id, current: f.name });
                                }}
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
                                onClick={() => setConfirm({ mode: "delete-file", id: f.id })}
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
                                onClick={() => setConfirm({ mode: "purge-file", id: f.id })}
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
                  </div>
                )}
              </section>
            ))}
          </div>
            )}
          </div>
        )}
      </div>

      {showDetail && previewFile ? (
        <FileDetailPanel
          file={previewFile}
          onClose={() => setPreviewFileId(null)}
          onDownload={() => openFile(previewFile.id, "attachment")}
          onRename={() => {
            setDialogError(null);
            setRenameTarget({ kind: "file", id: previewFile.id, current: previewFile.name });
          }}
          onCopy={() => void copyFile(previewFile.id, folderId).then(refresh)}
          onDelete={() => setConfirm({ mode: "delete-file", id: previewFile.id })}
        />
      ) : null}
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
                  onClick={() => setConfirm({ mode: "delete-selected" })}
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
                  onClick={() => setConfirm({ mode: "purge-selected" })}
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

      <NameDialog
        open={folderDialog}
        title="Новая папка"
        description="Папка появится в текущем каталоге пространства."
        confirmLabel="Создать"
        placeholder="Например, Договоры"
        submitting={dialogBusy}
        error={dialogError}
        onClose={() => setFolderDialog(false)}
        onConfirm={handleCreateFolder}
      />
      <NameDialog
        open={Boolean(renameTarget)}
        title="Переименовать"
        confirmLabel="Сохранить"
        initialValue={renameTarget?.current ?? ""}
        submitting={dialogBusy}
        error={dialogError}
        onClose={() => setRenameTarget(null)}
        onConfirm={handleRenameConfirm}
      />
      <ConfirmDialog
        open={Boolean(confirm)}
        title={
          confirm?.mode === "empty-trash"
            ? "Очистить корзину"
            : confirm?.mode === "purge-selected" || confirm?.mode === "purge-file" || confirm?.mode === "purge-folder"
              ? "Удалить навсегда"
              : "Удалить"
        }
        description={
          confirm?.mode === "empty-trash"
            ? "Все файлы из корзины будут удалены из хранилища навсегда."
            : confirm?.mode === "purge-selected" || confirm?.mode === "purge-file" || confirm?.mode === "purge-folder"
              ? "Объекты будут удалены из хранилища без восстановления."
              : confirm?.mode === "delete-folder" ||
                  (confirm?.mode === "delete-selected" &&
                    [...selected].some((id) => selectedKinds.get(id) === "folder"))
                ? "Папка и все файлы внутри будут удалены из хранилища навсегда."
                : "Файлы будут удалены из хранилища навсегда."
        }
        confirmLabel={confirm?.mode === "empty-trash" ? "Очистить" : "Удалить"}
        danger
        submitting={dialogBusy}
        onClose={() => setConfirm(null)}
        onConfirm={runConfirm}
      />
    </div>
  );
}
