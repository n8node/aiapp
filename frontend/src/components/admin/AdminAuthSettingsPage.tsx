"use client";

import { useEffect, useState } from "react";
import { ApiError, fetchAuthDomains, saveAuthDomains } from "@/lib/api";

export function AdminAuthSettingsPage() {
  const [domains, setDomains] = useState<string[]>([]);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    fetchAuthDomains()
      .then((res) => setDomains(res.data.domains ?? []))
      .catch((e) => setError(e instanceof ApiError ? e.message : "Не удалось загрузить домены"));
  }, []);

  function addDomain() {
    const d = draft.trim().toLowerCase().replace(/^@/, "");
    if (!d) return;
    if (!domains.includes(d)) setDomains([...domains, d]);
    setDraft("");
  }

  async function save() {
    setError(null);
    setSaved(false);
    try {
      const res = await saveAuthDomains(domains);
      setDomains(res.data.domains);
      setSaved(true);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось сохранить");
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Домены почты</h1>
        <p className="text-sm text-slate-500">
          Регистрация и вход разрешены только с этих доменов. Адрес
          erman.ai@yandex.ru зарезервирован за суперадмином и не добавляется в
          список.
        </p>
      </div>
      <div className="flex gap-2">
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="rigintel.ai"
          className="w-full max-w-sm rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
        />
        <button type="button" onClick={addDomain} className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm">
          Добавить
        </button>
      </div>
      <ul className="space-y-2">
        {domains.map((d) => (
          <li key={d} className="flex max-w-sm items-center justify-between rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm">
            <span>{d}</span>
            <button type="button" className="text-xs text-slate-500" onClick={() => setDomains(domains.filter((x) => x !== d))}>
              Убрать
            </button>
          </li>
        ))}
      </ul>
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      {saved ? <p className="text-sm text-emerald-700">Сохранено</p> : null}
      <button type="button" onClick={() => void save()} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white">
        Сохранить
      </button>
    </div>
  );
}
