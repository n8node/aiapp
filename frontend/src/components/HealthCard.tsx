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
      <p className="rounded-lg border border-red-400/40 bg-red-950/40 px-4 py-3 text-sm text-red-200">
        {error}
      </p>
    );
  }

  if (!health) {
    return (
      <p className="text-sm text-muted">Проверяем состояние платформы…</p>
    );
  }

  const ok = health.status === "ok";
  return (
    <p
      className={`rounded-lg border px-4 py-3 text-sm ${
        ok
          ? "border-emerald-400/30 bg-emerald-950/30 text-emerald-200"
          : "border-amber-400/30 bg-amber-950/30 text-amber-100"
      }`}
    >
      API: {ok ? "доступен" : health.status}
      {health.postgres ? ` · база: ${health.postgres}` : ""}
      {health.version ? ` · версия ${health.version}` : ""}
    </p>
  );
}
