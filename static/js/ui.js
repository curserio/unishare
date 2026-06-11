export function renderItems({ items, itemsEl, countEl, i18n, onShare, onCopy, onDelete }) {
  countEl.textContent = countLabel(items.length, i18n);
  if (items.length === 0) {
    const empty = document.createElement("article");
    empty.className = "empty";
    empty.textContent = i18n.t("emptyBuffer");
    itemsEl.replaceChildren(empty);
    return;
  }
  itemsEl.replaceChildren(
    ...items.map((item) => renderItem(item, { i18n, onShare, onCopy, onDelete })),
  );
}

export function setBusy(form, busy) {
  for (const element of form.querySelectorAll("button, input, textarea")) {
    element.disabled = busy;
  }
}

export function setStatus(statusEl, message) {
  statusEl.textContent = message;
  window.clearTimeout(setStatus.timer);
  setStatus.timer = window.setTimeout(() => {
    statusEl.textContent = "";
  }, 3200);
}

function renderItem(item, actions) {
  const article = document.createElement("article");
  article.className = "item";

  const meta = document.createElement("div");
  meta.className = "item-meta";
  const kind = document.createElement("span");
  kind.className = "item-kind";
  kind.textContent = itemKind(item, actions.i18n);
  const time = document.createElement("time");
  time.dateTime = item.createdAt;
  time.textContent = formatDate(item.createdAt, actions.i18n);
  meta.append(kind, time);

  const body = document.createElement("div");
  body.className = "item-body";

  if (item.title) {
    const title = document.createElement("h3");
    title.textContent = item.title;
    body.append(title);
  }
  if (item.text) {
    const text = document.createElement("p");
    text.className = "item-preview";
    text.textContent = item.text;
    body.append(text);
  }
  if (item.url) {
    const link = document.createElement("a");
    link.href = item.url;
    link.textContent = item.url;
    link.rel = "noreferrer";
    body.append(link);
  }
  if (item.files?.length) {
    const files = document.createElement("div");
    files.className = "file-list";
    for (const file of item.files) {
      const link = document.createElement("a");
      const basePath = window.__UNISHARE_CONFIG__?.basePath || "";
      link.href = `${basePath}/files/${item.id}/${file.id}`;
      link.textContent = `${file.name} (${formatBytes(file.size, actions.i18n)})`;
      link.className = "file-link";
      files.append(link);
    }
    body.append(files);
  }

  const actionBar = document.createElement("div");
  actionBar.className = "actions";
  actionBar.append(
    actionButton(actions.i18n.t("share"), "secondary-button", () => actions.onShare(item)),
    actionButton(actions.i18n.t("copy"), "secondary-button", () => actions.onCopy(item)),
    actionButton(actions.i18n.t("delete"), "danger-button", () => actions.onDelete(item)),
  );

  article.append(meta, body, actionBar);
  return article;
}

function actionButton(label, className, onClick) {
  const el = document.createElement("button");
  el.type = "button";
  el.className = className;
  el.textContent = label;
  el.addEventListener("click", onClick);
  return el;
}

function itemKind(item, i18n) {
  if (item.files?.length && (item.text || item.url)) return i18n.t("mixed");
  if (item.files?.length) return item.files.length === 1 ? i18n.t("file") : i18n.t("filePlural");
  if (item.url) return i18n.t("link");
  return i18n.t("text");
}

function countLabel(count, i18n) {
  if (count === 0) return i18n.t("noItems");
  const locale = i18n.resolved() === "ru" ? "ru-RU" : "en-US";
  const category = new Intl.PluralRules(locale).select(count);
  if (category === "one") return i18n.t("itemOne", { count });
  if (category === "few") return i18n.t("itemFew", { count });
  return i18n.t("itemMany", { count });
}

function formatDate(value, i18n) {
  return new Date(value).toLocaleString(i18n.resolved() === "ru" ? "ru-RU" : "en-US", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatBytes(size, i18n) {
  const locale = i18n.resolved() === "ru" ? "ru-RU" : "en-US";
  const number = new Intl.NumberFormat(locale, { maximumFractionDigits: 1 });
  if (size < 1024) return `${size} ${i18n.t("bytes")}`;
  if (size < 1024 * 1024) return `${number.format(Math.round(size / 1024))} ${i18n.t("kilobytes")}`;
  return `${number.format(size / 1024 / 1024)} ${i18n.t("megabytes")}`;
}
