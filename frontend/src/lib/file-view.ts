import type { WorkspaceFile, WorkspaceFolder } from "@/lib/files-api";

export type FileKind = "all" | "folder" | "document" | "image" | "video" | "audio" | "archive" | "other";

export const FILE_KIND_TABS: { id: FileKind; label: string }[] = [
  { id: "all", label: "Все" },
  { id: "folder", label: "Папки" },
  { id: "document", label: "Документы" },
  { id: "image", label: "Изображения" },
  { id: "video", label: "Видео" },
  { id: "audio", label: "Аудио" },
  { id: "archive", label: "Архивы" },
  { id: "other", label: "Другое" },
];

export const DEFAULT_VISIBLE_TABS: FileKind[] = ["all", "folder", "document", "image", "video"];

const TABS_STORAGE_KEY = "rigintel.files.visibleTabs";

export type SizeFilter = "any" | "lt1" | "1to10" | "10to100" | "gt100";
export type SortKey = "name" | "date" | "size" | "type";
export type SortDir = "asc" | "desc";

export type FileViewFilters = {
  kind: FileKind;
  dateFrom: string;
  dateTo: string;
  size: SizeFilter;
  sort: SortKey;
  dir: SortDir;
};

export const EMPTY_FILTERS: FileViewFilters = {
  kind: "all",
  dateFrom: "",
  dateTo: "",
  size: "any",
  sort: "date",
  dir: "desc",
};

const MB = 1024 * 1024;

export function classifyFile(file: Pick<WorkspaceFile, "name" | "mime_type">): Exclude<FileKind, "all" | "folder"> {
  const mime = (file.mime_type || "").toLowerCase();
  const ext = extOf(file.name);
  if (mime.startsWith("image/") || ["jpg", "jpeg", "png", "gif", "webp", "svg", "bmp", "heic", "tif", "tiff"].includes(ext)) {
    return "image";
  }
  if (mime.startsWith("video/") || ["mp4", "webm", "mov", "mkv", "avi", "m4v"].includes(ext)) {
    return "video";
  }
  if (mime.startsWith("audio/") || ["mp3", "wav", "m4a", "ogg", "flac", "aac"].includes(ext)) {
    return "audio";
  }
  if (
    mime.includes("zip") ||
    mime.includes("rar") ||
    mime.includes("7z") ||
    mime.includes("tar") ||
    mime.includes("gzip") ||
    ["zip", "rar", "7z", "tar", "gz"].includes(ext)
  ) {
    return "archive";
  }
  if (
    mime.includes("pdf") ||
    mime.includes("word") ||
    mime.includes("excel") ||
    mime.includes("spreadsheet") ||
    mime.includes("presentation") ||
    mime.includes("msword") ||
    mime.startsWith("text/") ||
    ["pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "rtf", "odt", "ods", "csv"].includes(ext)
  ) {
    return "document";
  }
  return "other";
}

export function kindLabel(kind: Exclude<FileKind, "all">): string {
  return FILE_KIND_TABS.find((t) => t.id === kind)?.label ?? "Файл";
}

export function loadVisibleTabs(): FileKind[] {
  if (typeof window === "undefined") return DEFAULT_VISIBLE_TABS;
  try {
    const raw = window.localStorage.getItem(TABS_STORAGE_KEY);
    if (!raw) return DEFAULT_VISIBLE_TABS;
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return DEFAULT_VISIBLE_TABS;
    const allowed = new Set(FILE_KIND_TABS.map((t) => t.id));
    const next = parsed.filter((id): id is FileKind => typeof id === "string" && allowed.has(id as FileKind));
    if (!next.includes("all")) next.unshift("all");
    return next.length ? next : DEFAULT_VISIBLE_TABS;
  } catch {
    return DEFAULT_VISIBLE_TABS;
  }
}

export function saveVisibleTabs(tabs: FileKind[]) {
  if (typeof window === "undefined") return;
  const next = tabs.includes("all") ? tabs : (["all", ...tabs] as FileKind[]);
  window.localStorage.setItem(TABS_STORAGE_KEY, JSON.stringify(next));
}

export function matchesDate(iso: string, from: string, to: string): boolean {
  if (!from && !to) return true;
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return false;
  if (from) {
    const start = new Date(`${from}T00:00:00`);
    if (t < start.getTime()) return false;
  }
  if (to) {
    const end = new Date(`${to}T23:59:59.999`);
    if (t > end.getTime()) return false;
  }
  return true;
}

export function matchesSize(bytes: number, size: SizeFilter): boolean {
  switch (size) {
    case "lt1":
      return bytes < MB;
    case "1to10":
      return bytes >= MB && bytes < 10 * MB;
    case "10to100":
      return bytes >= 10 * MB && bytes < 100 * MB;
    case "gt100":
      return bytes >= 100 * MB;
    default:
      return true;
  }
}

export function filterFolders(folders: WorkspaceFolder[], filters: FileViewFilters): WorkspaceFolder[] {
  if (filters.kind !== "all" && filters.kind !== "folder") return [];
  if (filters.size !== "any") return [];
  return folders.filter((fo) => matchesDate(fo.created_at, filters.dateFrom, filters.dateTo));
}

export function filterFiles(files: WorkspaceFile[], filters: FileViewFilters): WorkspaceFile[] {
  if (filters.kind === "folder") return [];
  return files.filter((f) => {
    if (filters.kind !== "all" && classifyFile(f) !== filters.kind) return false;
    if (!matchesDate(f.created_at, filters.dateFrom, filters.dateTo)) return false;
    if (!matchesSize(f.size, filters.size)) return false;
    return true;
  });
}

export function sortFiles(files: WorkspaceFile[], filters: FileViewFilters): WorkspaceFile[] {
  const mul = filters.dir === "asc" ? 1 : -1;
  return [...files].sort((a, b) => {
    let cmp = 0;
    switch (filters.sort) {
      case "name":
        cmp = a.name.localeCompare(b.name, "ru", { sensitivity: "base" });
        break;
      case "size":
        cmp = a.size - b.size;
        break;
      case "type":
        cmp = classifyFile(a).localeCompare(classifyFile(b));
        break;
      default:
        cmp = new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
    }
    return cmp * mul;
  });
}

export function sortFolders(folders: WorkspaceFolder[], filters: FileViewFilters): WorkspaceFolder[] {
  const mul = filters.dir === "asc" ? 1 : -1;
  return [...folders].sort((a, b) => {
    if (filters.sort === "name") {
      return a.name.localeCompare(b.name, "ru", { sensitivity: "base" }) * mul;
    }
    return (new Date(a.created_at).getTime() - new Date(b.created_at).getTime()) * mul;
  });
}

export function filtersActive(filters: FileViewFilters): boolean {
  return Boolean(filters.dateFrom || filters.dateTo || filters.size !== "any" || filters.kind !== "all");
}

function extOf(name: string) {
  const i = name.lastIndexOf(".");
  if (i <= 0 || i === name.length - 1) return "";
  return name.slice(i + 1).toLowerCase();
}
