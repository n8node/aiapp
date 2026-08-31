"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Settings2 } from "lucide-react";
import { cn } from "@/lib/cn";
import {
  FILE_KIND_TABS,
  type FileKind,
  type FileViewFilters,
  type SizeFilter,
  type SortKey,
} from "@/lib/file-view";

type Props = {
  filters: FileViewFilters;
  visibleTabs: FileKind[];
  hideKindTabs?: boolean;
  onChange: (next: FileViewFilters) => void;
  onVisibleTabsChange: (tabs: FileKind[]) => void;
};

const SIZE_OPTIONS: { id: SizeFilter; label: string }[] = [
  { id: "any", label: "Любой размер" },
  { id: "lt1", label: "До 1 МБ" },
  { id: "1to10", label: "1–10 МБ" },
  { id: "10to100", label: "10–100 МБ" },
  { id: "gt100", label: "Больше 100 МБ" },
];

const SORT_OPTIONS: { id: SortKey; label: string }[] = [
  { id: "date", label: "По дате" },
  { id: "name", label: "По имени" },
  { id: "size", label: "По размеру" },
  { id: "type", label: "По типу" },
];

export function FileToolbar({
  filters,
  visibleTabs,
  hideKindTabs = false,
  onChange,
  onVisibleTabsChange,
}: Props) {
  const [tabMenu, setTabMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!tabMenu) return;
    const onDoc = (e: MouseEvent) => {
      if (!menuRef.current?.contains(e.target as Node)) setTabMenu(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [tabMenu]);

  const shownTabs = FILE_KIND_TABS.filter((t) => visibleTabs.includes(t.id));

  return (
    <div className="mb-4 space-y-3">
      {!hideKindTabs ? (
        <div className="flex items-center gap-2">
          <div className="flex min-w-0 flex-1 gap-1 overflow-x-auto">
            {shownTabs.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => onChange({ ...filters, kind: tab.id })}
                className={cn(
                  "shrink-0 rounded-lg px-3 py-1.5 text-sm",
                  filters.kind === tab.id
                    ? "bg-zinc-100 font-medium text-text"
                    : "text-muted hover:bg-zinc-50",
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>
          <div className="relative shrink-0" ref={menuRef}>
            <button
              type="button"
              title="Настроить вкладки"
              onClick={() => setTabMenu((v) => !v)}
              className="rounded-lg border border-border bg-surface p-2 text-muted hover:bg-zinc-50"
            >
              <Settings2 className="h-4 w-4" />
            </button>
            {tabMenu ? (
              <div className="absolute right-0 z-30 mt-1 w-56 rounded-xl border border-border bg-surface p-2 shadow-sm">
                <p className="px-2 py-1 text-xs font-semibold uppercase tracking-wide text-muted">Вкладки</p>
                {FILE_KIND_TABS.map((tab) => {
                  const on = visibleTabs.includes(tab.id);
                  const locked = tab.id === "all";
                  return (
                    <button
                      key={tab.id}
                      type="button"
                      disabled={locked}
                      onClick={() => {
                        if (locked) return;
                        const next = on
                          ? visibleTabs.filter((id) => id !== tab.id)
                          : [...visibleTabs, tab.id];
                        onVisibleTabsChange(next);
                        if (on && filters.kind === tab.id) {
                          onChange({ ...filters, kind: "all" });
                        }
                      }}
                      className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-zinc-50 disabled:opacity-60"
                    >
                      <span
                        className={cn(
                          "flex h-4 w-4 items-center justify-center rounded border",
                          on ? "border-accent bg-accent text-white" : "border-zinc-300",
                        )}
                      >
                        {on ? <Check className="h-3 w-3" /> : null}
                      </span>
                      {tab.label}
                    </button>
                  );
                })}
              </div>
            ) : null}
          </div>
        </div>
      ) : null}

      <div className="flex flex-wrap items-end gap-2">
        <label className="text-xs text-muted">
          С
          <input
            type="date"
            value={filters.dateFrom}
            onChange={(e) => onChange({ ...filters, dateFrom: e.target.value })}
            className="mt-1 block rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-text"
          />
        </label>
        <label className="text-xs text-muted">
          По
          <input
            type="date"
            value={filters.dateTo}
            onChange={(e) => onChange({ ...filters, dateTo: e.target.value })}
            className="mt-1 block rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-text"
          />
        </label>
        <label className="text-xs text-muted">
          Размер
          <select
            value={filters.size}
            onChange={(e) => onChange({ ...filters, size: e.target.value as SizeFilter })}
            className="mt-1 block rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-text"
          >
            {SIZE_OPTIONS.map((o) => (
              <option key={o.id} value={o.id}>
                {o.label}
              </option>
            ))}
          </select>
        </label>
        {!hideKindTabs ? (
          <label className="text-xs text-muted">
            Тип
            <select
              value={filters.kind}
              onChange={(e) => onChange({ ...filters, kind: e.target.value as FileKind })}
              className="mt-1 block rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-text"
            >
              {FILE_KIND_TABS.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
        ) : null}
        <label className="text-xs text-muted">
          Сортировка
          <select
            value={filters.sort}
            onChange={(e) => onChange({ ...filters, sort: e.target.value as SortKey })}
            className="mt-1 block rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-text"
          >
            {SORT_OPTIONS.map((o) => (
              <option key={o.id} value={o.id}>
                {o.label}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          onClick={() => onChange({ ...filters, dir: filters.dir === "asc" ? "desc" : "asc" })}
          className="rounded-lg border border-border bg-surface px-3 py-1.5 text-sm hover:bg-zinc-50"
        >
          {filters.dir === "asc" ? "По возрастанию" : "По убыванию"}
        </button>
      </div>
    </div>
  );
}
