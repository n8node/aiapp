import { redirect } from "next/navigation";
import { AppShell } from "@/components/layout/AppShell";
import { getMe } from "@/lib/auth-server";

export default async function CabinetLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const me = await getMe();
  if (!me) redirect("/auth/login");
  if (!me.user.totp_enabled) redirect("/auth/2fa");

  return (
    <AppShell
      user={{
        id: me.user.id,
        email: me.user.email,
        name: me.user.name,
        is_platform_admin: me.user.is_platform_admin,
        is_blocked: false,
        totp_enabled: me.user.totp_enabled,
        created_at: "",
      }}
      workspace={me.workspace}
      workspaces={me.workspaces}
    >
      {children}
    </AppShell>
  );
}
