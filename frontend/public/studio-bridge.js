(() => {
  const STORAGE_KEY = "unsloth_locale";
  const AUTH_TOKEN_KEY = "unsloth_auth_token";
  const AUTH_REFRESH_TOKEN_KEY = "unsloth_auth_refresh_token";
  const ONBOARDING_DONE_KEY = "unsloth_onboarding_done";
  const AUTH_MUST_CHANGE_PASSWORD_KEY = "unsloth_auth_must_change_password";
  const DEFAULT_LOCALE = "ru";

  function applyLocale(locale) {
    const value = locale || DEFAULT_LOCALE;
    try {
      localStorage.setItem(STORAGE_KEY, value);
    } catch {
      /* ignore quota / private mode */
    }
    document.documentElement.lang = value === "ru" ? "ru" : value;
  }

  function hideLocaleControls(root) {
    const doc = root || document;
    const style = doc.getElementById("rigintel-hide-locale");
    if (style) return;
    const el = doc.createElement("style");
    el.id = "rigintel-hide-locale";
    el.textContent = `
      [data-locale-picker],
      [data-testid="locale-select"],
      select[name="locale"],
      label[for="locale"] { display: none !important; }
    `;
    doc.head.appendChild(el);
  }

  function storeStudioSession(data) {
    if (!data || !data.access_token || !data.refresh_token) return false;
    localStorage.setItem(AUTH_TOKEN_KEY, data.access_token);
    localStorage.setItem(AUTH_REFRESH_TOKEN_KEY, data.refresh_token);
    localStorage.setItem(ONBOARDING_DONE_KEY, "1");
    if (data.must_change_password) {
      localStorage.setItem(AUTH_MUST_CHANGE_PASSWORD_KEY, "1");
    } else {
      localStorage.removeItem(AUTH_MUST_CHANGE_PASSWORD_KEY);
    }
    return true;
  }

  async function bootstrapStudioAuth() {
    if (localStorage.getItem(AUTH_TOKEN_KEY)) return false;
    const res = await fetch("/app/api/v1/studio/session", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
    });
    if (!res.ok) return false;
    const body = await res.json();
    return storeStudioSession(body && body.data);
  }

  async function boot() {
    const path = window.location.pathname;
    if (path === "/app" || path.startsWith("/app/")) {
      window.location.replace("/hub");
      return;
    }
    applyLocale(DEFAULT_LOCALE);
    hideLocaleControls(document);
    fetch("/app/api/v1/auth/me", { credentials: "include" })
      .then((res) => (res.ok ? res.json() : null))
      .then((body) => {
        const locale = body && body.data && body.data.ui_locale;
        if (locale) applyLocale(locale);
      })
      .catch(() => undefined);
    try {
      const ready = await bootstrapStudioAuth();
      if (ready && path !== "/hub" && path !== "/chat") {
        window.location.replace("/hub");
      }
    } catch {
      /* keep Studio's own login if bootstrap fails */
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    void boot();
  }
})();
