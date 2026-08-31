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
          Чат по базе знаний и сверка документов для отделов компании. Кабинет
          доступен после входа. Маркетинговый сайт — на главной странице.
        </p>
      </div>
      <HealthCard />
    </main>
  );
}
