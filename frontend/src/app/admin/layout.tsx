import { redirect } from "next/navigation";
import { AdminShell } from "@/components/admin/AdminShell";
import { getMe } from "@/lib/auth-server";

export default async function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const me = await getMe();
  if (!me) redirect("/auth/login?next=/admin/users");
  if (!me.user.totp_enabled) redirect("/auth/2fa");
  if (!me.user.is_platform_admin) redirect("/dashboard");

  return (
    <AdminShell adminEmail={me.user.email} adminName={me.user.name}>
      {children}
    </AdminShell>
  );
}
