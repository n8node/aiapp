import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/** Re-emit the session cookie on Path=/ so Unsloth /assets and /api get auth_request. */
export function middleware(request: NextRequest) {
  const token = request.cookies.get("access_token")?.value;
  const res = NextResponse.next();
  if (!token) {
    return res;
  }
  res.cookies.set({
    name: "access_token",
    value: token,
    path: "/",
    httpOnly: true,
    sameSite: "lax",
    secure: request.nextUrl.protocol === "https:",
    maxAge: 60 * 60 * 24 * 7,
  });
  return res;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image).*)"],
};
