"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, login } from "@/lib/api";

export function LoginForm() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [needTotp, setNeedTotp] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await login(email, password, totpCode);
      if (!res.data.user.totp_enabled) {
        router.push("/auth/2fa");
        router.refresh();
        return;
      }
      router.push(res.data.user.is_platform_admin ? "/admin/users" : "/dashboard");
      router.refresh();
    } catch (err) {
      if (err instanceof ApiError && err.code === "totp_required") {
        setNeedTotp(true);
        setError("Введите код из приложения-аутентификатора");
      } else {
        setError(err instanceof ApiError ? err.message : "Не удалось войти");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <label className="block text-sm">
        <span className="text-muted">Email</span>
        <input
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm"
        />
      </label>
      <label className="block text-sm">
        <span className="text-muted">Пароль</span>
        <input
          type="password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm"
        />
      </label>
      {needTotp && (
        <label className="block text-sm">
          <span className="text-muted">Код 2FA</span>
          <input
            inputMode="numeric"
            autoComplete="one-time-code"
            value={totpCode}
            onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
            className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm"
            placeholder="000000"
          />
        </label>
      )}
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      <button
        type="submit"
        disabled={loading}
        className="w-full rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60"
      >
        {loading ? "Вход…" : "Войти"}
      </button>
      <p className="text-sm text-muted">
        Нет аккаунта?{" "}
        <Link href="/auth/register" className="text-accent hover:underline">
          Регистрация по инвайту
        </Link>
      </p>
    </form>
  );
}
