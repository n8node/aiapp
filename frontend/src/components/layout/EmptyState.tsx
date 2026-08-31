import type { ReactNode } from "react";

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-start rounded-xl border border-dashed border-border bg-surface px-6 py-10">
      <h2 className="text-base font-semibold text-text">{title}</h2>
      <p className="mt-2 max-w-lg text-sm text-muted">{description}</p>
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}
