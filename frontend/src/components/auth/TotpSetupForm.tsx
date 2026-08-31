"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import QRCode from "react-qr-code";
import { OtpCells, type OtpStatus } from "@/components/auth/OtpCells";
import { ApiError, confirmTotp, fetchMe, setupTotp } from "@/lib/api";

export function TotpSetupForm() {
  const router = useRouter();
  const [secret, setSecret] = useState("");
  const [otpauth, setOtpauth] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [otpStatus, setOtpStatus] = useState<OtpStatus>("idle");
  const tried = useRef("");

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

  useEffect(() => {
    if (code.length < 6) {
      tried.current = "";
      if (otpStatus === "error") setOtpStatus("idle");
      return;
    }
    if (code === tried.current || otpStatus === "verifying" || otpStatus === "success") {
      return;
    }
    tried.current = code;
    setOtpStatus("verifying");
    setError("");
    void confirmTotp(code)
      .then(async () => {
        setOtpStatus("success");
        const me = await fetchMe();
        window.setTimeout(() => {
          router.push(me.data.user.is_platform_admin ? "/admin/users" : "/dashboard");
          router.refresh();
        }, 450);
      })
      .catch((err) => {
        setOtpStatus("error");
        setError(err instanceof ApiError ? err.message : "Неверный код");
      });
  }, [code, otpStatus, router]);

  if (!secret && loading) {
    return <p className="text-sm text-muted">Готовим ключ двухфакторной аутентификации…</p>;
  }

  return (
    <div className="space-y-4">
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
          <li>Введите 6 цифр в ячейки — проверка начнётся сама.</li>
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
      <div className="space-y-2">
        <p className="text-center text-sm text-muted">Код из приложения</p>
        <OtpCells
          value={code}
          onChange={setCode}
          status={otpStatus}
          disabled={!secret}
        />
        {otpStatus === "verifying" ? (
          <p className="text-center text-xs text-muted">Проверяем код…</p>
        ) : null}
        {otpStatus === "success" ? (
          <p className="text-center text-xs font-medium text-emerald-700">Код подтверждён</p>
        ) : null}
        {otpStatus === "error" && error ? (
          <p className="text-center text-xs text-red-700">{error}</p>
        ) : null}
        {error && otpStatus !== "error" ? (
          <p className="text-center text-sm text-red-700">{error}</p>
        ) : null}
      </div>
    </div>
  );
}
