"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/layout/EmptyState";
import {
  ApiError,
  cancelTrainingRequest,
  createTrainingRequest,
  fetchTrainingRequests,
  submitTrainingRequest,
  type TrainingRequest,
  type User,
} from "@/lib/api";

const purposes = [
  { value: "behavior", label: "Поведение" },
  { value: "format", label: "Формат ответа" },
  { value: "extraction", label: "Извлечение" },
  { value: "classification", label: "Классификация" },
];

const statusLabel: Record<string, string> = {
  draft: "Черновик",
  submitted: "На проверке",
  approved: "Одобрена",
  rejected: "Отклонена",
  cancelled: "Отменена",
};

export function TrainingPage({ user, uiLocale }: { user: User; uiLocale: string }) {
  const [items, setItems] = useState<TrainingRequest[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [title, setTitle] = useState("");
  const [purpose, setPurpose] = useState("behavior");
  const [datasetNote, setDatasetNote] = useState("");
  const [baseModel, setBaseModel] = useState("");

  useEffect(() => {
    if (!user.is_platform_admin) return;
    try {
      window.localStorage.setItem("unsloth_locale", uiLocale || "ru");
    } catch {
      /* ignore */
    }
  }, [uiLocale, user.is_platform_admin]);

  async function refresh() {
    const res = await fetchTrainingRequests();
    setItems(res.data.requests ?? []);
  }

  useEffect(() => {
    void refresh().catch((e) => setError(e instanceof Error ? e.message : "Ошибка загрузки"));
  }, []);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await createTrainingRequest({
        title,
        purpose,
        dataset_note: datasetNote,
        base_model: baseModel,
      });
      setTitle("");
      setDatasetNote("");
      setBaseModel("");
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось создать заявку");
    } finally {
      setBusy(false);
    }
  }

  async function act(id: string, kind: "submit" | "cancel") {
    setError(null);
    try {
      if (kind === "submit") await submitTrainingRequest(id);
      else await cancelTrainingRequest(id);
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось изменить заявку");
    }
  }

  return (
    <div>
      <PageHeader
        title="Обучение"
        description="Заявки на fine-tune идут через RigIntel: снимок датасета, проверка, одобрение. Обучение исполняется в инженерном контуре, модели в чат попадают только после выкладки администратором."
      />
      {error ? <p className="mb-4 text-sm text-red-600">{error}</p> : null}

      <form onSubmit={(e) => void onCreate(e)} className="mb-6 rounded-xl border border-border bg-surface p-4 shadow-sm">
        <p className="text-sm font-semibold">Новая заявка</p>
        <div className="mt-3 grid gap-2 sm:grid-cols-2">
          <input
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Название"
            className="rounded-lg border border-border px-3 py-2 text-sm"
          />
          <select
            value={purpose}
            onChange={(e) => setPurpose(e.target.value)}
            className="rounded-lg border border-border bg-surface px-3 py-2 text-sm"
          >
            {purposes.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
          <input
            value={baseModel}
            onChange={(e) => setBaseModel(e.target.value)}
            placeholder="Базовая модель (необязательно)"
            className="rounded-lg border border-border px-3 py-2 text-sm"
          />
          <textarea
            value={datasetNote}
            onChange={(e) => setDatasetNote(e.target.value)}
            placeholder="Какие данные пространства использовать"
            className="rounded-lg border border-border px-3 py-2 text-sm sm:col-span-2"
            rows={3}
          />
        </div>
        <button type="submit" disabled={busy} className="mt-3 rounded-lg bg-accent px-3 py-2 text-sm text-white disabled:opacity-50">
          Сохранить черновик
        </button>
      </form>

      {items.length === 0 ? (
        <EmptyState title="Заявок пока нет" description="Создайте черновик, затем отправьте его на проверку платформенному администратору." />
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
          <table className="w-full text-left text-sm">
            <thead className="bg-zinc-50 text-xs uppercase text-muted">
              <tr>
                <th className="px-4 py-2">Заявка</th>
                <th className="px-4 py-2">Статус</th>
                <th className="px-4 py-2" />
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id} className="border-t border-border">
                  <td className="px-4 py-3">
                    <p className="font-medium">{item.title}</p>
                    <p className="text-xs text-muted">{item.purpose}</p>
                  </td>
                  <td className="px-4 py-3">{statusLabel[item.status] ?? item.status}</td>
                  <td className="px-4 py-3 text-right">
                    {item.status === "draft" ? (
                      <button type="button" className="mr-2 rounded-lg border border-border px-3 py-1 text-xs" onClick={() => void act(item.id, "submit")}>
                        Отправить
                      </button>
                    ) : null}
                    {item.status === "draft" || item.status === "submitted" ? (
                      <button type="button" className="rounded-lg border border-border px-3 py-1 text-xs" onClick={() => void act(item.id, "cancel")}>
                        Отменить
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {user.is_platform_admin ? (
        <p className="mt-6 text-sm text-muted">
          Полный контур обучения для инженеров открывается отдельно, без оболочки кабинета.{" "}
          <Link href="/admin/unsloth" className="text-accent hover:underline">
            Открыть контур обучения
          </Link>
        </p>
      ) : (
        <p className="mt-6 text-sm text-muted">
          Заявки проверяет администратор. Одобренные модели появляются в разделе «Чаты».
        </p>
      )}
    </div>
  );
}
