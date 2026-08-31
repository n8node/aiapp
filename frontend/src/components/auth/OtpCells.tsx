"use client";

import { useEffect, useRef } from "react";
import { cn } from "@/lib/cn";

export type OtpStatus = "idle" | "verifying" | "success" | "error";

type OtpCellsProps = {
  value: string;
  onChange: (value: string) => void;
  status?: OtpStatus;
  disabled?: boolean;
  autoFocus?: boolean;
  length?: number;
};

export function OtpCells({
  value,
  onChange,
  status = "idle",
  disabled = false,
  autoFocus = true,
  length = 6,
}: OtpCellsProps) {
  const refs = useRef<Array<HTMLInputElement | null>>([]);

  useEffect(() => {
    if (autoFocus) refs.current[0]?.focus();
  }, [autoFocus]);

  function setCode(next: string) {
    onChange(next.replace(/\D/g, "").slice(0, length));
  }

  function focusAt(index: number) {
    const i = Math.max(0, Math.min(length - 1, index));
    refs.current[i]?.focus();
    refs.current[i]?.select();
  }

  function onPaste(e: React.ClipboardEvent<HTMLInputElement>) {
    const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, length);
    if (!pasted) return;
    e.preventDefault();
    setCode(pasted);
    focusAt(Math.min(pasted.length, length - 1));
  }

  function onKeyDown(index: number, e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Backspace") {
      e.preventDefault();
      if (value[index]) {
        setCode(value.slice(0, index) + value.slice(index + 1));
        focusAt(index);
      } else {
        setCode(value.slice(0, Math.max(0, index - 1)) + value.slice(index));
        focusAt(index - 1);
      }
      return;
    }
    if (e.key === "ArrowLeft") {
      e.preventDefault();
      focusAt(index - 1);
    }
    if (e.key === "ArrowRight") {
      e.preventDefault();
      focusAt(index + 1);
    }
  }

  function onInput(index: number, raw: string) {
    const digits = raw.replace(/\D/g, "");
    if (!digits) {
      const arr = value.split("");
      arr[index] = "";
      setCode(arr.join(""));
      return;
    }
    if (digits.length > 1) {
      setCode(digits);
      focusAt(Math.min(digits.length, length - 1));
      return;
    }
    const arr = Array.from({ length }, (_, i) => value[i] ?? "");
    arr[index] = digits;
    const next = arr.join("").replace(/\s/g, "");
    setCode(next);
    if (index < length - 1) focusAt(index + 1);
  }

  const digits = Array.from({ length }, (_, i) => value[i] ?? "");

  return (
    <div className="flex justify-center gap-2" role="group" aria-label="Код из шести цифр">
      {digits.map((digit, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el;
          }}
          type="text"
          inputMode="numeric"
          autoComplete={i === 0 ? "one-time-code" : "off"}
          maxLength={1}
          value={digit}
          disabled={disabled || status === "verifying" || status === "success"}
          onChange={(e) => onInput(i, e.target.value)}
          onKeyDown={(e) => onKeyDown(i, e)}
          onPaste={onPaste}
          onFocus={(e) => e.currentTarget.select()}
          aria-label={`Цифра ${i + 1}`}
          className={cn(
            "h-12 w-10 rounded-xl border bg-surface text-center text-lg font-semibold tabular-nums outline-none transition-colors sm:h-14 sm:w-11",
            status === "success" && "border-emerald-500 text-emerald-700 ring-2 ring-emerald-500/25",
            status === "error" && "border-red-500 text-red-700 ring-2 ring-red-500/25",
            status === "verifying" && "border-accent ring-2 ring-accent/20",
            status === "idle" && "border-border focus:border-accent focus:ring-2 focus:ring-accent/20",
          )}
        />
      ))}
    </div>
  );
}
