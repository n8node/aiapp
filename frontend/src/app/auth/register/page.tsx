import { RegisterForm } from "@/components/auth/RegisterForm";

export default function RegisterPage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-6 py-16">
      <div className="mb-8">
        <p className="text-sm font-medium text-muted">RigIntel</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Регистрация</h1>
        <p className="mt-2 text-sm text-muted">
          Только по инвайт-ключу администратора и корпоративной почте.
        </p>
      </div>
      <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
        <RegisterForm />
      </div>
    </main>
  );
}
