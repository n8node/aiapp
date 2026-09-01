"use client";

import { useEffect, useState } from "react";
import {
  ApiError,
  fetchAdminOutboundProxy,
  saveAdminOutboundProxy,
  testAdminOutboundProxy,
  type OutboundProxySettings,
  type OutboundProxyTestResult,
} from "@/lib/api";

const empty: OutboundProxySettings = {
  proxy_enabled: false,
  proxy_active_url: "",
  proxy_urls: [],
  active_masked: "",
};

export function AdminOutboundProxyPage() {
  const [settings, setSettings] = useState<OutboundProxySettings>(empty);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [probe, setProbe] = useState<OutboundProxyTestResult | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    fetchAdminOutboundProxy()
      .then((res) => setSettings({ ...empty, ...res.data, proxy_urls: res.data.proxy_urls ?? [] }))
      .catch((e) => setError(e instanceof ApiError ? e.message : "Не удалось загрузить настройки"));
  }, []);

  function patch(next: Partial<OutboundProxySettings>) {
    setSettings((prev) => ({ ...prev, ...next }));
    setSuccess(null);
    setProbe(null);
  }

  const proxyOptions = settings.proxy_urls.filter(Boolean);
  const proxySelectValue =
    settings.proxy_active_url && proxyOptions.includes(settings.proxy_active_url)
      ? settings.proxy_active_url
      : "";

  async function save() {
    setError(null);
    setSuccess(null);
    setSaving(true);
    try {
      const res = await saveAdminOutboundProxy({
        proxy_enabled: settings.proxy_enabled,
        proxy_active_url: settings.proxy_active_url,
        proxy_urls: settings.proxy_urls,
      });
      setSettings({ ...empty, ...res.data, proxy_urls: res.data.proxy_urls ?? [] });
      setSuccess("Сохранено. Перезапустите unsloth-studio: список моделей останется прямым, файлы пойдут через прокси.");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось сохранить");
    } finally {
      setSaving(false);
    }
  }

  async function test() {
    setError(null);
    setTesting(true);
    try {
      const res = await testAdminOutboundProxy();
      setProbe(res.data);
      if (!res.data.ok) {
        setError(res.data.message);
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось проверить прокси");
    } finally {
      setTesting(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-slate-900">Исходящий прокси</h1>
        <p className="mt-1 text-sm text-slate-500">
          HTTP CONNECT только для файлов на CDN Hugging Face (GGUF в Unsloth Studio). Список моделей
          и API huggingface.co идут с сервера напрямую, как раньше. Токен Hub задаётся в Studio →
          Настройки, сюда его дублировать не нужно.
        </p>
      </div>

      {error ? (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      ) : null}
      {success ? (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">
          {success}
        </div>
      ) : null}
      {probe?.ok ? (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
          {probe.message}
          <p className="mt-1 font-mono text-xs text-emerald-700">Hub: {probe.hub}. Resolve: {probe.cdn}</p>
        </div>
      ) : null}

      <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 className="font-medium text-slate-900">Прокси для Hugging Face</h2>
        <p className="text-xs text-slate-500">
          huggingface.co с сервера открывается, раздача на us.aws.cdn.hf.co — нет. Прокси не должен
          перехватывать Hub (иначе список моделей зависает). Тот же HTTP-прокси, что для Telegram в
          POSTILKA: http://user:pass@host:port.
        </p>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={settings.proxy_enabled}
            onChange={(e) => patch({ proxy_enabled: e.target.checked })}
            className="rounded border-slate-300"
          />
          Включить прокси для загрузки файлов с CDN Hugging Face
        </label>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">
            Список прокси (по одному URL на строку)
          </label>
          <textarea
            value={settings.proxy_urls.join("\n")}
            onChange={(e) => {
              const proxy_urls = e.target.value
                .split("\n")
                .map((s) => s.trim())
                .filter(Boolean);
              const proxy_active_url =
                settings.proxy_active_url && proxy_urls.includes(settings.proxy_active_url)
                  ? settings.proxy_active_url
                  : "";
              patch({ proxy_urls, proxy_active_url });
            }}
            rows={3}
            placeholder="http://user:pass@host:3128"
            className="w-full rounded-lg border border-slate-200 px-3 py-2 font-mono text-sm"
          />
          <p className="mt-1 text-xs text-slate-400">Только http://. SOCKS5 не поддерживается.</p>
        </div>

        {proxyOptions.length > 0 ? (
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Активный прокси</label>
            <select
              value={proxySelectValue}
              onChange={(e) => patch({ proxy_active_url: e.target.value })}
              className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm"
            >
              <option value="">Авто: первый в списке</option>
              {proxyOptions.map((url) => (
                <option key={url} value={url}>
                  {url}
                </option>
              ))}
            </select>
            {settings.active_masked ? (
              <p className="mt-1 text-xs text-slate-400">Сейчас: {settings.active_masked}</p>
            ) : null}
          </div>
        ) : null}

        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => void save()}
            disabled={saving}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
          >
            {saving ? "Сохранение…" : "Сохранить"}
          </button>
          <button
            type="button"
            onClick={() => void test()}
            disabled={testing}
            className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 disabled:opacity-50"
          >
            {testing ? "Проверка…" : "Проверить"}
          </button>
        </div>
      </section>
    </div>
  );
}
