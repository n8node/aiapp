"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { OtpCells, type OtpStatus } from "@/components/auth/OtpCells";
import { ApiError, login } from "@/lib/api";

export function LoginForm() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [needTotp, setNeedTotp] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [otpStatus, setOtpStatus] = useState<OtpStatus>("idle");
  const tried = useRef("");

  async function authenticate(code: string) {
    return login(email, password, code);
  }

  function redirectAfterLogin(res: Awaited<ReturnType<typeof login>>) {
    if (!res.data.user.totp_enabled) {
      router.push("/auth/2fa");
    } else {
      router.push(res.data.user.is_platform_admin ? "/admin/users" : "/dashboard");
    }
    router.refresh();
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (needTotp) return;
    setError("");
    setLoading(true);
    try {
      redirectAfterLogin(await authenticate(""));
    } catch (err) {
      if (err instanceof ApiError && err.code === "totp_required") {
        setNeedTotp(true);
        setError("");
      } else {
        setError(err instanceof ApiError ? err.message : "Не удалось войти");
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!needTotp) return;
    if (totpCode.length < 6) {
      tried.current = "";
      if (otpStatus === "error") setOtpStatus("idle");
      return;
    }
    if (totpCode === tried.current || otpStatus === "verifying" || otpStatus === "success") {
      return;
    }
    tried.current = totpCode;
    setOtpStatus("verifying");
    setError("");
    void authenticate(totpCode)
      .then((res) => {
        setOtpStatus("success");
        window.setTimeout(() => redirectAfterLogin(res), 450);
      })
      .catch((err) => {
        setOtpStatus("error");
        setError(err instanceof ApiError ? err.message : "Неверный код");
      });
  }, [needTotp, totpCode, otpStatus, email, password, router]);

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
        <div className="space-y-2">
          <p className="text-center text-sm text-muted">Код из приложения</p>
          <OtpCells value={totpCode} onChange={setTotpCode} status={otpStatus} />
          {otpStatus === "verifying" ? (
            <p className="text-center text-xs text-muted">Проверяем код…</p>
          ) : null}
          {otpStatus === "success" ? (
            <p className="text-center text-xs font-medium text-emerald-700">Код подтверждён</p>
          ) : null}
        </div>
      )}
      {error ? <p className="text-center text-sm text-red-700">{error}</p> : null}
      {!needTotp && (
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60"
        >
          {loading ? "Вход…" : "Войти"}
        </button>
      )}
      <p className="text-sm text-muted">
        Нет аккаунта?{" "}
        <Link href="/auth/register" className="text-accent hover:underline">
          Регистрация по инвайту
        </Link>
      </p>
    </form>
  );
}
