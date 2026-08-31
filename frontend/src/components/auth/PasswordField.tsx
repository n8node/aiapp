"use client";

import { Eye, EyeOff, Sparkles } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {
  checkPasswordRules,
  isPasswordValid,
  PASSWORD_RULE_KEYS,
  PASSWORD_RULE_LABELS,
  PASSWORD_STRENGTH_LABELS,
  passwordStrength,
  type PasswordStrength,
} from "@/lib/password-policy";
import { copyToClipboard, generateSecurePassword } from "@/lib/generate-password";
import { cn } from "@/lib/cn";

const inputClass =
  "w-full rounded-lg border border-slate-200 bg-surface py-2 pl-3 text-sm outline-none focus:border-accent focus:ring-2 focus:ring-accent/20";

const STRENGTH_BAR: Record<PasswordStrength, string> = {
  0: "bg-slate-200",
  1: "bg-red-400",
  2: "bg-amber-400",
  3: "bg-accent",
  4: "bg-emerald-500",
};

const STRENGTH_TEXT: Record<PasswordStrength, string> = {
  0: "text-muted",
  1: "text-red-600",
  2: "text-amber-600",
  3: "text-accent",
  4: "text-emerald-600",
};

type TooltipPosition = {
  left: number;
  top: number;
};

type CursorTip = {
  text: string;
  key: number;
  position: TooltipPosition;
};

type PasswordFieldProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  autoComplete?: string;
  showStrength?: boolean;
  showRequirements?: boolean;
  required?: boolean;
  allowGenerate?: boolean;
  onGenerated?: (password: string) => void;
};

export function PasswordField({
  id,
  label,
  value,
  onChange,
  autoComplete,
  showStrength = false,
  showRequirements = false,
  required = true,
  allowGenerate = false,
  onGenerated,
}: PasswordFieldProps) {
  const generateBtnRef = useRef<HTMLButtonElement>(null);
  const [visible, setVisible] = useState(false);
  const [hoverTip, setHoverTip] = useState<CursorTip | null>(null);
  const [copyTip, setCopyTip] = useState<CursorTip | null>(null);
  const [mounted, setMounted] = useState(false);

  const rules = useMemo(() => checkPasswordRules(value), [value]);
  const strength = useMemo(() => passwordStrength(value, rules), [value, rules]);
  const valid = isPasswordValid(rules);

  const strengthLabel =
    strength === 0 ? "" : PASSWORD_STRENGTH_LABELS[strength];

  useEffect(() => {
    setMounted(true);
  }, []);

  const getGenerateTipPosition = useCallback((): TooltipPosition | null => {
    const btn = generateBtnRef.current;
    if (!btn) return null;
    const rect = btn.getBoundingClientRect();
    return {
      left: rect.left + rect.width / 2,
      top: rect.top - 8,
    };
  }, []);

  const showTip = useCallback(
    (text: string) => {
      const position = getGenerateTipPosition();
      if (!position) return;
      const tip = { text, key: Date.now(), position };
      setCopyTip(tip);
      window.setTimeout(() => {
        setCopyTip((current) => (current?.key === tip.key ? null : current));
      }, 2200);
    },
    [getGenerateTipPosition],
  );

  useEffect(() => {
    if (!copyTip) return;
    const onScroll = () => setCopyTip(null);
    window.addEventListener("scroll", onScroll, true);
    return () => window.removeEventListener("scroll", onScroll, true);
  }, [copyTip]);

  function showHoverTip() {
    const position = getGenerateTipPosition();
    if (!position) return;
    setHoverTip({
      text: "Сгенерировать надёжный пароль и скопировать",
      key: 0,
      position,
    });
  }

  async function handleGenerate() {
    const pwd = generateSecurePassword();
    onChange(pwd);
    onGenerated?.(pwd);
    setVisible(true);
    await copyToClipboard(pwd);
    showTip("Пароль скопирован в буфер обмена");
    setHoverTip(null);
  }

  const inputPad = allowGenerate ? "pr-[4.5rem]" : "pr-10";

  return (
    <div>
      <label htmlFor={id} className="mb-1.5 block text-xs font-medium">
        {label}
      </label>
      <div className="relative">
        <input
          id={id}
          type={visible ? "text" : "password"}
          autoComplete={autoComplete}
          required={required}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(inputClass, inputPad)}
          aria-describedby={
            showStrength || showRequirements ? `${id}-hints` : undefined
          }
        />
        <div className="absolute right-2 top-1/2 flex -translate-y-1/2 items-center gap-0.5">
          {allowGenerate && (
            <button
              ref={generateBtnRef}
              type="button"
              tabIndex={-1}
              onClick={handleGenerate}
              onMouseEnter={showHoverTip}
              onFocus={showHoverTip}
              onMouseLeave={() => setHoverTip(null)}
              onBlur={() => setHoverTip(null)}
              className="rounded p-1 text-muted transition-colors hover:bg-slate-100 hover:text-accent"
              aria-label="Сгенерировать пароль"
            >
              <Sparkles className="h-4 w-4" />
            </button>
          )}
          <button
            type="button"
            tabIndex={-1}
            onClick={() => setVisible((v) => !v)}
            className="rounded p-1 text-muted hover:text-text"
            aria-label={visible ? "Скрыть пароль" : "Показать пароль"}
          >
            {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
      </div>

      {mounted && hoverTip && createPortal(
        <CursorTooltip tip={hoverTip} variant="hint" />,
        document.body,
      )}
      {mounted && copyTip && createPortal(
        <CursorTooltip tip={copyTip} variant="success" />,
        document.body,
      )}

      {(showStrength || showRequirements) && value.length > 0 && (
        <div id={`${id}-hints`} className="mt-2 space-y-2">
          {showStrength && (
            <div className="space-y-1">
              <div
                className="flex gap-1"
                role="meter"
                aria-valuenow={strength}
                aria-valuemin={0}
                aria-valuemax={4}
              >
                {[1, 2, 3, 4].map((level) => (
                  <div
                    key={level}
                    className={cn(
                      "h-1 flex-1 rounded-full transition-colors duration-200",
                      strength >= level
                        ? STRENGTH_BAR[strength]
                        : "bg-slate-200",
                    )}
                  />
                ))}
              </div>
              {strengthLabel && (
                <p className={cn("text-xs font-medium", STRENGTH_TEXT[strength])}>
                  {strengthLabel}
                </p>
              )}
            </div>
          )}

          {showRequirements && (
            <ul className="space-y-1 text-xs">
              {PASSWORD_RULE_KEYS.map((key) => (
                <RuleItem key={key} met={rules[key]} label={PASSWORD_RULE_LABELS[key]} />
              ))}
            </ul>
          )}
        </div>
      )}

      {showRequirements && value.length > 0 && !valid && (
        <span className="sr-only">Пароль не соответствует требованиям</span>
      )}
    </div>
  );
}

function CursorTooltip({
  tip,
  variant,
}: {
  tip: CursorTip;
  variant: "hint" | "success";
}) {
  return (
    <div
      className={cn(
        "pointer-events-none fixed z-[9999] w-max max-w-[220px] -translate-x-1/2 -translate-y-full rounded-lg px-3 py-2 text-xs font-medium shadow-lg",
        variant === "success"
          ? "border border-emerald-200 bg-emerald-50 text-emerald-800"
          : "border border-slate-200 bg-white text-slate-700",
      )}
      style={{ left: tip.position.left, top: tip.position.top }}
      role="status"
    >
      {variant === "success" && (
        <span className="mr-1.5 inline-block text-emerald-600" aria-hidden>
          ✓
        </span>
      )}
      {tip.text}
    </div>
  );
}

function RuleItem({ met, label }: { met: boolean; label: string }) {
  return (
    <li
      className={cn(
        "flex items-center gap-2 transition-colors",
        met ? "text-emerald-700" : "text-muted",
      )}
    >
      <span
        className={cn(
          "inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full border text-[9px] leading-none",
          met
            ? "border-emerald-600 bg-emerald-50 text-emerald-700"
            : "border-slate-300 bg-white",
        )}
        aria-hidden
      >
        {met ? "✓" : ""}
      </span>
      <span>{label}</span>
    </li>
  );
}

export { checkPasswordRules, isPasswordValid };
