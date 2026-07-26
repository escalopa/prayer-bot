(() => {
  "use strict";

  const telegram = window.Telegram && window.Telegram.WebApp;
  let initData = "";
  let state = null;
  let activeDay = "today";
  let toastTimer = null;
  let dirty = false;
  let compassStarted = false;
  let calendarFeedURL = "";
  let offlineMode = false;
  let homeScreenStatus = "unknown";
  let zakatCurrency = "";
  let activeView = "prayer";
  let placesDay = "today";
  let placesLookupTimer = null;
  const places = { z: 5, lat: 21.4225, lng: 39.8262, initialized: false, lastKey: "" };
  const troyOunceGrams = 31.1035;
  const offlineCacheVersion = 2;
  const offlineCacheMaxAge = 48 * 60 * 60 * 1000;

  const launchCopy = {
    en: { open: "Open this page from the bot inside Telegram.", expired: "Your Telegram session expired. Close this window and reopen the app from the bot menu.", failed: "The app could not load. Please try again.", retry: "Try again" },
    ar: { open: "افتح هذه الصفحة من البوت داخل تيليجرام.", expired: "انتهت جلسة تيليجرام. أغلق هذه النافذة وافتح التطبيق مجددًا من قائمة البوت.", failed: "تعذر تحميل التطبيق. حاول مرة أخرى.", retry: "حاول مرة أخرى" },
    es: { open: "Abre esta página desde el bot en Telegram.", expired: "Tu sesión de Telegram caducó. Cierra esta ventana y vuelve a abrir la app desde el menú del bot.", failed: "No se pudo cargar la app. Inténtalo de nuevo.", retry: "Intentar de nuevo" },
    fr: { open: "Ouvrez cette page depuis le bot dans Telegram.", expired: "Votre session Telegram a expiré. Fermez cette fenêtre et rouvrez l’application depuis le menu du bot.", failed: "Impossible de charger l’application. Réessayez.", retry: "Réessayer" },
    ru: { open: "Откройте эту страницу через бота в Telegram.", expired: "Сессия Telegram истекла. Закройте окно и снова откройте приложение из меню бота.", failed: "Не удалось загрузить приложение. Попробуйте ещё раз.", retry: "Повторить" },
    tr: { open: "Bu sayfayı Telegram’daki bot üzerinden açın.", expired: "Telegram oturumunuz sona erdi. Bu pencereyi kapatıp uygulamayı bot menüsünden yeniden açın.", failed: "Uygulama yüklenemedi. Tekrar deneyin.", retry: "Tekrar dene" },
    uz: { open: "Bu sahifani Telegram ichidagi botdan oching.", expired: "Telegram seansi tugadi. Oynani yoping va ilovani bot menyusidan qayta oching.", failed: "Ilovani yuklab bo‘lmadi. Qayta urinib ko‘ring.", retry: "Qayta urinish" },
    tt: { open: "Бу битне Telegram эчендәге боттан ачыгыз.", expired: "Telegram сеансы тәмамланды. Тәрәзәне ябып, кушымтаны бот менюсыннан яңадан ачыгыз.", failed: "Кушымтаны йөкләп булмады. Кабатлап карагыз.", retry: "Кабатлау" },
  };

  const byId = (id) => document.getElementById(id);
  const loading = byId("loading");
  const standalone = byId("standalone");
  const locationGate = byId("location-gate");
  const dashboard = byId("dashboard");

  function currentInitData() {
    if (telegram && telegram.initData) initData = telegram.initData;
    if (!initData && window.location.hash) {
      initData = new URLSearchParams(window.location.hash.slice(1)).get("tgWebAppData") || "";
    }
    return initData;
  }

  function launchLanguage() {
    const code = telegram && telegram.initDataUnsafe && telegram.initDataUnsafe.user
      ? telegram.initDataUnsafe.user.language_code : "en";
    return String(code || "en").toLowerCase().split("-")[0];
  }

  if (telegram) {
    telegram.ready();
    telegram.expand();
    try {
      telegram.setHeaderColor("secondary_bg_color");
      telegram.setBackgroundColor("bg_color");
    } catch (_) {
      // Older Telegram clients still render correctly with CSS theme variables.
    }
  }

  if ("serviceWorker" in navigator) {
    window.addEventListener("load", () => {
      navigator.serviceWorker.register("./sw.js", { scope: "./" }).catch(() => {
        // Offline shell caching is an enhancement; live Mini App use continues.
      });
    });
  }

  async function request(path, method = "POST", body) {
    const signedData = currentInitData();
    const response = await fetch(path, {
      method,
      headers: {
        "Content-Type": "application/json",
        "X-Telegram-Init-Data": signedData,
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: "same-origin",
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(data.error || "temporary_failure");
      error.code = data.error;
      error.status = response.status;
      throw error;
    }
    return data;
  }

  function telegramUserID() {
    const user = telegram && telegram.initDataUnsafe && telegram.initDataUnsafe.user;
    return user && Number.isSafeInteger(Number(user.id)) ? String(user.id) : "";
  }

  function offlineCacheKey() {
    const userID = telegramUserID();
    return userID ? `global-prayer-miniapp-v${offlineCacheVersion}-${userID}` : "";
  }

  function telegramVersionAtLeast(version) {
    return Boolean(telegram && telegram.isVersionAtLeast && telegram.isVersionAtLeast(version));
  }

  function deviceStorageAvailable() {
    return Boolean(telegramVersionAtLeast("9.0") && telegram.DeviceStorage);
  }

  function readStoredValue(key) {
    if (!key) return Promise.resolve("");
    if (deviceStorageAvailable()) {
      return new Promise((resolve) => {
        telegram.DeviceStorage.getItem(key, (error, value) => resolve(error ? "" : (value || "")));
      });
    }
    try {
      return Promise.resolve(window.localStorage.getItem(key) || "");
    } catch (_) {
      return Promise.resolve("");
    }
  }

  function writeStoredValue(key, value) {
    if (!key) return Promise.resolve();
    if (deviceStorageAvailable()) {
      return new Promise((resolve) => {
        telegram.DeviceStorage.setItem(key, value, () => resolve());
      });
    }
    try {
      window.localStorage.setItem(key, value);
    } catch (_) {
      // A denied or full browser store must never prevent the live app loading.
    }
    return Promise.resolve();
  }

  function offlineSnapshot(next) {
    if (!next || next.needs_location || !next.today || !next.tomorrow || !next.labels) return null;
    const snapshot = JSON.parse(JSON.stringify(next));
    // The cache key is already scoped to Telegram's signed user. The numeric
    // account identifier is unnecessary for rendering and is not persisted.
    if (snapshot.user) delete snapshot.user.id;
    // The calendar path is a bearer credential and must remain memory-only.
    if (snapshot.calendar) delete snapshot.calendar.path;
    return snapshot;
  }

  async function cacheState(next) {
    const snapshot = offlineSnapshot(next);
    if (!snapshot) return;
    await writeStoredValue(offlineCacheKey(), JSON.stringify({
      saved_at: Date.now(),
      state: snapshot,
    }));
  }

  async function cachedState() {
    try {
      const stored = JSON.parse(await readStoredValue(offlineCacheKey()));
      const age = Date.now() - Number(stored.saved_at);
      if (!stored.state || age < 0 || age > offlineCacheMaxAge || !offlineSnapshot(stored.state)) return null;
      return stored;
    } catch (_) {
      return null;
    }
  }

  function setText(id, value) {
    const element = byId(id);
    if (element) element.textContent = value || "";
  }

  function applyLabels(labels) {
    document.documentElement.lang = state.locale;
    document.documentElement.dir = state.locale === "ar" ? "rtl" : "ltr";
    document.title = labels.app_title;
    setText("eyebrow", labels.companion);
    setText("app-title", labels.app_title);
    setText("user-name", state.user.first_name || "");
    setText("loading-text", labels.loading);
    setText("standalone-text", labels.open_in_telegram);
    setText("today-tab", labels.today);
    setText("tomorrow-tab", labels.tomorrow);
    setText("location-title", labels.location);
    setText("location-help", labels.location_help);
    setText("location-primary", labels.share_location);
    setText("location-secondary", labels.update_location);
    setText("reminders-title", labels.reminders);
    setText("settings-title", labels.settings);
    setText("prayer-reminders-label", labels.prayer_reminders);
    setText("pre-prayer-reminder-label", labels.pre_prayer_reminder);
    setText("fasting-reminders-label", labels.fasting_reminders);
    setText("kahf-reminders-label", labels.kahf_reminders);
    setText("fasting-schedule", labels.fasting_schedule);
    setText("kahf-schedule", labels.kahf_schedule);
    setText("occasions-title", labels.occasions_title);
    setText("occasions-help", labels.occasions_help);
    setText("occasions-disclaimer", labels.occasions_disclaimer);
    setText("occasion-major-reminders-label", labels.occasion_major_reminders);
    setText("occasion-fasting-reminders-label", labels.occasion_fasting_reminders);
    setText("occasion-observed-reminders-label", labels.occasion_observed_reminders);
    ["occasion-major-schedule", "occasion-fasting-schedule", "occasion-observed-schedule"]
      .forEach((id) => setText(id, labels.occasion_schedule));
    setText("language-label", labels.language);
    setText("method-label", labels.method);
    setText("madhab-label", labels.madhab);
    setText("highlat-label", labels.highlat);
    setText("hijri-label", labels.hijri);
    setText("adjustments-label", labels.adjustments);
    setText("save-preferences", labels.save);
    setText("calculation-note", labels.calculated_locally);
    setText("tools-title", labels.tools);
    setText("qibla-title", labels.qibla_title);
    setText("qibla-help", labels.qibla_help);
    setText("start-compass", labels.compass_start);
    setText("calendar-title", labels.calendar_title);
    setText("calendar-help", labels.calendar_help);
    setText("calendar-private", labels.calendar_private);
    setText("connect-calendar", labels.calendar_connect);
    setText("copy-calendar-link", labels.calendar_copy);
    setText("disconnect-calendar", labels.calendar_disconnect);
    setText("home-screen-title", labels.home_title);
    setText("home-screen-help", labels.home_help);
    setText("add-home-screen", homeScreenStatus === "added" ? labels.home_added : labels.home_add);
    setText("share-card-title", labels.share_title);
    setText("share-card-help", labels.share_help);
    setText("share-prayer-card", labels.share_action);
    setText("zakat-title", labels.zakat_title);
    setText("zakat-help", labels.zakat_help);
    setText("zakat-currency-label", labels.zakat_currency);
    setText("zakat-holdings-label", labels.zakat_holdings);
    setText("zakat-nisab-label", labels.zakat_nisab);
    setText("zakat-due-label", labels.zakat_due);
    setText("zakat-disclaimer", labels.zakat_disclaimer);
    setText("tab-prayer", labels.nav_prayer);
    setText("tab-dates", labels.nav_dates);
    setText("tab-zakat", labels.nav_zakat);
    setText("tab-places", labels.nav_places);
    setText("tab-tools", labels.tools);
    setText("tab-settings", labels.settings);
    setText("places-title", labels.places_title);
    setText("places-help", labels.places_help);
    setText("places-today-tab", labels.today);
    setText("places-tomorrow-tab", labels.tomorrow);
    setText("places-note", labels.calculated_locally);
  }

  function fillSelect(id, options, selected) {
    const select = byId(id);
    select.replaceChildren();
    options.forEach((item) => {
      const option = document.createElement("option");
      option.value = item.value;
      option.textContent = item.label;
      option.selected = item.value === String(selected);
      select.append(option);
    });
  }

  function renderSettings() {
    const profile = state.profile;
    fillSelect("language", state.options.languages, state.locale);
    fillSelect("method", state.options.methods, profile.method);
    fillSelect("madhab", state.options.madhabs, profile.madhab);
    fillSelect("highlat", state.options.high_latitude, profile.high_latitude_rule);
    fillSelect("hijri-adjustment", [-2, -1, 0, 1, 2].map((value) => ({
      value: String(value), label: value > 0 ? `+${value}` : String(value),
    })), profile.hijri_adjustment);

    const names = {};
    [...state.today.prayers].forEach((prayer) => { names[prayer.id] = prayer.name; });
    const grid = byId("adjustment-grid");
    grid.replaceChildren();
    Object.entries(profile.adjustments).forEach(([prayer, value]) => {
      const label = document.createElement("label");
      label.textContent = names[prayer] || prayer;
      const input = document.createElement("input");
      input.type = "number";
      input.min = "-30";
      input.max = "30";
      input.step = "1";
      input.value = String(value);
      input.dataset.prayer = prayer;
      label.append(input);
      grid.append(label);
    });
  }

  function renderSchedule() {
    const schedule = state[activeDay];
    setText("gregorian-date", schedule.gregorian);
    setText("hijri-date", `☾ ${schedule.hijri}`);
    setText("timezone", schedule.timezone);
    const grid = byId("prayer-grid");
    grid.replaceChildren();
    schedule.prayers.forEach((prayer) => {
      const item = document.createElement("div");
      item.className = "prayer";
      const emoji = document.createElement("span");
      emoji.className = "prayer-emoji";
      emoji.textContent = prayer.emoji;
      const name = document.createElement("span");
      name.className = "prayer-name";
      name.textContent = prayer.name;
      const time = document.createElement("strong");
      time.className = "prayer-time";
      time.textContent = prayer.time;
      item.append(emoji, name, time);
      grid.append(item);
    });
    setText("share-preview-date", schedule.gregorian);
    const nextPrayer = schedule.prayers.find((prayer) => prayer.time) || schedule.prayers[0];
    setText("share-preview-time", nextPrayer ? `${nextPrayer.name} · ${nextPrayer.time}` : "");
  }

  function formatLabel(template, values) {
    return Object.entries(values).reduce(
      (result, [key, value]) => result.replaceAll(`{${key}}`, String(value)),
      template || "",
    );
  }

  function renderTools() {
    if (!state.qibla) return;
    const bearing = Number(state.qibla.bearing_degrees);
    const number = new Intl.NumberFormat(state.locale, { maximumFractionDigits: 1 });
    byId("qibla-needle").style.setProperty("--qibla-rotation", `${bearing}deg`);
    setText("qibla-bearing", formatLabel(state.labels.qibla_bearing, { bearing: number.format(bearing) }));
    setText("qibla-distance", formatLabel(state.labels.qibla_distance, {
      distance: new Intl.NumberFormat(state.locale).format(state.qibla.distance_kilometres),
    }));
    byId("qibla-compass").setAttribute("aria-label", byId("qibla-bearing").textContent);
    compassStarted = false;
    byId("start-compass").disabled = false;
    setText("start-compass", state.labels.compass_start);
    byId("compass-status").classList.add("hidden");
    renderCalendarSubscription();
    renderHomeScreen();
  }

  function renderCalendarSubscription() {
    const enabled = Boolean(state.calendar && state.calendar.enabled);
    byId("disconnect-calendar").classList.toggle("hidden", !enabled);
    byId("calendar-status").classList.add("hidden");
    setText("connect-calendar", state.labels.calendar_connect);
    setText("copy-calendar-link", state.labels.calendar_copy);
    setText("disconnect-calendar", state.labels.calendar_disconnect);
  }

  function updateHomeScreenStatus(status) {
    homeScreenStatus = status || "unknown";
    if (homeScreenStatus === "unsupported") {
      byId("home-screen-card").classList.add("hidden");
      return;
    }
    const button = byId("add-home-screen");
    const statusElement = byId("home-screen-status");
    const added = homeScreenStatus === "added";
    button.disabled = added;
    setText("add-home-screen", added ? state.labels.home_added : state.labels.home_add);
    setText("home-screen-status", added ? state.labels.home_added : "");
    statusElement.classList.toggle("hidden", !added);
  }

  function renderHomeScreen() {
    const supported = telegramVersionAtLeast("8.0") &&
      telegram && typeof telegram.addToHomeScreen === "function";
    byId("home-screen-card").classList.toggle("hidden", !supported);
    if (!supported) return;
    updateHomeScreenStatus(homeScreenStatus);
    if (typeof telegram.checkHomeScreenStatus === "function") {
      telegram.checkHomeScreenStatus((status) => updateHomeScreenStatus(status));
    }
  }

  function addToHomeScreen() {
    if (!telegram || typeof telegram.addToHomeScreen !== "function") return;
    telegram.addToHomeScreen();
  }

  function renderReminders() {
    byId("prayer-reminders").checked = state.reminders.prayer;
    fillSelect("pre-prayer-minutes", state.options.pre_reminders, state.reminders.pre_prayer_minutes);
    byId("fasting-reminders").checked = state.reminders.fasting;
    byId("kahf-reminders").checked = state.reminders.kahf;
    byId("occasion-major-reminders").checked = state.reminders.occasion_major;
    byId("occasion-fasting-reminders").checked = state.reminders.occasion_fasting;
    byId("occasion-observed-reminders").checked = state.reminders.occasion_observed;
    syncPreReminderAvailability();
  }

  function sourceLink(source) {
    let url;
    try {
      url = new URL(source.url);
    } catch (_) {
      return null;
    }
    if (url.protocol !== "https:") return null;
    const link = document.createElement("a");
    link.className = "occasion-source";
    link.href = url.href;
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.textContent = source.label;
    return link;
  }

  function renderOccasions() {
    const list = byId("occasion-list");
    list.replaceChildren();
    (state.occasions || []).forEach((occasion) => {
      const article = document.createElement("article");
      article.className = "occasion-card";

      const header = document.createElement("div");
      header.className = "occasion-card-heading";
      const emoji = document.createElement("span");
      emoji.className = "occasion-emoji";
      emoji.textContent = occasion.emoji;
      const heading = document.createElement("div");
      const category = document.createElement("span");
      category.className = `occasion-category occasion-category-${occasion.category}`;
      category.textContent = occasion.category_label;
      const title = document.createElement("h3");
      title.textContent = occasion.title;
      const dates = document.createElement("p");
      dates.className = "occasion-dates";
      dates.textContent = `${occasion.hijri} · ${occasion.gregorian}`;
      heading.append(category, title, dates);
      header.append(emoji, heading);

      const summary = document.createElement("p");
      summary.className = "occasion-summary";
      summary.textContent = occasion.summary;
      const recommendation = document.createElement("p");
      recommendation.className = "occasion-recommendation";
      const recommendationLabel = document.createElement("strong");
      recommendationLabel.textContent = `${state.labels.occasion_recommended}: `;
      recommendation.append(recommendationLabel, document.createTextNode(occasion.action));

      const sources = document.createElement("div");
      sources.className = "occasion-sources";
      (occasion.sources || []).forEach((source) => {
        const link = sourceLink(source);
        if (link) sources.append(link);
      });
      if (sources.childElementCount > 0) {
        sources.setAttribute("aria-label", state.labels.occasion_sources);
      }
      article.append(header, summary, recommendation, sources);
      list.append(article);
    });
  }

  function syncPreReminderAvailability() {
    byId("pre-prayer-minutes").disabled = !byId("prayer-reminders").checked;
  }

  function zakatCurrencyKey() {
    const userID = telegramUserID();
    return userID ? `global-prayer-zakat-currency-${userID}` : "";
  }

  function currencyName(code) {
    try {
      const name = new Intl.DisplayNames([state.locale], { type: "currency" }).of(code);
      return name && name !== code ? `${code} · ${name}` : code;
    } catch (_) {
      return code;
    }
  }

  function formatMoney(amount, currency) {
    try {
      return new Intl.NumberFormat(state.locale, {
        style: "currency", currency, maximumFractionDigits: 2,
      }).format(amount);
    } catch (_) {
      return `${new Intl.NumberFormat(state.locale, { maximumFractionDigits: 2 }).format(amount)} ${currency}`;
    }
  }

  function nisabValue(nisab, grams, usdPerOunce, currency) {
    const rate = Number(nisab.rates[currency]) || 1;
    return (grams / troyOunceGrams) * usdPerOunce * rate;
  }

  function renderZakat() {
    const panel = byId("zakat-panel");
    const tab = byId("tab-zakat-btn");
    const nisab = state.nisab;
    if (!nisab || !Array.isArray(nisab.currencies) || nisab.currencies.length === 0) {
      panel.classList.add("hidden");
      if (tab) tab.classList.add("hidden");
      if (activeView === "zakat") selectView("prayer");
      return;
    }
    if (tab) tab.classList.remove("hidden");
    panel.classList.remove("hidden");
    if (!zakatCurrency || !nisab.rates[zakatCurrency]) zakatCurrency = nisab.default_currency;
    const select = byId("zakat-currency");
    select.replaceChildren();
    nisab.currencies.forEach((code) => {
      const option = document.createElement("option");
      option.value = code;
      option.textContent = currencyName(code);
      option.selected = code === zakatCurrency;
      select.append(option);
    });
    if (nisab.fetched_at) {
      const date = new Intl.DateTimeFormat(state.locale, { dateStyle: "medium" }).format(new Date(nisab.fetched_at));
      setText("zakat-updated", formatLabel(state.labels.zakat_updated, { date }));
    } else {
      setText("zakat-updated", "");
    }
    renderZakatResult();
  }

  function renderZakatResult() {
    const nisab = state && state.nisab;
    if (!nisab) return;
    const currency = zakatCurrency;
    const gold = nisabValue(nisab, nisab.gold_grams, nisab.gold_usd_per_ounce, currency);
    const silver = nisabValue(nisab, nisab.silver_grams, nisab.silver_usd_per_ounce, currency);
    // The lower (usually silver) threshold is the applicable niSab in the
    // majority view, so more wealth qualifies for zakat.
    const threshold = Math.min(gold, silver);
    setText("zakat-nisab-value", formatMoney(threshold, currency));
    setText("zakat-nisab-note", formatLabel(state.labels.zakat_nisab_note, {
      gold: formatMoney(gold, currency), silver: formatMoney(silver, currency),
    }));
    const holdings = Number(byId("zakat-holdings").value);
    const wealth = Number.isFinite(holdings) && holdings > 0 ? holdings : 0;
    const due = wealth >= threshold ? wealth * 0.025 : 0;
    setText("zakat-due-value", formatMoney(due, currency));
    setText("zakat-status", wealth > 0 && wealth < threshold ? state.labels.zakat_below : "");
  }

  function applyState(next) {
    state = next;
    loading.classList.add("hidden");
    standalone.classList.add("hidden");
    byId("retry-app").classList.add("hidden");
    applyLabels(state.labels);
    if (state.needs_location) {
      dashboard.classList.add("hidden");
      locationGate.classList.remove("hidden");
      setDirty(false);
      return;
    }
    locationGate.classList.add("hidden");
    dashboard.classList.remove("hidden");
    renderSchedule();
    renderTools();
    renderOccasions();
    renderZakat();
    renderReminders();
    renderSettings();
    selectView(activeView);
    setDirty(false);
  }

  function setOnlineControlsDisabled(value) {
    offlineMode = value;
    byId("location-primary").disabled = value;
    byId("location-secondary").disabled = value;
    setPreferencesDisabled(value);
    setCalendarButtonsDisabled(value);
  }

  function showConnectionState(kind, savedAt) {
    const banner = byId("connection-banner");
    banner.classList.remove("hidden");
    banner.classList.toggle("refreshing", kind === "refreshing");
    if (kind === "refreshing") {
      setText("connection-title", state.labels.offline_updating);
      setText("connection-message", state.labels.offline_updating_help);
      return;
    }
    const time = new Intl.DateTimeFormat(state.locale, {
      hour: "2-digit", minute: "2-digit",
    }).format(new Date(savedAt));
    setText("connection-title", state.labels.offline_title);
    setText("connection-message", formatLabel(state.labels.offline_help, { time }));
  }

  function hideConnectionState() {
    byId("connection-banner").classList.add("hidden");
  }

  function setDirty(value) {
    dirty = value;
    byId("save-bar").classList.toggle("hidden", !dirty);
  }

  function showToast(message, isError = false) {
    const toast = byId("toast");
    toast.textContent = message;
    toast.classList.toggle("error", isError);
    toast.classList.remove("hidden");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toast.classList.add("hidden"), 3000);
  }

  function locationFromTelegram() {
    return new Promise((resolve, reject) => {
      const manager = telegram && telegram.LocationManager;
      if (!manager || !telegram.isVersionAtLeast("8.0")) {
        reject(new Error("telegram_location_unavailable"));
        return;
      }
      manager.init(() => manager.getLocation((location) => {
        if (location) resolve(location);
        else reject(new Error("location_denied"));
      }));
    });
  }

  function locationFromBrowser() {
    return new Promise((resolve, reject) => {
      if (!navigator.geolocation) {
        reject(new Error("browser_location_unavailable"));
        return;
      }
      navigator.geolocation.getCurrentPosition(
        (position) => resolve(position.coords), reject,
        { enableHighAccuracy: false, timeout: 15000, maximumAge: 60000 },
      );
    });
  }

  async function updateLocation(button) {
    button.disabled = true;
    try {
      let location;
      try {
        location = await locationFromTelegram();
      } catch (_) {
        location = await locationFromBrowser();
      }
      const next = await request("/api/miniapp/location", "PUT", {
        latitude: location.latitude,
        longitude: location.longitude,
      });
      applyState(next);
      setOnlineControlsDisabled(false);
      hideConnectionState();
      void cacheState(next);
      showToast(next.labels.saved);
      if (telegram && telegram.HapticFeedback) telegram.HapticFeedback.notificationOccurred("success");
    } catch (_) {
      showToast(state ? state.labels.location_error : "Location access failed.", true);
      if (telegram && telegram.HapticFeedback) telegram.HapticFeedback.notificationOccurred("error");
    } finally {
      button.disabled = false;
    }
  }

  function collectSettings() {
    const adjustments = {};
    document.querySelectorAll("#adjustment-grid input").forEach((input) => {
      adjustments[input.dataset.prayer] = Number(input.value);
    });
    return {
      language: byId("language").value,
      method: byId("method").value,
      madhab: byId("madhab").value,
      high_latitude_rule: byId("highlat").value,
      hijri_adjustment: Number(byId("hijri-adjustment").value),
      adjustments,
    };
  }

  function collectReminders() {
    return {
      prayer: byId("prayer-reminders").checked,
      pre_prayer_minutes: Number(byId("pre-prayer-minutes").value),
      fasting: byId("fasting-reminders").checked,
      kahf: byId("kahf-reminders").checked,
      occasion_major: byId("occasion-major-reminders").checked,
      occasion_fasting: byId("occasion-fasting-reminders").checked,
      occasion_observed: byId("occasion-observed-reminders").checked,
    };
  }

  function setPreferencesDisabled(value) {
    byId("save-preferences").disabled = value;
    document.querySelectorAll("#dashboard select, #dashboard input").forEach((control) => {
      control.disabled = value;
    });
    if (!value) syncPreReminderAvailability();
  }

  async function savePreferences() {
    if (!dirty) return;
    setPreferencesDisabled(true);
    try {
      const next = await request("/api/miniapp/preferences", "PUT", {
        settings: collectSettings(),
        reminders: collectReminders(),
      });
      applyState(next);
      setOnlineControlsDisabled(false);
      hideConnectionState();
      void cacheState(next);
      showToast(next.labels.saved);
      if (telegram && telegram.HapticFeedback) telegram.HapticFeedback.notificationOccurred("success");
    } catch (_) {
      showToast(state.labels.temporary_failure, true);
    } finally {
      setPreferencesDisabled(false);
    }
  }

  function showCompassUnavailable() {
    const sensor = telegram && telegram.DeviceOrientation;
    if (sensor && sensor.isStarted) sensor.stop();
    compassStarted = false;
    setText("compass-status", state.labels.compass_unavailable);
    byId("compass-status").classList.remove("hidden");
    byId("start-compass").disabled = false;
    setText("start-compass", state.labels.compass_start);
  }

  function updateCompassOrientation() {
    const sensor = telegram && telegram.DeviceOrientation;
    if (!state || !state.qibla || !sensor || !sensor.isStarted) return;
    if (!sensor.absolute || !Number.isFinite(sensor.alpha)) {
      showCompassUnavailable();
      return;
    }
    // Telegram exposes the standard positive Z-axis rotation. Convert it to
    // the clockwise compass heading used by bearings from magnetic north.
    const alpha = ((sensor.alpha * 180 / Math.PI) % 360 + 360) % 360;
    const heading = (360 - alpha) % 360;
    const rotation = Number(state.qibla.bearing_degrees) - heading;
    byId("qibla-needle").style.setProperty("--qibla-rotation", `${rotation}deg`);
    if (!compassStarted) {
      compassStarted = true;
      setText("start-compass", state.labels.compass_active);
      setText("compass-status", state.labels.compass_active);
      byId("compass-status").classList.remove("hidden");
      byId("start-compass").disabled = true;
      if (telegram.HapticFeedback) telegram.HapticFeedback.notificationOccurred("success");
    }
  }

  function startCompass() {
    const sensor = telegram && telegram.DeviceOrientation;
    if (!sensor || !telegram.isVersionAtLeast("8.0")) {
      showCompassUnavailable();
      return;
    }
    byId("start-compass").disabled = true;
    sensor.start({ refresh_rate: 100, need_absolute: true }, (started) => {
      if (!started) showCompassUnavailable();
    });
  }

  function setCalendarButtonsDisabled(value) {
    ["connect-calendar", "copy-calendar-link", "disconnect-calendar"].forEach((id) => {
      byId(id).disabled = value;
    });
  }

  async function ensureCalendarSubscription() {
    const subscription = await request("/api/miniapp/calendar-subscription", "POST");
    state.calendar = subscription;
    calendarFeedURL = new URL(subscription.path, window.location.origin).href;
    renderCalendarSubscription();
    void cacheState(state);
    return calendarFeedURL;
  }

  async function connectGoogleCalendar() {
    setCalendarButtonsDisabled(true);
    setText("calendar-status", state.labels.calendar_opening);
    byId("calendar-status").classList.remove("hidden");
    try {
      const feedURL = await ensureCalendarSubscription();
      const googleURL = `https://calendar.google.com/calendar/render?cid=${encodeURIComponent(feedURL)}`;
      if (telegram && telegram.openLink) {
        telegram.openLink(googleURL);
      } else {
        window.open(googleURL, "_blank", "noopener");
      }
      showToast(state.labels.calendar_opening);
    } catch (_) {
      showToast(state.labels.temporary_failure, true);
    } finally {
      setCalendarButtonsDisabled(false);
    }
  }

  function copyText(value) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(value);
    }
    const input = document.createElement("textarea");
    input.value = value;
    input.setAttribute("readonly", "");
    input.style.position = "fixed";
    input.style.opacity = "0";
    document.body.append(input);
    input.select();
    const copied = document.execCommand("copy");
    input.remove();
    return copied ? Promise.resolve() : Promise.reject(new Error("copy_failed"));
  }

  async function copyCalendarLink() {
    setCalendarButtonsDisabled(true);
    try {
      const feedURL = calendarFeedURL || await ensureCalendarSubscription();
      await copyText(feedURL);
      showToast(state.labels.calendar_copied);
    } catch (_) {
      showToast(state.labels.temporary_failure, true);
    } finally {
      setCalendarButtonsDisabled(false);
    }
  }

  async function disconnectCalendar() {
    setCalendarButtonsDisabled(true);
    try {
      state.calendar = await request("/api/miniapp/calendar-subscription", "DELETE");
      calendarFeedURL = "";
      renderCalendarSubscription();
      void cacheState(state);
      showToast(state.labels.calendar_disconnected);
    } catch (_) {
      showToast(state.labels.temporary_failure, true);
    } finally {
      setCalendarButtonsDisabled(false);
    }
  }

  function roundedRectangle(context, x, y, width, height, radius) {
    const r = Math.min(radius, width / 2, height / 2);
    context.beginPath();
    context.moveTo(x + r, y);
    context.arcTo(x + width, y, x + width, y + height, r);
    context.arcTo(x + width, y + height, x, y + height, r);
    context.arcTo(x, y + height, x, y, r);
    context.arcTo(x, y, x + width, y, r);
    context.closePath();
  }

  function fitCanvasText(context, text, maxWidth, initialSize, weight = 700) {
    let size = initialSize;
    do {
      context.font = `${weight} ${size}px -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif`;
      size -= 2;
    } while (size > 24 && context.measureText(text).width > maxWidth);
  }

  function prayerCardCanvas() {
    const schedule = state[activeDay];
    const canvas = document.createElement("canvas");
    canvas.width = 1080;
    canvas.height = 1350;
    const context = canvas.getContext("2d");
    const rtl = state.locale === "ar";
    const start = rtl ? 940 : 140;
    const end = rtl ? 140 : 940;

    const background = context.createLinearGradient(0, 0, 1080, 1350);
    background.addColorStop(0, "#174d40");
    background.addColorStop(.55, "#0b2d31");
    background.addColorStop(1, "#061b29");
    context.fillStyle = background;
    context.fillRect(0, 0, 1080, 1350);

    const glow = context.createRadialGradient(875, 130, 10, 875, 130, 360);
    glow.addColorStop(0, "rgba(228,190,88,.34)");
    glow.addColorStop(1, "rgba(228,190,88,0)");
    context.fillStyle = glow;
    context.fillRect(500, 0, 580, 560);

    context.strokeStyle = "rgba(240,207,114,.18)";
    context.lineWidth = 2;
    context.beginPath();
    context.arc(930, 170, 235, 0, Math.PI * 2);
    context.stroke();
    context.beginPath();
    context.arc(930, 170, 176, 0, Math.PI * 2);
    context.stroke();

    context.direction = rtl ? "rtl" : "ltr";
    context.textAlign = rtl ? "right" : "left";
    context.textBaseline = "alphabetic";
    context.fillStyle = "#f0cf72";
    context.font = '700 76px Georgia, "Times New Roman", serif';
    context.fillText("☾", start, 135);

    context.fillStyle = "#fffdf2";
    fitCanvasText(context, state.labels.share_card_heading, 780, 52, 800);
    context.fillText(state.labels.share_card_heading, start, 220);

    context.fillStyle = "rgba(255,253,242,.72)";
    fitCanvasText(context, schedule.gregorian, 800, 35, 650);
    context.fillText(schedule.gregorian, start, 278);
    context.fillStyle = "#d9bd6e";
    fitCanvasText(context, schedule.hijri, 800, 31, 650);
    context.fillText(schedule.hijri, start, 326);

    roundedRectangle(context, 90, 365, 900, 760, 38);
    context.fillStyle = "rgba(255,255,255,.095)";
    context.fill();
    context.strokeStyle = "rgba(255,255,255,.12)";
    context.stroke();

    schedule.prayers.forEach((prayer, index) => {
      const y = 455 + index * 108;
      context.direction = rtl ? "rtl" : "ltr";
      context.textAlign = rtl ? "right" : "left";
      context.fillStyle = "#fffdf2";
      fitCanvasText(context, prayer.name, 500, 36, 700);
      context.fillText(prayer.name, start, y);

      context.direction = "ltr";
      context.textAlign = rtl ? "left" : "right";
      context.fillStyle = "#f0cf72";
      context.font = '800 42px -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif';
      context.fillText(prayer.time, end, y);

      if (index < schedule.prayers.length - 1) {
        context.strokeStyle = "rgba(255,255,255,.09)";
        context.beginPath();
        context.moveTo(140, y + 42);
        context.lineTo(940, y + 42);
        context.stroke();
      }
    });

    const method = state.options.methods.find((item) => item.value === state.profile.method);
    const footer = `${schedule.timezone} · ${method ? method.label : state.profile.method}`;
    context.direction = rtl ? "rtl" : "ltr";
    context.textAlign = rtl ? "right" : "left";
    context.fillStyle = "rgba(255,253,242,.62)";
    fitCanvasText(context, footer, 800, 27, 600);
    context.fillText(footer, start, 1196);
    context.fillStyle = "#f0cf72";
    context.font = '650 25px -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, sans-serif';
    context.fillText(state.labels.share_card_footer, start, 1250);

    return canvas;
  }

  function canvasBlob(canvas) {
    const dataURL = canvas.toDataURL("image/png", 1);
    const [header, encoded] = dataURL.split(",");
    const mediaType = (/^data:([^;]+)/.exec(header) || [])[1] || "image/png";
    const binary = window.atob(encoded);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
    return new Blob([bytes], { type: mediaType });
  }

  async function sendPrayerCardToChat(blob, filename) {
    const body = new FormData();
    body.append("card", blob, filename);
    const response = await fetch("/api/miniapp/prayer-card", {
      method: "POST",
      headers: { "X-Telegram-Init-Data": currentInitData() },
      body,
      credentials: "same-origin",
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(data.error || "temporary_failure");
      error.code = data.error;
      error.status = response.status;
      throw error;
    }
  }

  async function sharePrayerCard() {
    const button = byId("share-prayer-card");
    button.disabled = true;
    setText("share-prayer-card", state.labels.share_preparing);
    try {
      // Keep generation synchronous until navigator.share is called so the
      // browser preserves the button click's required transient activation.
      const blob = canvasBlob(prayerCardCanvas());
      const filename = `prayer-times-${activeDay}.png`;
      const file = typeof File === "function" ? new File([blob], filename, { type: "image/png" }) : null;
      if (file && typeof navigator.share === "function") {
        try {
          // Some Telegram WebViews implement file sharing without exposing a
          // reliable navigator.canShare result, so attempt the native sheet.
          await navigator.share({
            title: state.labels.share_card_heading,
            text: `${state.labels.share_message}\n${state[activeDay].gregorian}`,
            files: [file],
          });
          return;
        } catch (error) {
          if (error && error.name === "AbortError") return;
          // Fall through to a Telegram chat upload when the WebView rejects
          // file sharing. An <a download> click is silently ignored on many
          // Android Telegram clients.
        }
      }
      await sendPrayerCardToChat(blob, filename);
      showToast(state.labels.share_sent);
      if (telegram && telegram.HapticFeedback) telegram.HapticFeedback.notificationOccurred("success");
    } catch (error) {
      if (!error || error.name !== "AbortError") showToast(state.labels.share_failed, true);
    } finally {
      button.disabled = false;
      setText("share-prayer-card", state.labels.share_action);
    }
  }

  function showLaunchError(kind) {
    const copy = launchCopy[launchLanguage()] || launchCopy.en;
    loading.classList.add("hidden");
    locationGate.classList.add("hidden");
    dashboard.classList.add("hidden");
    setText("standalone-text", copy[kind]);
    setText("retry-app", copy.retry);
    byId("retry-app").classList.toggle("hidden", kind === "open");
    standalone.classList.remove("hidden");
  }

  async function bootstrapApp() {
    if (!currentInitData()) {
      showLaunchError("open");
      return;
    }
    standalone.classList.add("hidden");
    if (!zakatCurrency) zakatCurrency = await readStoredValue(zakatCurrencyKey());
    const cached = await cachedState();
    if (cached) {
      applyState(cached.state);
      setOnlineControlsDisabled(true);
      showConnectionState("refreshing", cached.saved_at);
    } else {
      loading.classList.remove("hidden");
    }
    try {
      const fresh = await request("/api/miniapp/bootstrap");
      applyState(fresh);
      setOnlineControlsDisabled(false);
      hideConnectionState();
      await cacheState(fresh);
    } catch (error) {
      if (error.status === 401 || !cached) {
        showLaunchError(error.status === 401 ? "expired" : "failed");
        return;
      }
      setOnlineControlsDisabled(true);
      showConnectionState("offline", cached.saved_at);
    }
  }

  function selectView(view) {
    const known = ["prayer", "dates", "zakat", "places", "tools", "settings"];
    if (!known.includes(view)) view = "prayer";
    activeView = view;
    known.forEach((name) => {
      const panel = byId(`view-${name}`);
      if (panel) panel.classList.toggle("hidden", name !== view);
    });
    document.querySelectorAll(".tab").forEach((tab) => {
      const active = tab.dataset.view === view;
      tab.classList.toggle("active", active);
      tab.setAttribute("aria-current", active ? "page" : "false");
    });
    window.scrollTo({ top: 0, behavior: "auto" });
    if (view === "places") ensurePlacesMap();
  }

  // --- "Prayer times anywhere" map lookup (dependency-free OSM slippy map) ---

  function lngToTileX(lng, z) { return (lng + 180) / 360 * Math.pow(2, z); }
  function latToTileY(lat, z) {
    const r = lat * Math.PI / 180;
    return (1 - Math.log(Math.tan(r) + 1 / Math.cos(r)) / Math.PI) / 2 * Math.pow(2, z);
  }
  function tileXToLng(x, z) { return x / Math.pow(2, z) * 360 - 180; }
  function tileYToLat(y, z) {
    const n = Math.PI - 2 * Math.PI * y / Math.pow(2, z);
    return 180 / Math.PI * Math.atan(0.5 * (Math.exp(n) - Math.exp(-n)));
  }
  function clampLat(lat) { return Math.max(-85.05, Math.min(85.05, lat)); }
  function wrapLng(lng) { return ((lng + 180) % 360 + 360) % 360 - 180; }

  function renderPlacesTiles() {
    const map = byId("places-map");
    const layer = byId("places-tiles");
    const w = map.clientWidth, h = map.clientHeight;
    if (!w || !h) return;
    layer.style.transform = "";
    layer.replaceChildren();
    const z = places.z;
    const world = Math.pow(2, z);
    const originX = lngToTileX(places.lng, z) * 256 - w / 2;
    const originY = latToTileY(places.lat, z) * 256 - h / 2;
    const firstCol = Math.floor(originX / 256), lastCol = Math.floor((originX + w) / 256);
    const firstRow = Math.floor(originY / 256), lastRow = Math.floor((originY + h) / 256);
    for (let ty = firstRow; ty <= lastRow; ty += 1) {
      if (ty < 0 || ty >= world) continue;
      for (let tx = firstCol; tx <= lastCol; tx += 1) {
        const wx = ((tx % world) + world) % world;
        const img = document.createElement("img");
        img.src = `https://tile.openstreetmap.org/${z}/${wx}/${ty}.png`;
        img.alt = "";
        img.style.left = `${tx * 256 - originX}px`;
        img.style.top = `${ty * 256 - originY}px`;
        layer.append(img);
      }
    }
  }

  function finishPan(dx, dy) {
    const z = places.z;
    places.lng = wrapLng(tileXToLng((lngToTileX(places.lng, z) * 256 - dx) / 256, z));
    places.lat = clampLat(tileYToLat((latToTileY(places.lat, z) * 256 - dy) / 256, z));
    renderPlacesTiles();
    schedulePlacesLookup();
  }

  function setPlacesZoom(delta) {
    places.z = Math.max(2, Math.min(18, places.z + delta));
    renderPlacesTiles();
    schedulePlacesLookup();
  }

  function ensurePlacesMap() {
    places.initialized = true;
    renderPlacesTiles();
    if (!places.lastKey) schedulePlacesLookup();
  }

  function schedulePlacesLookup() {
    clearTimeout(placesLookupTimer);
    setText("places-location", "…");
    placesLookupTimer = setTimeout(runPlacesLookup, 450);
  }

  async function runPlacesLookup() {
    const lat = Number(places.lat.toFixed(4));
    const lng = Number(places.lng.toFixed(4));
    const key = `${lat},${lng},${placesDay}`;
    if (key === places.lastKey) return;
    places.lastKey = key;
    try {
      const data = await request("/api/miniapp/lookup", "POST", { latitude: lat, longitude: lng, day: placesDay });
      setText("places-location", data.location_name || "");
      renderPlacesSchedule(data.schedule);
    } catch (_) {
      places.lastKey = "";
      setText("places-location", state ? state.labels.temporary_failure : "");
    }
  }

  function renderPlacesSchedule(schedule) {
    if (!schedule) return;
    setText("places-gregorian", schedule.gregorian);
    setText("places-hijri", `☾ ${schedule.hijri}`);
    setText("places-timezone", schedule.timezone);
    const grid = byId("places-grid");
    grid.replaceChildren();
    schedule.prayers.forEach((prayer) => {
      const item = document.createElement("div");
      item.className = "prayer";
      const emoji = document.createElement("span");
      emoji.className = "prayer-emoji";
      emoji.textContent = prayer.emoji;
      const name = document.createElement("span");
      name.className = "prayer-name";
      name.textContent = prayer.name;
      const time = document.createElement("strong");
      time.className = "prayer-time";
      time.textContent = prayer.time;
      item.append(emoji, name, time);
      grid.append(item);
    });
  }

  function selectPlacesDay(day) {
    placesDay = day;
    ["today", "tomorrow"].forEach((name) => {
      const tab = byId(`places-${name}-tab`);
      const active = name === day;
      tab.classList.toggle("active", active);
      tab.setAttribute("aria-selected", String(active));
    });
    places.lastKey = "";
    runPlacesLookup();
  }

  function bindPlacesMap() {
    const map = byId("places-map");
    if (!map) return;
    let dragging = false, startX = 0, startY = 0, dx = 0, dy = 0;
    map.addEventListener("pointerdown", (event) => {
      dragging = true; startX = event.clientX; startY = event.clientY; dx = 0; dy = 0;
      try { map.setPointerCapture(event.pointerId); } catch (_) { /* capture is best-effort */ }
    });
    map.addEventListener("pointermove", (event) => {
      if (!dragging) return;
      dx = event.clientX - startX; dy = event.clientY - startY;
      byId("places-tiles").style.transform = `translate(${dx}px, ${dy}px)`;
    });
    const end = (event) => {
      if (!dragging) return;
      dragging = false;
      try { map.releasePointerCapture(event.pointerId); } catch (_) { /* best-effort */ }
      if (dx || dy) finishPan(dx, dy); else renderPlacesTiles();
    };
    map.addEventListener("pointerup", end);
    map.addEventListener("pointercancel", () => { dragging = false; renderPlacesTiles(); });
    byId("places-zoom-in").addEventListener("click", () => setPlacesZoom(1));
    byId("places-zoom-out").addEventListener("click", () => setPlacesZoom(-1));
    byId("places-today-tab").addEventListener("click", () => selectPlacesDay("today"));
    byId("places-tomorrow-tab").addEventListener("click", () => selectPlacesDay("tomorrow"));
  }

  function selectDay(day) {
    activeDay = day;
    ["today", "tomorrow"].forEach((name) => {
      const tab = byId(`${name}-tab`);
      const active = name === day;
      tab.classList.toggle("active", active);
      tab.setAttribute("aria-selected", String(active));
    });
    renderSchedule();
  }

  byId("today-tab").addEventListener("click", () => selectDay("today"));
  byId("tomorrow-tab").addEventListener("click", () => selectDay("tomorrow"));
  document.querySelectorAll(".tab").forEach((tab) => {
    tab.addEventListener("click", () => selectView(tab.dataset.view));
  });
  bindPlacesMap();
  byId("location-primary").addEventListener("click", (event) => updateLocation(event.currentTarget));
  byId("location-secondary").addEventListener("click", (event) => updateLocation(event.currentTarget));
  byId("start-compass").addEventListener("click", startCompass);
  byId("connect-calendar").addEventListener("click", connectGoogleCalendar);
  byId("copy-calendar-link").addEventListener("click", copyCalendarLink);
  byId("disconnect-calendar").addEventListener("click", disconnectCalendar);
  byId("add-home-screen").addEventListener("click", addToHomeScreen);
  byId("share-prayer-card").addEventListener("click", sharePrayerCard);
  byId("save-preferences").addEventListener("click", savePreferences);
  byId("retry-app").addEventListener("click", bootstrapApp);
  ["prayer-reminders", "pre-prayer-minutes", "fasting-reminders", "kahf-reminders",
    "occasion-major-reminders", "occasion-fasting-reminders", "occasion-observed-reminders",
    "language", "method", "madhab", "highlat", "hijri-adjustment"]
    .forEach((id) => byId(id).addEventListener("change", () => setDirty(true)));
  byId("prayer-reminders").addEventListener("change", syncPreReminderAvailability);
  byId("adjustment-grid").addEventListener("input", () => setDirty(true));
  byId("zakat-currency").addEventListener("change", () => {
    zakatCurrency = byId("zakat-currency").value;
    void writeStoredValue(zakatCurrencyKey(), zakatCurrency);
    renderZakatResult();
  });
  byId("zakat-holdings").addEventListener("input", renderZakatResult);
  window.addEventListener("pageshow", (event) => {
    if (event.persisted) bootstrapApp();
  });
  window.addEventListener("online", () => {
    if (offlineMode) bootstrapApp();
  });
  window.addEventListener("pagehide", () => {
    const sensor = telegram && telegram.DeviceOrientation;
    if (sensor && sensor.isStarted) sensor.stop();
  });
  if (telegram && telegram.onEvent) {
    telegram.onEvent("deviceOrientationChanged", updateCompassOrientation);
    telegram.onEvent("deviceOrientationFailed", showCompassUnavailable);
    telegram.onEvent("homeScreenAdded", () => updateHomeScreenStatus("added"));
    telegram.onEvent("homeScreenChecked", updateHomeScreenStatus);
  }

  bootstrapApp();
})();
