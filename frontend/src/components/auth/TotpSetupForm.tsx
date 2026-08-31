"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import QRCode from "react-qr-code";
import { ApiError, confirmTotp, fetchMe, setupTotp } from "@/lib/api";

export function TotpSetupForm() {
  const router = useRouter();
  const [secret, setSecret] = useState("");
  const [otpauth, setOtpauth] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setupTotp()
      .then((res) => {
        setSecret(res.data.secret);
        setOtpauth(res.data.otpauth_url);
      })
      .catch((err) => {
        setError(err instanceof ApiError ? err.message : "Не удалось начать настройку 2FA");
      })
      .finally(() => setLoading(false));
  }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await confirmTotp(code);
      const me = await fetchMe();
      router.push(me.data.user.is_platform_admin ? "/admin/users" : "/dashboard");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Неверный код");
    } finally {
      setLoading(false);
    }
  }

  if (!secret && loading) {
    return <p className="text-sm text-muted">Готовим ключ двухфакторной аутентификации…</p>;
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div className="rounded-xl border border-border bg-bg px-4 py-3 text-sm">
        <p className="font-medium text-text">Как подключить</p>
        <ol className="mt-2 list-decimal space-y-2 pl-4 text-muted">
          <li>
            На телефоне откройте{" "}
            <span className="font-medium text-text">Яндекс ID</span>
            — это приложение для кодов 2FA. Раньше оно называлось Яндекс.Ключ:
            если Ключ уже стоит, просто обновите его. Нет приложения?{" "}
            <a
              href="https://ya.ru/all?mode=apps&service=key"
              target="_blank"
              rel="noopener noreferrer"
              className="text-accent hover:underline"
            >
              Скачать
            </a>
            . Ссылка сама откроет магазин вашего устройства.
          </li>
          <li>
            В приложении нажмите «Сканировать QR» и наведите камеру на код
            ниже. Так делают, когда эта страница открыта на компьютере, а
            приложение — на телефоне.
          </li>
          <li>
            Если настраиваете с телефона и камера не видит этот экран — в
            Яндекс ID выберите «Настроить 2FA TOTP» → «Добавить ключ вручную»
            и введите секрет под QR-кодом.
          </li>
          <li>Введите сюда 6 цифр из приложения и подтвердите.</li>
        </ol>
        <p className="mt-3 text-xs text-muted">
          Подойдёт и другое приложение с TOTP. Подробнее:{" "}
          <a
            href="https://yandex.ru/support/id/ru/authorization/twofa"
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent hover:underline"
          >
            справка Яндекс ID
          </a>
          .
        </p>
      </div>
      {otpauth ? (
        <div className="flex justify-center rounded-xl border border-border bg-white p-4">
          <QRCode value={otpauth} size={192} bgColor="#ffffff" fgColor="#111111" />
        </div>
      ) : null}
      {secret ? (
        <div className="rounded-xl border border-border bg-bg px-3 py-2">
          <p className="text-xs text-muted">Ключ вручную</p>
          <code className="break-all font-mono text-xs">{secret}</code>
        </div>
      ) : null}
      <label className="block text-sm">
        <span className="text-muted">Код из приложения</span>
        <input
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
          className="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm"
          placeholder="000000"
        />
      </label>
      {error ? <p className="text-sm text-red-700">{error}</p> : null}
      <button
        type="submit"
        disabled={loading || code.length !== 6}
        className="w-full rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60"
      >
        {loading ? "Проверка…" : "Подтвердить и продолжить"}
      </button>
    </form>
  );
}
