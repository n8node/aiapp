"use client";

import { useEffect, useState } from "react";
import { ApiError, fetchAdminUILocale, saveAdminUILocale, type UILocaleOption } from "@/lib/api";

export function AdminLocalePage() {
  const [locale, setLocale] = useState("ru");
  const [locales, setLocales] = useState<UILocaleOption[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    fetchAdminUILocale()
      .then((res) => {
        setLocale(res.data.locale);
        setLocales(res.data.locales ?? []);
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : "Не удалось загрузить язык"));
  }, []);

  async function save() {
    setError(null);
    setSaved(false);
    try {
      const res = await saveAdminUILocale(locale);
      setLocale(res.data.locale);
      setLocales(res.data.locales ?? []);
      setSaved(true);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось сохранить");
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Язык интерфейса</h1>
        <p className="text-sm text-slate-500">
          Язык кабинета и Unsloth Studio задаётся здесь. Пользователи не могут сменить его в своих настройках.
          По умолчанию — русский.
        </p>
      </div>
      <select
        value={locale}
        onChange={(e) => setLocale(e.target.value)}
        className="w-full max-w-sm rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
      >
        {locales.map((item) => (
          <option key={item.code} value={item.code}>
            {item.label} ({item.code})
          </option>
        ))}
      </select>
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      {saved ? <p className="text-sm text-emerald-700">Сохранено</p> : null}
      <button type="button" onClick={() => void save()} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white">
        Сохранить
      </button>
    </div>
  );
}
