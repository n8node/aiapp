"use client";

import { Check, Copy, Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ApiError,
  deleteAdminInvites,
  fetchAdminInvites,
  issueAdminInvites,
  revokeAdminInvite,
  type Invite,
} from "@/lib/api";

function formatDateTime(iso: string | null | undefined) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function statusLabel(status: string) {
  if (status === "ACTIVE") return "Активен";
  if (status === "USED") return "Использован";
  if (status === "REVOKED") return "Отозван";
  return status;
}

export function AdminInvitesPage() {
  const [invites, setInvites] = useState<Invite[]>([]);
  const [total, setTotal] = useState(0);
  const [fresh, setFresh] = useState<Invite[]>([]);
  const [count, setCount] = useState(1);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [createdFrom, setCreatedFrom] = useState("");
  const [createdTo, setCreatedTo] = useState("");
  const [usedFrom, setUsedFrom] = useState("");
  const [usedTo, setUsedTo] = useState("");
  const [usedEmail, setUsedEmail] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchAdminInvites({
        created_from: createdFrom || undefined,
        created_to: createdTo || undefined,
        used_from: usedFrom || undefined,
        used_to: usedTo || undefined,
        used_email: usedEmail.trim() || undefined,
        limit: 500,
      });
      setInvites(data.data.invites ?? []);
      setTotal(data.data.total);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось загрузить ключи");
    } finally {
      setLoading(false);
    }
  }, [createdFrom, createdTo, usedFrom, usedTo, usedEmail]);

  useEffect(() => {
    const t = window.setTimeout(() => void load(), 250);
    return () => window.clearTimeout(t);
  }, [load]);

  useEffect(() => {
    setSelected(new Set());
  }, [createdFrom, createdTo, usedFrom, usedTo, usedEmail]);

  const allSelected = invites.length > 0 && invites.every((i) => selected.has(i.id));

  const selectedCount = useMemo(() => selected.size, [selected]);

  async function copyCode(id: string, code: string) {
    try {
      await navigator.clipboard.writeText(code);
      setCopiedId(id);
      window.setTimeout(() => setCopiedId((cur) => (cur === id ? null : cur)), 1500);
    } catch {
      setError("Не удалось скопировать ключ");
    }
  }

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
    setSelected((prev) => {
      const next = new Set(prev);
      next.delete(id);
      return next;
    });
    await load();
  }

  async function bulkDelete() {
    if (selectedCount === 0) return;
    if (!window.confirm(`Удалить выбранные ключи (${selectedCount})? Это нельзя отменить.`)) {
      return;
    }
    setError(null);
    try {
      await deleteAdminInvites([...selected]);
      setSelected(new Set());
      setFresh((prev) => prev.filter((i) => !selected.has(i.id)));
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось удалить ключи");
    }
  }

  function toggleAll() {
    if (allSelected) {
      setSelected(new Set());
      return;
    }
    setSelected(new Set(invites.map((i) => i.id)));
  }

  function toggleOne(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Инвайт-ключи</h1>
        <p className="text-sm text-slate-500">
          Ключи видны только в админке. На странице регистрации список не
          публикуется.
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <input
          type="number"
          min={1}
          max={200}
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
        <button
          type="button"
          disabled={selectedCount === 0}
          onClick={() => void bulkDelete()}
          className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm disabled:opacity-40"
        >
          <Trash2 className="h-4 w-4" />
          Удалить выбранные{selectedCount > 0 ? ` (${selectedCount})` : ""}
        </button>
      </div>
      <div className="grid gap-3 rounded-xl border border-slate-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-3">
        <label className="text-sm">
          <span className="text-slate-500">Создан с</span>
          <input
            type="date"
            value={createdFrom}
            onChange={(e) => setCreatedFrom(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
          />
        </label>
        <label className="text-sm">
          <span className="text-slate-500">Создан по</span>
          <input
            type="date"
            value={createdTo}
            onChange={(e) => setCreatedTo(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
          />
        </label>
        <label className="text-sm">
          <span className="text-slate-500">Использован с</span>
          <input
            type="date"
            value={usedFrom}
            onChange={(e) => setUsedFrom(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
          />
        </label>
        <label className="text-sm">
          <span className="text-slate-500">Использован по</span>
          <input
            type="date"
            value={usedTo}
            onChange={(e) => setUsedTo(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
          />
        </label>
        <label className="text-sm sm:col-span-2 lg:col-span-1">
          <span className="text-slate-500">Почта использования</span>
          <input
            type="search"
            value={usedEmail}
            onChange={(e) => setUsedEmail(e.target.value)}
            placeholder="email пользователя"
            className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
          />
        </label>
      </div>
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      {fresh.length > 0 && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-4">
          <p className="text-sm font-medium">Только что выпущено</p>
          <ul className="mt-2 space-y-1">
            {fresh.map((i) => (
              <li key={i.id} className="flex items-center gap-2 font-mono text-sm">
                <span>{i.code}</span>
                {i.code ? (
                  <CopyButton
                    copied={copiedId === `fresh-${i.id}`}
                    onCopy={() => void copyCode(`fresh-${i.id}`, i.code!)}
                  />
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      )}
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase text-slate-500">
            <tr>
              <th className="px-4 py-2">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleAll}
                  aria-label="Выбрать все"
                />
              </th>
              <th className="px-4 py-2">Ключ</th>
              <th className="px-4 py-2">Статус</th>
              <th className="px-4 py-2">Создан</th>
              <th className="px-4 py-2">Использован</th>
              <th className="px-4 py-2">Почта</th>
              <th className="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td className="px-4 py-6 text-slate-500" colSpan={7}>
                  Загрузка…
                </td>
              </tr>
            ) : invites.length === 0 ? (
              <tr>
                <td className="px-4 py-6 text-slate-500" colSpan={7}>
                  Ключей нет
                </td>
              </tr>
            ) : (
              invites.map((i) => (
                <tr key={i.id} className="border-t border-slate-100">
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={selected.has(i.id)}
                      onChange={() => toggleOne(i.id)}
                      aria-label={`Выбрать ${i.code_prefix}`}
                    />
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span className="font-mono">
                        {i.code || `${i.code_prefix}••••`}
                      </span>
                      {i.code ? (
                        <CopyButton
                          copied={copiedId === i.id}
                          onCopy={() => void copyCode(i.id, i.code!)}
                        />
                      ) : (
                        <span className="text-xs text-slate-400" title="Ключ выпущен до сохранения кода">
                          нет кода
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">{statusLabel(i.status)}</td>
                  <td className="px-4 py-3">{formatDateTime(i.created_at)}</td>
                  <td className="px-4 py-3">{formatDateTime(i.used_at)}</td>
                  <td className="px-4 py-3">{i.used_by_email || "—"}</td>
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
              ))
            )}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-slate-500">Всего: {total}</p>
    </div>
  );
}

function CopyButton({ copied, onCopy }: { copied: boolean; onCopy: () => void }) {
  return (
    <button
      type="button"
      onClick={onCopy}
      className="rounded-md p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-800"
      aria-label="Скопировать ключ"
    >
      {copied ? <Check className="h-4 w-4 text-emerald-600" /> : <Copy className="h-4 w-4" />}
    </button>
  );
}
