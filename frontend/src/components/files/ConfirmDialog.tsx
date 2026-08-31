"use client";

import { X } from "lucide-react";

type Props = {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  danger?: boolean;
  submitting?: boolean;
  onClose: () => void;
  onConfirm: () => void | Promise<void>;
};

export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  danger = false,
  submitting = false,
  onClose,
  onConfirm,
}: Props) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <button type="button" className="absolute inset-0 bg-black/40" aria-label="Закрыть" onClick={onClose} />
      <div className="relative z-10 w-full max-w-md overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
        <div className="flex items-center justify-between border-b border-border px-5 py-4">
          <h2 className="text-lg font-semibold">{title}</h2>
          <button type="button" onClick={onClose} className="rounded-lg p-2 hover:bg-zinc-100">
            <X className="h-5 w-5" />
          </button>
        </div>
        <p className="px-5 py-4 text-sm text-muted">{description}</p>
        <div className="flex justify-end gap-2 border-t border-border px-5 py-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border bg-surface px-4 py-2 text-sm hover:bg-zinc-50"
          >
            Отмена
          </button>
          <button
            type="button"
            disabled={submitting}
            onClick={() => void onConfirm()}
            className={
              danger
                ? "rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
                : "rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent/90 disabled:opacity-50"
            }
          >
            {submitting ? "…" : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
