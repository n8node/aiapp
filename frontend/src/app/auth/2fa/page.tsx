import { redirect } from "next/navigation";
import { TotpSetupForm } from "@/components/auth/TotpSetupForm";
import { getMe } from "@/lib/auth-server";

export default async function TotpPage() {
  const me = await getMe();
  if (!me) redirect("/auth/login");
	if (me.user.totp_enabled) {
    redirect(me.user.is_platform_admin ? "/admin/users" : "/dashboard");
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-6 py-16">
      <div className="mb-8">
        <p className="text-sm font-medium text-muted">RigIntel</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Двухфакторная аутентификация</h1>
        <p className="mt-2 text-sm text-muted">
          Обязательный шаг. Без подтверждения доступ к порталу закрыт.
        </p>
      </div>
      <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
        <TotpSetupForm />
      </div>
    </main>
  );
}
