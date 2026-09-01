"use client";

import { useEffect, useState } from "react";
import {
  ApiError,
  approveAdminTrainingRequest,
  fetchAdminTrainingRequests,
  rejectAdminTrainingRequest,
  type TrainingRequest,
} from "@/lib/api";

const statusLabel: Record<string, string> = {
  draft: "Черновик",
  submitted: "На проверке",
  approved: "Одобрена",
  rejected: "Отклонена",
  cancelled: "Отменена",
};

export function AdminTrainingPage() {
  const [items, setItems] = useState<TrainingRequest[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [note, setNote] = useState("");

  async function load() {
    const res = await fetchAdminTrainingRequests();
    setItems(res.data.requests ?? []);
  }

  useEffect(() => {
    void load().catch((e) => setError(e instanceof ApiError ? e.message : "Не удалось загрузить заявки"));
  }, []);

  async function act(id: string, kind: "approve" | "reject") {
    setError(null);
    try {
      if (kind === "approve") await approveAdminTrainingRequest(id, note);
      else await rejectAdminTrainingRequest(id, note);
      setNote("");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Не удалось изменить заявку");
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Заявки на обучение</h1>
        <p className="text-sm text-slate-500">
          Одобрение открывает право запустить обучение в Studio. Выкат в шлюз остаётся в реестре моделей.
        </p>
      </div>
      <textarea
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder="Комментарий к решению"
        className="w-full max-w-xl rounded-lg border border-slate-200 px-3 py-2 text-sm"
        rows={2}
      />
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase text-slate-500">
            <tr>
              <th className="px-4 py-2">Заявка</th>
              <th className="px-4 py-2">Автор</th>
              <th className="px-4 py-2">Статус</th>
              <th className="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            {items.length === 0 ? (
              <tr>
                <td className="px-4 py-6 text-slate-500" colSpan={4}>
                  Заявок нет
                </td>
              </tr>
            ) : (
              items.map((item) => (
                <tr key={item.id} className="border-t border-slate-100">
                  <td className="px-4 py-3">
                    <p className="font-medium">{item.title}</p>
                    <p className="text-xs text-slate-500">{item.purpose}</p>
                  </td>
                  <td className="px-4 py-3 text-xs">{item.requester_email || item.requested_by_user_id}</td>
                  <td className="px-4 py-3">{statusLabel[item.status] ?? item.status}</td>
                  <td className="px-4 py-3 text-right">
                    {item.status === "submitted" ? (
                      <>
                        <button type="button" className="mr-2 rounded-lg bg-blue-600 px-3 py-1 text-xs text-white" onClick={() => void act(item.id, "approve")}>
                          Одобрить
                        </button>
                        <button type="button" className="rounded-lg border border-slate-200 px-3 py-1 text-xs" onClick={() => void act(item.id, "reject")}>
                          Отклонить
                        </button>
                      </>
                    ) : null}
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
