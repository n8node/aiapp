"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, register, verifyInvite } from "@/lib/api";

export function RegisterForm() {
  const router = useRouter();
  const [inviteCode, setInviteCode] = useState("");
  const [inviteOk, setInviteOk] = useState(false);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function checkInvite(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await verifyInvite(inviteCode);
      setInviteOk(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Инвайт недействителен");
    } finally {
      setLoading(false);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (password !== confirm) {
      setError("Пароли не совпадают");
      return;
    }
    setLoading(true);
    try {
      await register(email, password, name, inviteCode);
      router.push("/auth/2fa");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Не удалось зарегистрироваться");
    } finally {
      setLoading(false);
    }
  }

  if (!inviteOk) {
    return (
      <form onSubmit={checkInvite} className="space-y-4">
        <p className="text-sm text-muted">
          Регистрация доступна только по инвайт-ключу, который выдаёт
          администратор.
        </p>
        <label className="block text-sm">
          <span className="text-muted">Инвайт-ключ</span>
          <input
            required
            value={inviteCode}
            onChange={(e) => setInviteCode(e.target.value.trim())}
            className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 font-mono text-sm"
            autoComplete="off"
          />
        </label>
        {error ? <p className="text-sm text-red-700">{error}</p> : null}
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60"
        >
          {loading ? "Проверка…" : "Продолжить"}
        </button>
        <p className="text-sm text-muted">
          Уже есть аккаунт?{" "}
          <Link href="/auth/login" className="text-accent hover:underline">
            Войти
          </Link>
        </p>
      </form>
    );
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <label className="block text-sm">
        <span className="text-muted">Имя</span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm"
        />
      </label>
      <label className="block text-sm">
        <span className="text-muted">Корпоративный email</span>
        <input
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm"
          placeholder="имя@rigintel.ai"
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
        <span className="mt-1 block text-xs text-muted">
          Минимум 10 символов, заглавная, строчная и цифра
        </span>
      </label>
      <label className="block text-sm">
        <span className="text-muted">Повтор пароля</span>
        <input
          type="password"
          required
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm"
        />
      </label>
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      <button
        type="submit"
        disabled={loading}
        className="w-full rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60"
      >
        {loading ? "Создание…" : "Создать аккаунт"}
      </button>
    </form>
  );
}
