"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  fetchAdminInvites,
  issueAdminInvites,
  revokeAdminInvite,
  type Invite,
} from "@/lib/api";

export function AdminInvitesPage() {
  const [invites, setInvites] = useState<Invite[]>([]);
  const [fresh, setFresh] = useState<Invite[]>([]);
  const [count, setCount] = useState(1);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const data = await fetchAdminInvites();
      setInvites(data.data.invites ?? []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось загрузить ключи");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function issue() {
    setError(null);
    try {
      const data = await issueAdminInvites(count);
      setFresh(data.data.invites ?? []);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось выпустить ключи");
    }
  }

  async function revoke(id: string) {
    await revokeAdminInvite(id);
    await load();
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Инвайт-ключи</h1>
        <p className="text-sm text-slate-500">
          Полный ключ показывается один раз при выпуске. На регистрации список
          ключей не публикуется.
        </p>
      </div>
      <div className="flex items-center gap-2">
        <input
          type="number"
          min={1}
          max={50}
          value={count}
          onChange={(e) => setCount(Number(e.target.value))}
          className="w-24 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
        />
        <button
          type="button"
          onClick={() => void issue()}
          className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white"
        >
          Выпустить
        </button>
      </div>
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      {fresh.length > 0 && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-4">
          <p className="text-sm font-medium">Скопируйте сейчас — повторно ключ не покажем</p>
          <ul className="mt-2 space-y-1 font-mono text-sm">
            {fresh.map((i) => (
              <li key={i.id}>{i.code}</li>
            ))}
          </ul>
        </div>
      )}
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase text-slate-500">
            <tr>
              <th className="px-4 py-2">Префикс</th>
              <th className="px-4 py-2">Статус</th>
              <th className="px-4 py-2">Создан</th>
              <th className="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            {invites.map((i) => (
              <tr key={i.id} className="border-t border-slate-100">
                <td className="px-4 py-3 font-mono">{i.code_prefix}••••</td>
                <td className="px-4 py-3">{i.status}</td>
                <td className="px-4 py-3">{new Date(i.created_at).toLocaleString("ru-RU")}</td>
                <td className="px-4 py-3 text-right">
                  {i.status === "ACTIVE" && (
                    <button
                      type="button"
                      onClick={() => void revoke(i.id)}
                      className="rounded-lg border border-slate-200 px-3 py-1 text-xs"
                    >
                      Отозвать
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
