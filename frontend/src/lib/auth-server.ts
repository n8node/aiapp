import { cookies } from "next/headers";

export type ServerMe = {
  user: {
    id: string;
    email: string;
    name: string;
    is_platform_admin: boolean;
    totp_enabled: boolean;
  };
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
  return body.data ?? null;
}
