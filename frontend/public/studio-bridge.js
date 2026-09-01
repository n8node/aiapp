(() => {
  const STORAGE_KEY = "unsloth_locale";
  const AUTH_TOKEN_KEY = "unsloth_auth_token";
  const AUTH_REFRESH_TOKEN_KEY = "unsloth_auth_refresh_token";
  const ONBOARDING_DONE_KEY = "unsloth_onboarding_done";
  const AUTH_MUST_CHANGE_PASSWORD_KEY = "unsloth_auth_must_change_password";
  const DEFAULT_LOCALE = "ru";
  const HOME = "/chat";
  const PHRASES_URL = "/app/studio-i18n-ru.json";

  function pinLocale(locale) {
    const value = locale || DEFAULT_LOCALE;
    try {
      localStorage.setItem(STORAGE_KEY, value);
    } catch {
      /* ignore quota / private mode */
    }
    if (document.documentElement) {
      document.documentElement.lang = value;
    }
  }

  pinLocale(DEFAULT_LOCALE);

  function hideLocaleControls(root) {
    const doc = root || document;
    const style = doc.getElementById("rigintel-hide-locale");
    if (style) return;
    const el = doc.createElement("style");
    el.id = "rigintel-hide-locale";
    el.textContent = `
      [data-locale-picker],
      [data-testid="locale-select"],
      [data-testid="language-select"],
      select[name="locale"],
      label[for="locale"] { display: none !important; }
    `;
    doc.head.appendChild(el);
  }

  function looksLatin(text) {
    return /[A-Za-z]/.test(text) && !/[А-Яа-яЁё]/.test(text);
  }

  function translateValue(raw, phrases) {
    if (!raw) return raw;
    const trimmed = raw.trim();
    if (!trimmed || !looksLatin(trimmed)) return raw;
    const ru = phrases[trimmed];
    if (!ru) return raw;
    return raw.replace(trimmed, ru);
  }

  function translateNode(root, phrases) {
    if (!root || !phrases) return;
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
      acceptNode(node) {
        const parent = node.parentElement;
        if (!parent) return NodeFilter.FILTER_REJECT;
        const tag = parent.tagName;
        if (tag === "SCRIPT" || tag === "STYLE" || tag === "CODE" || tag === "PRE" || tag === "TEXTAREA" || tag === "NOSCRIPT") {
          return NodeFilter.FILTER_REJECT;
        }
        return NodeFilter.FILTER_ACCEPT;
      },
    });
    const nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    for (const node of nodes) {
      const next = translateValue(node.nodeValue, phrases);
      if (next !== node.nodeValue) node.nodeValue = next;
    }
    if (root.querySelectorAll) {
      root.querySelectorAll("[title],[aria-label],[placeholder],[alt]").forEach((el) => {
        ["title", "aria-label", "placeholder", "alt"].forEach((attr) => {
          if (!el.hasAttribute(attr)) return;
          const next = translateValue(el.getAttribute(attr), phrases);
          if (next !== el.getAttribute(attr)) el.setAttribute(attr, next);
        });
      });
    }
  }

  function startPhraseOverlay() {
    fetch(PHRASES_URL, { credentials: "same-origin" })
      .then((res) => (res.ok ? res.json() : null))
      .then((phrases) => {
        if (!phrases || typeof phrases !== "object") return;
        const apply = (node) => translateNode(node || document.body, phrases);
        apply(document.body);
        const obs = new MutationObserver((mutations) => {
          for (const m of mutations) {
            if (m.type === "characterData") apply(m.target.parentElement || document.body);
            m.addedNodes.forEach((n) => {
              if (n.nodeType === 1 || n.nodeType === 3) apply(n.nodeType === 3 ? n.parentElement : n);
            });
          }
        });
        obs.observe(document.documentElement, {
          childList: true,
          subtree: true,
          characterData: true,
        });
      })
      .catch(() => undefined);
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
      window.location.replace(HOME);
      return;
    }
    if (path === "/hub" || path === "/hub/") {
      window.location.replace(HOME);
      return;
    }
    hideLocaleControls(document);
    startPhraseOverlay();
    try {
      const res = await fetch("/app/api/v1/auth/me", { credentials: "include" });
      const body = res.ok ? await res.json() : null;
      const locale = body && body.data && body.data.ui_locale;
      if (locale && locale !== localStorage.getItem(STORAGE_KEY)) {
        pinLocale(locale);
        window.location.reload();
        return;
      }
      if (locale) pinLocale(locale);
    } catch {
      /* keep pinned default */
    }
    try {
      const ready = await bootstrapStudioAuth();
      if (ready && path !== HOME && path !== "/studio" && path !== "/export") {
        window.location.replace(HOME);
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
