"use client";

import { useEffect, useState } from "react";
import { fetchModelRuntime } from "@/lib/models-api";

export function AdminUnslothPage() {
  const [reachable, setReachable] = useState<boolean | null>(null);

  useEffect(() => {
    void fetchModelRuntime()
      .then((r) => setReachable(r.data.studio.reachable))
      .catch(() => setReachable(false));
  }, []);

  return (
    <div className="flex h-[calc(100vh-8rem)] flex-col">
      <div className="mb-3">
        <h1 className="text-2xl font-semibold tracking-tight">Unsloth Studio</h1>
        <p className="mt-1 text-sm text-slate-500">
          Изолированный инженерный контур поверх RigIntel-сессии: загрузка моделей, обучение LoRA/QLoRA, экспорт.
          Не является шлюзом кабинета. Доступ: администратор платформы или пользователь с правом Studio.
        </p>
        {reachable === false ? (
          <p className="mt-2 text-sm text-amber-700">
            Studio сейчас не отвечает. После деплоя контейнера unsloth-studio интерфейс появится ниже.
          </p>
        ) : null}
      </div>
      <iframe
        title="Unsloth Studio"
        src="/app/studio/"
        className="min-h-0 flex-1 rounded-xl border border-slate-200 bg-white"
      />
    </div>
  );
}
