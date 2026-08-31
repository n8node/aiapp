import { Suspense } from "react";
import { LoginForm } from "@/components/auth/LoginForm";

export default function LoginPage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-6 py-16">
      <div className="mb-8">
        <p className="text-sm font-medium text-muted">RigIntel</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Вход</h1>
        <p className="mt-2 text-sm text-muted">Корпоративная AI-платформа</p>
      </div>
      <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
        <Suspense fallback={<p className="text-sm text-muted">Загрузка…</p>}>
          <LoginForm />
        </Suspense>
      </div>
    </main>
  );
}
