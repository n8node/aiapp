"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { logout } from "@/lib/api";
import { cn } from "@/lib/cn";

const nav = [
  {
    label: "Платформа",
    items: [
      { href: "/admin/users", label: "Пользователи" },
      { href: "/admin/invites", label: "Инвайт-ключи" },
      { href: "/admin/auth-settings", label: "Домены почты" },
      { href: "/admin/locale", label: "Язык интерфейса" },
      { href: "/admin/bitrix", label: "Битрикс24" },
      { href: "/admin/workspaces", label: "Пространства" },
      { href: "/admin/storage-settings", label: "S3 — хранилище" },
      { href: "/admin/outbound-proxy", label: "Исходящий прокси" },
    ],
  },
  {
    label: "Модели",
    items: [
      { href: "/admin/models", label: "Реестр" },
      { href: "/admin/training", label: "Заявки на обучение" },
      { href: "/admin/unsloth", label: "Unsloth Studio" },
    ],
  },
];

export function AdminShell({
  adminEmail,
  adminName,
  children,
}: {
  adminEmail: string;
  adminName: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();

  async function handleLogout() {
    await logout();
    router.push("/auth/login");
    router.refresh();
  }

  return (
    <div className="flex min-h-screen bg-[#eef1f6] text-slate-900">
      <aside className="flex w-60 shrink-0 flex-col bg-[#0b1220] text-slate-200">
        <div className="border-b border-white/10 px-5 py-5">
          <Link href="/admin/users" className="text-lg font-semibold tracking-wide text-white">
            RIGINTEL
          </Link>
          <span className="mt-2 inline-flex rounded bg-blue-500/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-blue-300">
            Superadmin
          </span>
        </div>
        <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-5">
          {nav.map((section) => (
            <div key={section.label}>
              <p className="mb-2 px-2 text-[11px] font-semibold uppercase tracking-wider text-slate-500">
                {section.label}
              </p>
              <ul className="space-y-0.5">
                {section.items.map((item) => {
                  const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
                  return (
                    <li key={item.href}>
                      <Link
                        href={item.href}
                        className={cn(
                          "block rounded-md px-2.5 py-2 text-sm",
                          active ? "bg-blue-600 text-white" : "text-slate-300 hover:bg-white/5 hover:text-white",
                        )}
                      >
                        {item.label}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </nav>
        <div className="border-t border-white/10 px-4 py-4">
          <p className="truncate text-sm font-medium text-white">{adminName || adminEmail}</p>
          <p className="truncate text-xs text-slate-400">{adminEmail}</p>
          <div className="mt-3 flex gap-3 text-xs">
            <Link href="/dashboard" className="text-blue-300 hover:underline">
              В приложение
            </Link>
            <button type="button" onClick={handleLogout} className="text-slate-400 hover:text-white">
              Выйти
            </button>
          </div>
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-slate-200/80 bg-white/70 px-6 py-3 backdrop-blur">
          <p className="text-sm text-slate-500">Администрирование</p>
        </header>
        <main className="flex-1 px-6 py-6">{children}</main>
      </div>
    </div>
  );
}
