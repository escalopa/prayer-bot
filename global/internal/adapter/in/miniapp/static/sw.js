"use strict";

const cacheName = "global-prayer-miniapp-shell-v11";
const shellAssets = [
  "./",
  "./app.css",
  "./app.js",
  "https://telegram.org/js/telegram-web-app.js?63",
];

self.addEventListener("install", (event) => {
  event.waitUntil((async () => {
    const cache = await caches.open(cacheName);
    await Promise.allSettled(shellAssets.map((asset) => cache.add(
      asset.startsWith("https://") ? new Request(asset, { mode: "no-cors" }) : asset,
    )));
    await self.skipWaiting();
  })());
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    const names = await caches.keys();
    await Promise.all(names
      .filter((name) => name.startsWith("global-prayer-miniapp-shell-") && name !== cacheName)
      .map((name) => caches.delete(name)));
    await self.clients.claim();
  })());
});

self.addEventListener("fetch", (event) => {
  if (event.request.method !== "GET") return;
  const url = new URL(event.request.url);
  const isAppShell = url.origin === self.location.origin && url.pathname.startsWith("/app/");
  const isTelegramSDK = url.origin === "https://telegram.org" &&
    url.pathname === "/js/telegram-web-app.js";
  if (!isAppShell && !isTelegramSDK) return;

  // The Telegram SDK is a stable, versioned URL, so cache-first avoids a network
  // round-trip and never goes stale.
  if (isTelegramSDK) {
    event.respondWith((async () => {
      const cached = await caches.match(event.request, { ignoreSearch: true });
      if (cached) return cached;
      const response = await fetch(event.request);
      if (response.ok || response.type === "opaque") {
        const cache = await caches.open(cacheName);
        await cache.put(event.request, response.clone());
      }
      return response;
    })());
    return;
  }

  // The app shell (HTML, CSS, JS) is network-first: a fresh deploy is visible on
  // the next launch without any manual cache clearing, while the cached copy is
  // still used as an offline fallback so a previously opened Mini App can start
  // during a network interruption.
  event.respondWith((async () => {
    try {
      const response = await fetch(event.request);
      if (response.ok) {
        const cache = await caches.open(cacheName);
        await cache.put(event.request, response.clone());
      }
      return response;
    } catch (error) {
      const cached = await caches.match(event.request);
      if (cached) return cached;
      throw error;
    }
  })());
});
