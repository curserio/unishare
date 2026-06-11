const STORAGE_KEY = "unishare.language";
const languages = new Set(["system", "ru", "en"]);

const dictionaries = {
  en: {
    appTitle: "Unishare",
    brandTagline: "Personal dropbox for links, text, and files",
    menu: "Menu",
    settings: "Settings",
    logout: "Log out",
    languageTitle: "Language",
    languageDescription: "Use your system language by default.",
    appearanceTitle: "Appearance",
    appearanceDescription: "Unishare follows your system theme by default.",
    themeTitle: "Theme",
    optionSystem: "System",
    optionLight: "Light",
    optionDark: "Dark",
    loginTitle: "Sign in",
    loginDescription: "Enter the access code once on every device, profile, or space.",
    accessCode: "Access code",
    signIn: "Sign in",
    textOrLink: "Link or text",
    textPlaceholder: "Paste anything you want to pass along",
    paste: "Paste",
    pasteFromClipboard: "Paste from clipboard",
    files: "Files",
    addToBuffer: "Add to buffer",
    latest: "Latest",
    loading: "Loading",
    refresh: "Refresh",
    emptyBuffer: "The buffer is empty. Add a link, text, or file.",
    share: "Share",
    copy: "Copy",
    delete: "Delete",
    mixed: "Mixed",
    file: "File",
    filePlural: "Files",
    link: "Link",
    text: "Text",
    noItems: "No items",
    itemOne: "{count} item",
    itemFew: "{count} items",
    itemMany: "{count} items",
    bytes: "B",
    kilobytes: "KB",
    megabytes: "MB",
    invalidToken: "Invalid access code",
    addContentFirst: "Add text, a link, or a file",
    added: "Added",
    addFailed: "Could not add",
    serverUnavailable: "Server is unavailable",
    notSignedIn: "Not signed in",
    copied: "Copied",
    actionFailed: "Could not complete the action",
  },
  ru: {
    appTitle: "Unishare",
    brandTagline: "Личный буфер для ссылок, текста и файлов",
    menu: "Меню",
    settings: "Настройки",
    logout: "Выйти",
    languageTitle: "Язык",
    languageDescription: "По умолчанию используется язык системы.",
    appearanceTitle: "Внешний вид",
    appearanceDescription: "По умолчанию Unishare использует тему системы.",
    themeTitle: "Тема",
    optionSystem: "Система",
    optionLight: "Светлая",
    optionDark: "Темная",
    loginTitle: "Вход",
    loginDescription: "Введите код доступа один раз на каждом устройстве, в профиле или пространстве.",
    accessCode: "Код доступа",
    signIn: "Войти",
    textOrLink: "Ссылка или текст",
    textPlaceholder: "Вставьте то, что нужно передать дальше",
    paste: "Вставить",
    pasteFromClipboard: "Вставить из буфера",
    files: "Файлы",
    addToBuffer: "Положить в буфер",
    latest: "Последнее",
    loading: "Загрузка",
    refresh: "Обновить",
    emptyBuffer: "Буфер пуст. Добавьте ссылку, текст или файл.",
    share: "Поделиться",
    copy: "Копировать",
    delete: "Удалить",
    mixed: "Смешанное",
    file: "Файл",
    filePlural: "Файлы",
    link: "Ссылка",
    text: "Текст",
    noItems: "Нет элементов",
    itemOne: "{count} элемент",
    itemFew: "{count} элемента",
    itemMany: "{count} элементов",
    bytes: "Б",
    kilobytes: "КБ",
    megabytes: "МБ",
    invalidToken: "Неверный код доступа",
    addContentFirst: "Добавьте текст, ссылку или файл",
    added: "Добавлено",
    addFailed: "Не удалось добавить",
    serverUnavailable: "Сервер недоступен",
    notSignedIn: "Нет входа",
    copied: "Скопировано",
    actionFailed: "Не удалось выполнить действие",
  },
};

export function initI18n(onChange) {
  const selected = normalize(localStorage.getItem(STORAGE_KEY));
  applyDocumentLanguage(resolveLanguage(selected));
  return {
    get: getLanguage,
    set(value) {
      const language = normalize(value);
      localStorage.setItem(STORAGE_KEY, language);
      applyDocumentLanguage(resolveLanguage(language));
      onChange?.(language);
    },
    resolved() {
      return resolveLanguage(getLanguage());
    },
    t(key, params) {
      return translate(resolveLanguage(getLanguage()), key, params);
    },
    render(root = document) {
      const resolved = resolveLanguage(getLanguage());
      applyDocumentLanguage(resolved);
      renderStaticText(root, resolved);
    },
  };
}

export function getLanguage() {
  return normalize(localStorage.getItem(STORAGE_KEY));
}

export function resolveLanguage(language) {
  if (language === "ru" || language === "en") return language;
  return navigator.language?.toLowerCase().startsWith("ru") ? "ru" : "en";
}

function normalize(value) {
  return languages.has(value) ? value : "system";
}

function translate(language, key, params = {}) {
  const template = dictionaries[language]?.[key] ?? dictionaries.en[key] ?? key;
  return template.replace(/\{(\w+)\}/g, (_, name) => String(params[name] ?? ""));
}

function renderStaticText(root, language) {
  for (const element of root.querySelectorAll("[data-i18n]")) {
    element.textContent = translate(language, element.dataset.i18n);
  }
  for (const element of root.querySelectorAll("[data-i18n-placeholder]")) {
    element.setAttribute("placeholder", translate(language, element.dataset.i18nPlaceholder));
  }
  for (const element of root.querySelectorAll("[data-i18n-aria-label]")) {
    element.setAttribute("aria-label", translate(language, element.dataset.i18nAriaLabel));
  }
  for (const element of root.querySelectorAll("[data-i18n-title]")) {
    element.setAttribute("title", translate(language, element.dataset.i18nTitle));
  }
  document.title = translate(language, "appTitle");
}

function applyDocumentLanguage(language) {
  document.documentElement.lang = language;
}
