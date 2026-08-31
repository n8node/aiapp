import Link from "next/link";
import { HealthCard } from "@/components/HealthCard";

export default function HomePage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col justify-center gap-8 px-6 py-16">
      <div>
        <p className="text-sm font-medium text-muted">RigIntel</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight text-text">
          Корпоративная AI-платформа
        </h1>
        <p className="mt-3 max-w-xl text-sm text-muted">
          Чат по базе знаний и сверка документов для отделов компании.
        </p>
      </div>
      <div className="flex flex-wrap gap-3">
        <Link
          href="/auth/login"
          className="rounded-lg bg-accent px-5 py-2.5 text-sm font-medium text-white"
        >
          Войти
        </Link>
        <Link
          href="/auth/register"
          className="rounded-lg border border-border bg-surface px-5 py-2.5 text-sm font-medium"
        >
          Регистрация
        </Link>
      </div>
      <HealthCard />
    </main>
  );
}
