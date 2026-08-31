"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { BookOpen, FileText, HardDrive, LayoutDashboard, LogOut, Settings } from "lucide-react";
import { useState } from "react";
import { logout, switchWorkspace, type User, type Workspace } from "@/lib/api";
import { cn } from "@/lib/cn";

const nav = [
  { href: "/dashboard", label: "Обзор", icon: LayoutDashboard },
  { href: "/files", label: "Файлы", icon: HardDrive },
  { href: "/documents", label: "Документы", icon: FileText },
  { href: "/knowledge", label: "Базы знаний", icon: BookOpen },
  { href: "/settings", label: "Настройки", icon: Settings },
];

export function AppShell({
  user,
  workspace,
  workspaces = [],
  children,
}: {
  user: User;
  workspace?: Workspace | null;
  workspaces?: Workspace[];
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const [collapsed, setCollapsed] = useState(false);

  async function handleLogout() {
    await logout();
    router.push("/auth/login");
    router.refresh();
  }

  async function handleWorkspace(id: string) {
    if (!id || id === workspace?.id) return;
    await switchWorkspace(id);
    router.refresh();
  }

  return (
    <div className="flex min-h-screen bg-bg text-text">
      <aside
        className={cn(
          "sticky top-0 flex h-screen shrink-0 flex-col border-r border-border bg-surface",
          collapsed ? "w-[4.25rem]" : "w-60",
        )}
      >
        <div className="flex h-14 items-center justify-between border-b border-border px-4">
          {!collapsed && (
            <Link href="/dashboard" className="text-base font-semibold tracking-tight">
              RigIntel
            </Link>
          )}
          <button
            type="button"
            onClick={() => setCollapsed((v) => !v)}
            className="rounded-md p-1.5 text-muted hover:bg-zinc-100"
            aria-label="Свернуть меню"
          >
            {collapsed ? "»" : "«"}
          </button>
        </div>
        {workspaces.length > 0 && !collapsed ? (
          <div className="border-b border-border px-3 py-3">
            <label className="mb-1 block text-[11px] font-medium uppercase tracking-wide text-muted">
              Пространство
            </label>
            <select
              value={workspace?.id ?? ""}
              onChange={(e) => void handleWorkspace(e.target.value)}
              className="w-full rounded-lg border border-border bg-surface px-2.5 py-2 text-sm"
            >
              {workspaces.map((ws) => (
                <option key={ws.id} value={ws.id}>
                  {ws.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}
        <nav className="flex flex-1 flex-col gap-1 p-2">
          {nav.map((item) => {
            const Icon = item.icon;
            const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm",
                  active ? "bg-zinc-100 font-medium text-text" : "text-muted hover:bg-zinc-50 hover:text-text",
                  collapsed && "justify-center px-2",
                )}
              >
                <Icon className="h-4 w-4 shrink-0" />
                {!collapsed && item.label}
              </Link>
            );
          })}
        </nav>
        <div className="border-t border-border p-2">
          {user.is_platform_admin && !collapsed && (
            <Link href="/admin/users" className="mb-2 block rounded-md px-2.5 py-1.5 text-xs text-accent hover:underline">
              Админка платформы
            </Link>
          )}
          <div className={cn("flex items-center gap-2 px-2 py-2", collapsed && "justify-center")}>
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-zinc-200 text-xs font-semibold">
              {(user.name || user.email)[0]?.toUpperCase()}
            </div>
            {!collapsed && (
              <>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{user.name || user.email}</p>
                  <p className="truncate text-xs text-muted">{user.email}</p>
                </div>
                <button type="button" onClick={handleLogout} className="rounded-md p-1.5 text-muted hover:bg-zinc-100" aria-label="Выйти">
                  <LogOut className="h-4 w-4" />
                </button>
              </>
            )}
          </div>
        </div>
      </aside>
      <main className="min-w-0 flex-1 overflow-x-clip">
        <div
          className={cn(
            "mx-auto px-4 py-6 sm:px-6 lg:px-8",
            pathname.startsWith("/files") || pathname.startsWith("/documents") || pathname.startsWith("/knowledge") || pathname.startsWith("/settings") || pathname.startsWith("/dashboard")
              ? "max-w-none"
              : "max-w-7xl",
          )}
        >
          {children}
        </div>
      </main>
    </div>
  );
}
