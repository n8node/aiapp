"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  applyWorkspaceMemberships,
  fetchAdminWorkspaces,
  unlinkWorkspaceDepartment,
  type AdminWorkspace,
  type MembershipApplyResult,
} from "@/lib/api";

export function AdminWorkspacesPage() {
  const [items, setItems] = useState<AdminWorkspace[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    const res = await fetchAdminWorkspaces();
    setItems(res.data.workspaces ?? []);
  }, []);

  useEffect(() => {
    void load().catch((e) => setError(e instanceof ApiError ? e.message : "Не удалось загрузить пространства"));
  }, [load]);

  async function apply() {
    setError(null);
    setNotice(null);
    setBusy(true);
    try {
      const res = await applyWorkspaceMemberships();
      setNotice(formatApply(res.data));
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось обновить членства");
    } finally {
      setBusy(false);
    }
  }

  async function unlink(ws: AdminWorkspace, deptID: number) {
    setError(null);
    try {
      await unlinkWorkspaceDepartment(ws.id, deptID);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось отвязать отдел");
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Пространства</h1>
          <p className="text-sm text-slate-500">
            Отдел Битрикс становится workspace. Членства обновляются по email
            уже зарегистрированных пользователей. Новые учётки из Битрикс не
            создаются.
          </p>
        </div>
        <button
          type="button"
          disabled={busy}
          onClick={() => void apply()}
          className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-40"
        >
          {busy ? "Обновление…" : "Обновить членства"}
        </button>
      </div>
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      {notice ? <p className="text-sm text-emerald-700">{notice}</p> : null}
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase text-slate-500">
            <tr>
              <th className="px-4 py-2">Пространство</th>
              <th className="px-4 py-2">Отделы Битрикс</th>
              <th className="px-4 py-2">Участники</th>
            </tr>
          </thead>
          <tbody>
            {items.length === 0 ? (
              <tr>
                <td className="px-4 py-6 text-slate-500" colSpan={3}>
                  Пока нет пространств. Создайте их из отделов на странице Битрикс24.
                </td>
              </tr>
            ) : (
              items.map((ws) => (
                <tr key={ws.id} className="border-t border-slate-100 align-top">
                  <td className="px-4 py-3">
                    <p className="font-medium">{ws.name}</p>
                    <p className="font-mono text-xs text-slate-500">{ws.slug}</p>
                  </td>
                  <td className="px-4 py-3">
                    {(ws.bitrix_departments ?? []).length === 0 ? (
                      <span className="text-slate-500">Не привязано</span>
                    ) : (
                      <ul className="space-y-1">
                        {ws.bitrix_departments.map((d) => (
                          <li key={d.bitrix_department_id} className="flex flex-wrap items-center gap-2">
                            <span>{d.department_name || d.bitrix_department_id}</span>
                            {d.include_descendants ? (
                              <span className="text-xs text-slate-500">+ подотделы</span>
                            ) : null}
                            <button
                              type="button"
                              className="text-xs text-slate-500 hover:text-slate-800"
                              onClick={() => void unlink(ws, d.bitrix_department_id)}
                            >
                              Отвязать
                            </button>
                          </li>
                        ))}
                      </ul>
                    )}
                  </td>
                  <td className="px-4 py-3">{ws.member_count}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function formatApply(r: MembershipApplyResult) {
  return `Членства: совпало ${r.matched}, добавлено ${r.added}, обновлено ${r.updated}, снято ${r.removed}, без учётки ${r.unmatched}`;
}
