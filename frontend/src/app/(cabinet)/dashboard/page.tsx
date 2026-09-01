import { getMe } from "@/lib/auth-server";

function roleLabel(role?: string) {
  if (role === "department_manager") return "Руководитель отдела";
  if (role === "employee") return "Сотрудник";
  return role || "";
}

export default async function DashboardPage() {
  const me = await getMe();
  const ws = me?.workspace;

  return (
    <div>
      <h1 className="text-2xl font-semibold tracking-tight">Обзор</h1>
      {ws ? (
        <p className="mt-2 text-sm text-muted">
          Сейчас открыто пространство <span className="font-medium text-text">{ws.name}</span>
          {ws.role ? ` · ${roleLabel(ws.role)}` : ""}. Пишите моделям в «Чатах», подключайте RAG-коллекции как контекст.
        </p>
      ) : (
        <p className="mt-2 text-sm text-muted">
          Кабинет готов. Пространство ещё не назначено.
        </p>
      )}
    </div>
  );
}
