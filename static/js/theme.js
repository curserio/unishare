const STORAGE_KEY = "unishare.theme";
const themes = new Set(["system", "light", "dark"]);
const darkMedia = window.matchMedia("(prefers-color-scheme: dark)");

export function initTheme(onChange) {
  const selected = normalize(localStorage.getItem(STORAGE_KEY));
  applyTheme(selected);
  darkMedia.addEventListener("change", () => {
    if (getTheme() === "system") applyTheme("system");
  });
  return {
    get: getTheme,
    set(value) {
      const theme = normalize(value);
      localStorage.setItem(STORAGE_KEY, theme);
      applyTheme(theme);
      onChange?.(theme);
    },
  };
}

export function getTheme() {
  return normalize(localStorage.getItem(STORAGE_KEY));
}

function normalize(value) {
  return themes.has(value) ? value : "system";
}

function applyTheme(theme) {
  const root = document.documentElement;
  if (theme === "system") {
    root.removeAttribute("data-theme");
  } else {
    root.dataset.theme = theme;
  }
  const resolved = theme === "system" ? (darkMedia.matches ? "dark" : "light") : theme;
  document.querySelector("meta[name='theme-color']")?.setAttribute(
    "content",
    resolved === "dark" ? "#111513" : "#f6f7f4",
  );
}
