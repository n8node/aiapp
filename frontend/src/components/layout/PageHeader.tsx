import Link from "next/link";
import { ChevronRight } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export type Crumb = {
  label: string;
  href?: string;
  onClick?: () => void;
};

export function PageHeader({
  title,
  crumbs,
  actions,
  description,
  className,
}: {
  title: string;
  crumbs?: Crumb[];
  description?: string;
  actions?: ReactNode;
  className?: string;
}) {
  const trail =
    crumbs && crumbs.length > 0
      ? crumbs
      : [{ label: "Главная", href: "/dashboard" }, { label: title }];

  return (
    <div className={cn("mb-6 flex flex-wrap items-start justify-between gap-4", className)}>
      <div className="min-w-0 space-y-1">
        <nav className="flex flex-wrap items-center gap-1 text-sm text-muted">
          {trail.map((crumb, i) => {
            const last = i === trail.length - 1;
            return (
              <span key={`${crumb.label}-${i}`} className="flex items-center gap-1">
                {i > 0 && <ChevronRight className="h-3.5 w-3.5 shrink-0 opacity-60" />}
                {crumb.href && !last ? (
                  <Link href={crumb.href} className="hover:text-text">
                    {crumb.label}
                  </Link>
                ) : crumb.onClick && !last ? (
                  <button type="button" onClick={crumb.onClick} className="hover:text-text">
                    {crumb.label}
                  </button>
                ) : (
                  <span className={last ? "text-text" : undefined}>{crumb.label}</span>
                )}
              </span>
            );
          })}
        </nav>
        <h1 className="text-2xl font-semibold tracking-tight text-text">{title}</h1>
        {description ? <p className="max-w-2xl text-sm text-muted">{description}</p> : null}
      </div>
      {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}
