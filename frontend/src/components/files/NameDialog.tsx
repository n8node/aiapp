"use client";

import { useEffect, useId, useRef, useState } from "react";
import { X } from "lucide-react";

type Props = {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel: string;
  initialValue?: string;
  placeholder?: string;
  submitting?: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: (name: string) => void | Promise<void>;
};

export function NameDialog({
  open,
  title,
  description,
  confirmLabel,
  initialValue = "",
  placeholder = "Название",
  submitting = false,
  error,
  onClose,
  onConfirm,
}: Props) {
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [value, setValue] = useState(initialValue);

  useEffect(() => {
    if (!open) return;
    setValue(initialValue);
    const t = window.setTimeout(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    }, 20);
    return () => window.clearTimeout(t);
  }, [open, initialValue]);

  if (!open) return null;

  const trimmed = value.trim();
  const canSubmit = trimmed.length > 0 && trimmed.length <= 255 && !submitting;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <button type="button" className="absolute inset-0 bg-black/40" aria-label="Закрыть" onClick={onClose} />
      <form
        className="relative z-10 w-full max-w-md overflow-hidden rounded-xl border border-border bg-surface shadow-sm"
        onSubmit={(e) => {
          e.preventDefault();
          if (!canSubmit) return;
          void onConfirm(trimmed);
        }}
      >
        <div className="flex items-center justify-between border-b border-border px-5 py-4">
          <h2 className="text-lg font-semibold">{title}</h2>
          <button type="button" onClick={onClose} className="rounded-lg p-2 hover:bg-zinc-100">
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-3 px-5 py-4">
          {description ? <p className="text-sm text-muted">{description}</p> : null}
          <div>
            <label htmlFor={inputId} className="mb-1.5 block text-sm font-medium">
              Имя
            </label>
            <input
              ref={inputRef}
              id={inputId}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={placeholder}
              maxLength={255}
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm outline-none focus:border-accent"
            />
          </div>
          {error ? (
            <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
          ) : null}
        </div>
        <div className="flex justify-end gap-2 border-t border-border px-5 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border bg-surface px-4 py-2 text-sm hover:bg-zinc-50"
          >
            Отмена
          </button>
          <button
            type="submit"
            disabled={!canSubmit}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent/90 disabled:opacity-50"
          >
            {submitting ? "…" : confirmLabel}
          </button>
        </div>
      </form>
    </div>
  );
}
