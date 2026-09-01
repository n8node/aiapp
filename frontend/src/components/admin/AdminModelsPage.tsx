"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
} from "@/lib/api";
import {
  PURPOSE_LABEL,
  STATUS_LABEL,
  actionAdminModel,
  createAdminModel,
  fetchModelRuntime,
  listAdminModels,
  type GatewayStatus,
  type MLModel,
  type StudioStatus,
} from "@/lib/models-api";

export function AdminModelsPage() {
  const [items, setItems] = useState<MLModel[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [studio, setStudio] = useState<StudioStatus | null>(null);
  const [gateway, setGateway] = useState<GatewayStatus | null>(null);
  const [slug, setSlug] = useState("");
  const [name, setName] = useState("");
  const [purpose, setPurpose] = useState("embeddings");
  const [sourceRef, setSourceRef] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [list, runtime] = await Promise.all([listAdminModels(), fetchModelRuntime()]);
      setItems(list.data.models ?? []);
      setStudio(runtime.data.studio);
      setGateway(runtime.data.gateway);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось загрузить модели");
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await createAdminModel({
        slug,
        display_name: name,
        purpose,
        source_type: "huggingface",
        source_ref: sourceRef,
      });
      setSlug("");
      setName("");
      setSourceRef("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось зарегистрировать модель");
    } finally {
      setBusy(false);
    }
  }

  async function onAction(id: string, action: string) {
    setBusy(true);
    setError(null);
    try {
      await actionAdminModel(id, action);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось изменить модель");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Модели</h1>
        <p className="mt-1 text-sm text-slate-500">
          Реестр: карантин → одобрение → шлюз. Веса чата и обучение загружаются в Unsloth Studio. Эмбеддинги по
          умолчанию отдаёт внутренний шлюз (E5-small, CPU).
        </p>
      </div>
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Шлюз эмбеддингов</p>
          <p className="mt-1 text-sm font-medium">{gateway?.reachable ? "Доступен" : "Недоступен"}</p>
          <p className="text-xs text-slate-500">{gateway?.embedding_model}</p>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Unsloth Studio</p>
          <p className="mt-1 text-sm font-medium">{studio?.reachable ? "Доступен" : "Недоступен"}</p>
          <p className="text-xs text-slate-500">Открывается в разделе Unsloth. Право выдаётся в карточке пользователя.</p>
        </div>
      </div>

      <form onSubmit={(e) => void onCreate(e)} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
        <p className="text-sm font-semibold">Зарегистрировать модель (карантин)</p>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          <input
            required
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            placeholder="slug (e5-large)"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
          />
          <input
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Отображаемое имя"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
          />
          <select
            value={purpose}
            onChange={(e) => setPurpose(e.target.value)}
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
          >
            <option value="embeddings">Эмбеддинги</option>
            <option value="chat">Чат</option>
            <option value="ocr">OCR</option>
            <option value="rerank">Реранк</option>
          </select>
          <input
            required
            value={sourceRef}
            onChange={(e) => setSourceRef(e.target.value)}
            placeholder="Hugging Face id (org/model)"
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
          />
        </div>
        <button
          type="submit"
          disabled={busy}
          className="mt-3 rounded-lg bg-blue-600 px-3 py-2 text-sm text-white disabled:opacity-50"
        >
          В карантин
        </button>
      </form>

      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase text-slate-500">
            <tr>
              <th className="px-4 py-2">Модель</th>
              <th className="px-4 py-2">Назначение</th>
              <th className="px-4 py-2">Статус</th>
              <th className="px-4 py-2">Источник</th>
              <th className="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            {items.map((m) => (
              <tr key={m.id} className="border-b border-slate-100">
                <td className="px-4 py-3">
                  <p className="font-medium">{m.display_name}</p>
                  <p className="text-xs text-slate-500">{m.slug}</p>
                </td>
                <td className="px-4 py-3">{PURPOSE_LABEL[m.purpose] ?? m.purpose}</td>
                <td className="px-4 py-3">{STATUS_LABEL[m.status] ?? m.status}</td>
                <td className="max-w-[14rem] truncate px-4 py-3 text-xs text-slate-500">{m.source_ref}</td>
                <td className="px-4 py-3 text-right">
                  {m.status === "quarantine" ? (
                    <>
                      <button type="button" disabled={busy} className="mr-2 text-blue-600 hover:underline" onClick={() => void onAction(m.id, "approve")}>
                        Одобрить
                      </button>
                      <button type="button" disabled={busy} className="text-red-600 hover:underline" onClick={() => void onAction(m.id, "reject")}>
                        Отклонить
                      </button>
                    </>
                  ) : null}
                  {m.status === "approved" ? (
                    <button type="button" disabled={busy} className="text-blue-600 hover:underline" onClick={() => void onAction(m.id, "deploy")}>
                      В шлюз
                    </button>
                  ) : null}
                  {m.status === "deployed" ? (
                    <button type="button" disabled={busy} className="text-slate-600 hover:underline" onClick={() => void onAction(m.id, "disable")}>
                      Отключить
                    </button>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
