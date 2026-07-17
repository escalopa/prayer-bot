(() => {
  "use strict";

  const telegram = window.Telegram && window.Telegram.WebApp;
  let initData = "";
  let state = null;
  let activeDay = "today";
  let toastTimer = null;
  let dirty = false;

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
    setText("language-label", labels.language);
    setText("method-label", labels.method);
    setText("madhab-label", labels.madhab);
    setText("highlat-label", labels.highlat);
    setText("hijri-label", labels.hijri);
    setText("adjustments-label", labels.adjustments);
    setText("save-preferences", labels.save);
    setText("calculation-note", labels.calculated_locally);
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
  }

  function renderReminders() {
    byId("prayer-reminders").checked = state.reminders.prayer;
    fillSelect("pre-prayer-minutes", state.options.pre_reminders, state.reminders.pre_prayer_minutes);
    byId("fasting-reminders").checked = state.reminders.fasting;
    byId("kahf-reminders").checked = state.reminders.kahf;
    syncPreReminderAvailability();
  }

  function syncPreReminderAvailability() {
    byId("pre-prayer-minutes").disabled = !byId("prayer-reminders").checked;
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
    renderReminders();
    renderSettings();
    setDirty(false);
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
      showToast(next.labels.saved);
      if (telegram && telegram.HapticFeedback) telegram.HapticFeedback.notificationOccurred("success");
    } catch (_) {
      showToast(state.labels.temporary_failure, true);
    } finally {
      setPreferencesDisabled(false);
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
    loading.classList.remove("hidden");
    try {
      applyState(await request("/api/miniapp/bootstrap"));
    } catch (error) {
      showLaunchError(error.status === 401 ? "expired" : "failed");
    }
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
  byId("location-primary").addEventListener("click", (event) => updateLocation(event.currentTarget));
  byId("location-secondary").addEventListener("click", (event) => updateLocation(event.currentTarget));
  byId("save-preferences").addEventListener("click", savePreferences);
  byId("retry-app").addEventListener("click", bootstrapApp);
  ["prayer-reminders", "pre-prayer-minutes", "fasting-reminders", "kahf-reminders", "language", "method", "madhab", "highlat", "hijri-adjustment"]
    .forEach((id) => byId(id).addEventListener("change", () => setDirty(true)));
  byId("prayer-reminders").addEventListener("change", syncPreReminderAvailability);
  byId("adjustment-grid").addEventListener("input", () => setDirty(true));
  window.addEventListener("pageshow", (event) => {
    if (event.persisted) bootstrapApp();
  });

  bootstrapApp();
})();
