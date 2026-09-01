(() => {
  const STORAGE_KEY = "unsloth_locale";
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

  function boot() {
    const path = window.location.pathname;
    if (path === "/login" || path === "/onboarding" || path.startsWith("/login/") || path.startsWith("/onboarding/") || path.startsWith("/change-password")) {
      window.location.replace("/app/training");
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
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
