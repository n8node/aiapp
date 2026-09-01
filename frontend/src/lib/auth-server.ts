import { cookies } from "next/headers";

export type ServerWorkspace = {
  id: string;
  name: string;
  slug: string;
  role?: string;
};

export type ServerMe = {
  user: {
    id: string;
    email: string;
    name: string;
    is_platform_admin: boolean;
    studio_access?: boolean;
    totp_enabled: boolean;
  };
  workspace: ServerWorkspace | null;
  workspaces: ServerWorkspace[];
  ui_locale?: string;
};

export async function getMe(): Promise<ServerMe | null> {
  const jar = await cookies();
  const token = jar.get("access_token")?.value;
  if (!token) return null;
  const base =
    process.env.INTERNAL_API_URL?.replace(/\/$/, "") || "http://backend:8080";
  const res = await fetch(`${base}/api/v1/auth/me`, {
    headers: { Cookie: `access_token=${token}` },
    cache: "no-store",
  });
  if (!res.ok) return null;
  const body = (await res.json()) as { data?: ServerMe };
  const data = body.data;
  if (!data?.user) return null;
  return {
    user: data.user,
    workspace: data.workspace ?? null,
    workspaces: data.workspaces ?? [],
    ui_locale: data.ui_locale,
  };
}
