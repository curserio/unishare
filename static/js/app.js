import { createItem, deleteItem, getSession, listItems, login, logout } from "./api.js";
import { initI18n } from "./i18n.js";
import { initTheme } from "./theme.js";
import { renderItems, setBusy, setStatus } from "./ui.js";

const loginView = document.querySelector("#loginView");
const appView = document.querySelector("#appView");
const loginForm = document.querySelector("#loginForm");
const shareForm = document.querySelector("#shareForm");
const pasteButton = document.querySelector("#pasteButton");
const itemsEl = document.querySelector("#items");
const itemsCount = document.querySelector("#itemsCount");
const statusEl = document.querySelector("#status");
const logoutButton = document.querySelector("#logoutButton");
const refreshButton = document.querySelector("#refreshButton");
const settingsButton = document.querySelector("#settingsButton");
const settingsPanel = document.querySelector("#settingsPanel");
const themeInputs = [...document.querySelectorAll("input[name='theme']")];
const languageInputs = [...document.querySelectorAll("input[name='language']")];
const textInput = document.querySelector("#textInput");

const i18n = initI18n();
const theme = initTheme(syncThemeInputs);
i18n.render();
syncLanguageInputs(i18n.get());
syncThemeInputs(theme.get());

if ("serviceWorker" in navigator) {
  const basePath = window.__UNISHARE_CONFIG__?.basePath || "";
  navigator.serviceWorker.register(`${basePath}/sw.js`, { scope: `${basePath}/` }).catch(() => {});
}

for (const input of themeInputs) {
  input.addEventListener("change", () => theme.set(input.value));
}

for (const input of languageInputs) {
  input.addEventListener("change", () => {
    i18n.set(input.value);
    syncLanguageInputs(i18n.get());
  });
}

loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const token = new FormData(loginForm).get("token");
  setBusy(loginForm, true);
  try {
    await login(token);
    loginForm.reset();
    await showApp();
  } catch {
    setStatus(statusEl, i18n.t("invalidToken"));
  } finally {
    setBusy(loginForm, false);
  }
});

logoutButton.addEventListener("click", async () => {
  await logout();
  closeSettings();
  showLogin();
});

refreshButton.addEventListener("click", () => loadItems());

settingsButton.addEventListener("click", () => {
  const open = settingsPanel.classList.toggle("hidden") === false;
  settingsButton.setAttribute("aria-expanded", String(open));
});

document.addEventListener("click", (event) => {
  if (settingsPanel.classList.contains("hidden")) return;
  if (settingsPanel.contains(event.target) || settingsButton.contains(event.target)) return;
  closeSettings();
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") closeSettings();
});

shareForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = new FormData(shareForm);
  const text = String(form.get("text") || "").trim();
  const files = shareForm.querySelector("input[type=file]").files;
  if (!text && files.length === 0) {
    setStatus(statusEl, i18n.t("addContentFirst"));
    return;
  }
  setBusy(shareForm, true);
  try {
    await createItem(form);
    shareForm.reset();
    setStatus(statusEl, i18n.t("added"));
    await loadItems();
  } catch (error) {
    setStatus(statusEl, error.message || i18n.t("addFailed"));
  } finally {
    setBusy(shareForm, false);
  }
});

pasteButton.addEventListener("click", async () => {
  const text = await readClipboardText();
  if (!text) {
    await updatePasteButton();
    return;
  }
  insertClipboardText(text);
});

textInput.addEventListener("focus", () => updatePasteButton());
textInput.addEventListener("input", () => updatePasteButton());
document.addEventListener("visibilitychange", () => {
  if (!document.hidden) updatePasteButton();
});

boot();

async function boot() {
  try {
    const session = await getSession();
    if (session.authenticated) {
      await showApp();
    } else {
      showLogin();
    }
  } catch {
    showLogin();
    setStatus(statusEl, i18n.t("serverUnavailable"));
  }
}

function showLogin() {
  loginView.classList.remove("hidden");
  appView.classList.add("hidden");
  logoutButton.classList.add("hidden");
  itemsEl.innerHTML = "";
  itemsCount.textContent = i18n.t("notSignedIn");
}

async function showApp() {
  loginView.classList.add("hidden");
  appView.classList.remove("hidden");
  logoutButton.classList.remove("hidden");
  await updatePasteButton();
  await loadItems();
}

async function loadItems() {
  refreshButton.disabled = true;
  itemsCount.textContent = i18n.t("loading");
  try {
    const items = await listItems();
    renderItems({
      items,
      itemsEl,
      countEl: itemsCount,
      i18n,
      onShare: safeAction(shareItem),
      onCopy: safeAction(copyItem),
      onDelete: safeAction(removeItem),
    });
  } catch {
    showLogin();
  } finally {
    refreshButton.disabled = false;
  }
}

async function shareItem(item) {
  if (navigator.share) {
    await navigator.share({
      title: item.title || "Unishare",
      text: item.shareText,
      url: item.url || undefined,
    });
    return;
  }
  await copyItem(item);
}

async function copyItem(item) {
  await navigator.clipboard.writeText(item.shareText);
  setStatus(statusEl, i18n.t("copied"));
}

async function removeItem(item) {
  await deleteItem(item.id);
  await loadItems();
}

function safeAction(action) {
  return async (item) => {
    try {
      await action(item);
    } catch {
      setStatus(statusEl, i18n.t("actionFailed"));
    }
  };
}

function syncLanguageInputs(value) {
  for (const input of languageInputs) {
    input.checked = input.value === value;
  }
  i18n.render();
  if (!appView.classList.contains("hidden")) loadItems();
}

function syncThemeInputs(value) {
  for (const input of themeInputs) {
    input.checked = input.value === value;
  }
}

function closeSettings() {
  settingsPanel.classList.add("hidden");
  settingsButton.setAttribute("aria-expanded", "false");
}

async function updatePasteButton() {
  if (appView.classList.contains("hidden")) {
    pasteButton.classList.add("hidden");
    return;
  }
  const text = await readClipboardText();
  pasteButton.classList.toggle("hidden", !text);
}

async function readClipboardText() {
  if (!navigator.clipboard?.readText) return "";
  try {
    return (await navigator.clipboard.readText()).trim();
  } catch {
    return "";
  }
}

function insertClipboardText(text) {
  const start = textInput.selectionStart ?? textInput.value.length;
  const end = textInput.selectionEnd ?? textInput.value.length;
  const separator = textInput.value && start === end ? (textInput.value.endsWith("\n") ? "" : "\n") : "";
  const nextValue = textInput.value.slice(0, start) + separator + text + textInput.value.slice(end);
  textInput.value = nextValue;
  const cursor = start + separator.length + text.length;
  textInput.setSelectionRange(cursor, cursor);
  textInput.focus();
  updatePasteButton();
}
