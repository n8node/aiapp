"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  fetchAdminUsers,
  setAdminUserBlocked,
  setAdminUserStudioAccess,
  type User,
} from "@/lib/api";

export function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [q, setQ] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchAdminUsers({ q: q.trim() || undefined });
      setUsers(data.data.users ?? []);
      setTotal(data.data.total);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось загрузить пользователей");
    } finally {
      setLoading(false);
    }
  }, [q]);

  useEffect(() => {
    const t = window.setTimeout(() => {
      void load();
    }, 200);
    return () => window.clearTimeout(t);
  }, [load]);

  async function toggleStudio(user: User) {
    try {
      await setAdminUserStudioAccess(user.id, !user.studio_access);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось изменить доступ к Studio");
    }
  }

  async function toggleBlock(user: User) {
    try {
      await setAdminUserBlocked(user.id, !user.is_blocked);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось изменить статус");
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Пользователи</h1>
        <p className="text-sm text-slate-500">Всего: {total}</p>
      </div>
      <input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Поиск по email или имени"
        className="w-full max-w-sm rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
      />
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase text-slate-500">
            <tr>
              <th className="px-4 py-2">Пользователь</th>
              <th className="px-4 py-2">2FA</th>
              <th className="px-4 py-2">Роль</th>
              <th className="px-4 py-2">Studio</th>
              <th className="px-4 py-2">Статус</th>
              <th className="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td className="px-4 py-6 text-slate-500" colSpan={6}>
                  Загрузка…
                </td>
              </tr>
            ) : (
              users.map((u) => (
                <tr key={u.id} className="border-t border-slate-100">
                  <td className="px-4 py-3">
                    <p className="font-medium">{u.name || "—"}</p>
                    <p className="text-xs text-slate-500">{u.email}</p>
                  </td>
                  <td className="px-4 py-3">{u.totp_enabled ? "Вкл" : "Нет"}</td>
                  <td className="px-4 py-3">{u.is_platform_admin ? "Суперадмин" : "Пользователь"}</td>
                  <td className="px-4 py-3">
                    {u.is_platform_admin ? (
                      "Всегда"
                    ) : (
                      <button
                        type="button"
                        onClick={() => void toggleStudio(u)}
                        className="rounded-lg border border-slate-200 px-3 py-1 text-xs"
                      >
                        {u.studio_access ? "Есть" : "Нет"}
                      </button>
                    )}
                  </td>
                  <td className="px-4 py-3">{u.is_blocked ? "Блок" : "Активен"}</td>
                  <td className="px-4 py-3 text-right">
                    {!u.is_platform_admin && (
                      <button
                        type="button"
                        onClick={() => void toggleBlock(u)}
                        className="rounded-lg border border-slate-200 px-3 py-1 text-xs"
                      >
                        {u.is_blocked ? "Разблокировать" : "Заблокировать"}
                      </button>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
