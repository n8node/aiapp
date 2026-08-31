"use client";

import { useEffect, useState } from "react";
import { HEALTH_PATH } from "@/lib/urls";

type Health = {
  status: string;
  postgres?: string;
  version?: string;
};

export function HealthCard() {
  const [health, setHealth] = useState<Health | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch(HEALTH_PATH, { cache: "no-store" })
      .then(async (res) => {
        const body = (await res.json()) as Health;
        if (!cancelled) {
          setHealth(body);
          setError(res.ok ? null : "Сервис отвечает с ошибкой");
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError("Не удалось проверить состояние платформы");
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (error) {
    return (
      <p className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
        {error}
      </p>
    );
  }

  if (!health) {
    return <p className="text-sm text-muted">Проверяем состояние платформы…</p>;
  }

  const ok = health.status === "ok";
  return (
    <p
      className={`rounded-xl border px-4 py-3 text-sm ${
        ok
          ? "border-border bg-surface text-muted"
          : "border-amber-200 bg-amber-50 text-amber-900"
      }`}
    >
      API: {ok ? "доступен" : health.status}
      {health.postgres ? ` · база: ${health.postgres}` : ""}
      {health.version ? ` · версия ${health.version}` : ""}
    </p>
  );
}
