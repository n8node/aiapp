import { redirect } from "next/navigation";
import { TrainingPage } from "@/components/training/TrainingPage";
import { getMe } from "@/lib/auth-server";

export default async function TrainingRoute() {
  const me = await getMe();
  if (!me) redirect("/auth/login");
  return (
    <TrainingPage
      user={{
        id: me.user.id,
        email: me.user.email,
        name: me.user.name,
        is_platform_admin: me.user.is_platform_admin,
        studio_access: me.user.studio_access,
        is_blocked: false,
        totp_enabled: me.user.totp_enabled,
        created_at: "",
      }}
      uiLocale={me.ui_locale || "ru"}
    />
  );
}
