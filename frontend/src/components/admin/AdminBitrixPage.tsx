"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ApiError,
  createWorkspaceFromDepartment,
  disconnectBitrix,
  fetchBitrixDepartments,
  fetchBitrixStatus,
  fetchBitrixUsers,
  saveBitrixWebhook,
  syncBitrix,
  testBitrix,
  type BitrixDepartment,
  type BitrixStatus,
  type BitrixUser,
} from "@/lib/api";

function formatWhen(iso?: string | null) {
  if (!iso) return "ещё не было";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "ещё не было";
  return d.toLocaleString("ru-RU");
}

export function AdminBitrixPage() {
  const [status, setStatus] = useState<BitrixStatus | null>(null);
  const [webhook, setWebhook] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState<"save" | "test" | "sync" | "disconnect" | "workspace" | null>(null);
  const [includeChildren, setIncludeChildren] = useState(true);
  const [departments, setDepartments] = useState<BitrixDepartment[]>([]);
  const [users, setUsers] = useState<BitrixUser[]>([]);
  const [userQ, setUserQ] = useState("");

  const namesById = useMemo(() => {
    const m = new Map<number, string>();
    for (const d of departments) m.set(d.bitrix_id, d.name);
    return m;
  }, [departments]);

  const loadStatus = useCallback(async () => {
    const res = await fetchBitrixStatus();
    setStatus(res.data);
  }, []);

  const loadSnapshot = useCallback(async (q?: string) => {
    const [deps, people] = await Promise.all([
      fetchBitrixDepartments(),
      fetchBitrixUsers(q ?? ""),
    ]);
    setDepartments(deps.data.departments ?? []);
    setUsers(people.data.users ?? []);
  }, []);

  useEffect(() => {
    void loadStatus()
      .then(() => loadSnapshot())
      .catch((e) => setError(e instanceof ApiError ? e.message : "Не удалось загрузить Битрикс"));
  }, [loadStatus, loadSnapshot]);

  useEffect(() => {
    if (!status?.configured) return;
    const t = window.setTimeout(() => {
      void loadSnapshot(userQ).catch(() => undefined);
    }, 250);
    return () => window.clearTimeout(t);
  }, [userQ, loadSnapshot, status?.configured]);

  async function save() {
    setError(null);
    setNotice(null);
    setBusy("save");
    try {
      const res = await saveBitrixWebhook(webhook.trim());
      setStatus(res.data);
      setWebhook("");
      setNotice("Вебхук сохранён. Секрет больше не показывается.");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось сохранить");
    } finally {
      setBusy(null);
    }
  }

  async function disconnect() {
    if (!window.confirm("Отключить Битрикс24? Сохранённая структура останется до следующей синхронизации.")) {
      return;
    }
    setError(null);
    setBusy("disconnect");
    try {
      setStatus((await disconnectBitrix()).data);
      setNotice("Подключение отключено");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось отключить");
    } finally {
      setBusy(null);
    }
  }

  async function test() {
    setError(null);
    setNotice(null);
    setBusy("test");
    try {
      await testBitrix();
      setNotice("Подключение работает");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Проверка не удалась");
    } finally {
      setBusy(null);
    }
  }

  async function syncNow() {
    setError(null);
    setNotice(null);
    setBusy("sync");
    try {
      const res = await syncBitrix();
      let msg = `Синхронизация завершена: ${res.data.departments} отделов, ${res.data.users} сотрудников`;
      const m = res.data.memberships;
      if (m) {
        msg += `. Членства: +${m.added}, без учётки ${m.unmatched}`;
      }
      setNotice(msg);
      await loadStatus();
      await loadSnapshot(userQ);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Синхронизация не удалась");
      void loadStatus();
    } finally {
      setBusy(null);
    }
  }

  async function createWorkspace(dept: BitrixDepartment) {
    setError(null);
    setNotice(null);
    setBusy("workspace");
    try {
      const res = await createWorkspaceFromDepartment(dept.bitrix_id, includeChildren, dept.name);
      setNotice(`Создано пространство «${res.data.name}»`);
      await loadSnapshot(userQ);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось создать пространство");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Битрикс24</h1>
        <p className="text-sm text-slate-500">
          Только чтение оргструктуры. Синхронизацию можно запускать в любой
          момент, когда в Битриксе что-то изменилось.
        </p>
      </div>

      <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 className="text-base font-semibold">Подключение</h2>
        <p className="mt-1 text-sm text-slate-500">
          Вставьте входящий вебхук целиком. Он хранится в зашифрованном виде и
          в списке виден только маской.
        </p>
        {status?.configured ? (
          <p className="mt-3 text-sm">
            Портал: <span className="font-medium">{status.portal_host}</span>
            <span className="mx-2 text-slate-300">·</span>
            <span className="font-mono text-xs text-slate-600">{status.webhook_masked}</span>
          </p>
        ) : (
          <p className="mt-3 text-sm text-slate-500">Вебхук ещё не задан</p>
        )}
        <div className="mt-3 flex flex-wrap gap-2">
          <input
            type="password"
            autoComplete="off"
            value={webhook}
            onChange={(e) => setWebhook(e.target.value)}
            placeholder="https://bitrix.rigintel.ai/rest/1/секрет/"
            className="min-w-[20rem] flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 font-mono text-sm"
          />
          <button
            type="button"
            disabled={busy !== null || !webhook.trim()}
            onClick={() => void save()}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-40"
          >
            {busy === "save" ? "Сохранение…" : "Сохранить"}
          </button>
          {status?.configured ? (
            <button
              type="button"
              disabled={busy !== null}
              onClick={() => void disconnect()}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm"
            >
              Отключить
            </button>
          ) : null}
        </div>
      </section>

      <section className="rounded-xl border border-slate-200 bg-slate-50 p-5">
        <h2 className="text-base font-semibold">Где взять вебхук в Битрикс24</h2>
        <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm text-slate-700">
          <li>Войдите в портал администратором (облако или коробка, например bitrix.rigintel.ai).</li>
          <li>
            Откройте <span className="font-medium">Приложения</span> в левом меню
            (иконка сетки) → <span className="font-medium">Разработчикам</span>.
          </li>
          <li>
            В блоке «Другое» создайте <span className="font-medium">Входящий вебхук</span>
            {" "}(Inbound webhook). Название можно поставить <span className="font-mono">RigIntel</span>.
          </li>
          <li>
            В правах вебхука отметьте оба пункта и сохраните вебхук в Битриксе:{" "}
            <span className="font-medium">Пользователи</span> (<span className="font-mono">user</span>) и{" "}
            <span className="font-medium">Структура компании</span>{" "}
            (<span className="font-mono">department</span>). Без второго права отделы не подтянутся.
            CRM, диск и задачи не включайте.
          </li>
          <li>Скопируйте <span className="font-medium">URL вебхука</span> целиком после сохранения прав. Секрет уже есть в пути.</li>
          <li>Вставьте URL сюда → «Сохранить» → «Проверить» → «Синхронизировать сейчас».</li>
        </ol>
        <p className="mt-3 text-sm text-slate-600">
          В коробке путь может быть такой:{" "}
          <span className="font-medium">Настройки → Настройки продукта → Настройки модулей → REST API</span>
          {" "}→ входящие вебхуки. Если пункта «Разработчикам» нет, включите модуль REST.
        </p>
        <p className="mt-2 text-sm">
          <a
            href="https://apidocs.bitrix24.ru/local-integrations/local-webhooks.html"
            target="_blank"
            rel="noreferrer"
            className="text-blue-700 hover:underline"
          >
            Справка Bitrix24: входящие вебхуки
          </a>
        </p>
      </section>

      <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">Синхронизация</h2>
            <p className="text-sm text-slate-500">
              Последний запуск: {formatWhen(status?.last_sync_at)}
              {status?.last_sync_status ? ` · ${status.last_sync_status}` : ""}
            </p>
            {status?.last_sync_error ? (
              <p className="mt-1 text-sm text-red-700">{status.last_sync_error}</p>
            ) : null}
            <p className="mt-1 text-sm text-slate-500">
              В снимке: {status?.departments_count ?? 0} отделов, {status?.users_count ?? 0} сотрудников
            </p>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={busy !== null || !status?.configured}
              onClick={() => void test()}
              className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm"
            >
              {busy === "test" ? "Проверка…" : "Проверить"}
            </button>
            <button
              type="button"
              disabled={busy !== null || !status?.configured}
              onClick={() => void syncNow()}
              className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-40"
            >
              {busy === "sync" ? "Синхронизация…" : "Синхронизировать сейчас"}
            </button>
          </div>
        </div>
      </section>

      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      {notice ? <p className="text-sm text-emerald-700">{notice}</p> : null}

      <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-5 py-3">
          <h2 className="text-base font-semibold">Отделы из Битрикс</h2>
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input
              type="checkbox"
              checked={includeChildren}
              onChange={(e) => setIncludeChildren(e.target.checked)}
            />
            Включать подотделы
          </label>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs uppercase text-slate-500">
              <tr>
                <th className="px-4 py-2">Отдел</th>
                <th className="px-4 py-2">Родитель</th>
                <th className="px-4 py-2">Пространство</th>
                <th className="px-4 py-2" />
              </tr>
            </thead>
            <tbody>
              {departments.length === 0 ? (
                <tr>
                  <td className="px-4 py-6 text-slate-500" colSpan={4}>
                    Пока пусто — запустите синхронизацию
                  </td>
                </tr>
              ) : (
                departments.map((d) => (
                  <tr key={d.bitrix_id} className="border-t border-slate-100">
                    <td className="px-4 py-3 font-medium">{d.name}</td>
                    <td className="px-4 py-3 text-slate-500">
                      {d.parent_bitrix_id ? namesById.get(d.parent_bitrix_id) || d.parent_bitrix_id : "—"}
                    </td>
                    <td className="px-4 py-3 text-slate-600">
                      {d.workspace_name || "—"}
                      {d.workspace_id && d.include_descendants ? (
                        <span className="ml-1 text-xs text-slate-400">+ подотделы</span>
                      ) : null}
                    </td>
                    <td className="px-4 py-3 text-right">
                      {d.workspace_id ? null : (
                        <button
                          type="button"
                          disabled={busy !== null}
                          onClick={() => void createWorkspace(d)}
                          className="text-sm text-blue-700 hover:underline disabled:opacity-40"
                        >
                          Создать пространство
                        </button>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-5 py-3">
          <h2 className="text-base font-semibold">Сотрудники из Битрикс</h2>
          <input
            value={userQ}
            onChange={(e) => setUserQ(e.target.value)}
            placeholder="Поиск по имени или email"
            className="w-full max-w-xs rounded-lg border border-slate-200 px-3 py-2 text-sm"
          />
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs uppercase text-slate-500">
              <tr>
                <th className="px-4 py-2">Сотрудник</th>
                <th className="px-4 py-2">Email</th>
                <th className="px-4 py-2">Отделы</th>
                <th className="px-4 py-2">Статус</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 ? (
                <tr>
                  <td className="px-4 py-6 text-slate-500" colSpan={4}>
                    Нет записей
                  </td>
                </tr>
              ) : (
                users.map((u) => (
                  <tr key={u.bitrix_id} className="border-t border-slate-100">
                    <td className="px-4 py-3">
                      {[u.last_name, u.name].filter(Boolean).join(" ") || "—"}
                    </td>
                    <td className="px-4 py-3">{u.email || "—"}</td>
                    <td className="px-4 py-3 text-slate-500">
                      {(u.department_ids ?? []).length
                        ? (u.department_ids ?? []).map((id) => namesById.get(id) || String(id)).join(", ")
                        : "—"}
                    </td>
                    <td className="px-4 py-3">{u.active ? "Активен" : "Отключён"}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
