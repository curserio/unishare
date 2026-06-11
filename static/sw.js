const CACHE = "unishare-v3";
const BASE = new URL(self.registration.scope).pathname.replace(/\/$/, "");
const ASSETS = [
  "/",
  "/styles.css",
  "/config.js",
  "/js/api.js",
  "/js/app.js",
  "/js/i18n.js",
  "/js/theme.js",
  "/js/ui.js",
  "/manifest.webmanifest",
  "/icons/icon-32.png",
  "/icons/icon-180.png",
  "/icons/icon-192.png",
  "/icons/icon-512.png",
  "/icons/icon-maskable-512.png",
].map((path) => `${BASE}${path}`);

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(ASSETS)));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((key) => key !== CACHE).map((key) => caches.delete(key))),
    ),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;
  const path = new URL(request.url).pathname;
  if (path.startsWith(`${BASE}/api/`)) return;
  event.respondWith(
    fetch(request)
      .then((response) => {
        const copy = response.clone();
        caches.open(CACHE).then((cache) => cache.put(request, copy));
        return response;
      })
      .catch(() => caches.match(request)),
  );
});
