package i18n

type localeExtension struct {
	buttons     map[string]string
	text        map[string]string
	hijriMonths []string
}

func init() {
	extensions := map[string]localeExtension{
		"en": {
			buttons: map[string]string{
				"hijri": "🌙 Hijri date correction", "prayer_reminders": "Prayer times",
				"fasting_reminders": "Monday & Thursday fasting", "kahf_reminders": "Friday Al-Kahf",
			},
			text: map[string]string{
				"reminders_title": "<b>Reminders</b> 🔔", "enabled": "enabled", "disabled": "disabled",
				"pre_prayer_reminder": "Pre-prayer reminder", "pre_reminder_off": "At prayer time only", "minutes_before": "%d minutes before",
				"choose_pre_reminder": "<b>Choose a pre-prayer reminder</b> ⏳\n\nYou will receive one message before every obligatory prayer, followed by the prayer-time notification.",
				"fasting_schedule":    "Evening before · 20:00", "kahf_schedule": "Friday · 09:00",
				"hijri_date": "Hijri date", "hijri_era": "AH", "hijri_setting": "Umm al-Qura correction: %+d day(s)",
				"hijri_note": "Umm al-Qura · calculated", "choose_hijri": "<b>Correct the calculated Hijri date</b> 🌙\n\nLocal moon-sighting calendars can differ. Choose a correction from −2 to +2 days; the current value is marked ✓.",
				"reminder_fasting": "<b>Fasting reminder</b> 🌙\nTomorrow is Monday or Thursday, a day for voluntary fasting. May Allah accept it from you.",
				"reminder_kahf":    "<b>Friday reminder</b> 📖\nMake time to read Surah Al-Kahf today.",
			},
			hijriMonths: []string{"Muharram", "Safar", "Rabi' al-Awwal", "Rabi' al-Thani", "Jumada al-Awwal", "Jumada al-Thani", "Rajab", "Sha'ban", "Ramadan", "Shawwal", "Dhu al-Qi'dah", "Dhu al-Hijjah"},
		},
		"ar": {
			buttons: map[string]string{"hijri": "🌙 تصحيح التاريخ الهجري", "prayer_reminders": "مواقيت الصلاة", "fasting_reminders": "صيام الاثنين والخميس", "kahf_reminders": "سورة الكهف يوم الجمعة"},
			text: map[string]string{
				"reminders_title": "<b>التنبيهات</b> 🔔", "enabled": "مفعّل", "disabled": "متوقف", "fasting_schedule": "مساء اليوم السابق · 20:00", "kahf_schedule": "الجمعة · 09:00",
				"pre_prayer_reminder": "تنبيه قبل الصلاة", "pre_reminder_off": "عند دخول وقت الصلاة فقط", "minutes_before": "قبل الصلاة بـ %d دقيقة",
				"choose_pre_reminder": "<b>اختر موعد التنبيه قبل الصلاة</b> ⏳\n\nسيصلك تنبيه قبل كل صلاة مفروضة، ثم تنبيه عند دخول وقت الصلاة.",
				"hijri_date":          "التاريخ الهجري", "hijri_era": "هـ", "hijri_setting": "تصحيح أم القرى: %+d يوم", "hijri_note": "أم القرى · محسوب", "choose_hijri": "<b>تصحيح التاريخ الهجري المحسوب</b> 🌙\n\nقد يختلف ثبوت الهلال محليًا. اختر تصحيحًا من −2 إلى +2 يوم، والقيمة الحالية مميزة بعلامة ✓.",
				"reminder_fasting": "<b>تذكير بالصيام</b> 🌙\nغدًا الاثنين أو الخميس، وهو يوم من أيام صيام التطوع. تقبل الله منك.", "reminder_kahf": "<b>تذكير الجمعة</b> 📖\nلا تنسَ قراءة سورة الكهف اليوم.",
			},
			hijriMonths: []string{"محرم", "صفر", "ربيع الأول", "ربيع الآخر", "جمادى الأولى", "جمادى الآخرة", "رجب", "شعبان", "رمضان", "شوال", "ذو القعدة", "ذو الحجة"},
		},
		"es": {
			buttons: map[string]string{"hijri": "🌙 Corrección de fecha hiyri", "prayer_reminders": "Horarios de oración", "fasting_reminders": "Ayuno lunes y jueves", "kahf_reminders": "Al-Kahf del viernes"},
			text: map[string]string{
				"reminders_title": "<b>Recordatorios</b> 🔔", "enabled": "activado", "disabled": "desactivado", "fasting_schedule": "Víspera · 20:00", "kahf_schedule": "Viernes · 09:00",
				"pre_prayer_reminder": "Aviso antes de la oración", "pre_reminder_off": "Solo al comenzar la oración", "minutes_before": "%d minutos antes",
				"choose_pre_reminder": "<b>Elige el aviso previo</b> ⏳\n\nRecibirás un mensaje antes de cada oración obligatoria y otro cuando llegue su hora.",
				"hijri_date":          "Fecha hiyri", "hijri_era": "AH", "hijri_setting": "Corrección Umm al-Qura: %+d día(s)", "hijri_note": "Umm al-Qura · calculada", "choose_hijri": "<b>Corrige la fecha hiyri calculada</b> 🌙\n\nLa observación lunar local puede variar. Elige una corrección de −2 a +2 días; el valor actual está marcado ✓.",
				"reminder_fasting": "<b>Recordatorio de ayuno</b> 🌙\nMañana es lunes o jueves, un día de ayuno voluntario. Que Allah lo acepte.", "reminder_kahf": "<b>Recordatorio del viernes</b> 📖\nReserva tiempo para leer la sura Al-Kahf hoy.",
			},
			hijriMonths: []string{"Muharram", "Safar", "Rabi al-Awwal", "Rabi al-Thani", "Yumada al-Awwal", "Yumada al-Thani", "Rayab", "Sha'ban", "Ramadán", "Shawwal", "Dhu al-Qi'dah", "Dhu al-Hiyyah"},
		},
		"fr": {
			buttons: map[string]string{"hijri": "🌙 Correction de date hégirienne", "prayer_reminders": "Horaires de prière", "fasting_reminders": "Jeûne lundi et jeudi", "kahf_reminders": "Al-Kahf du vendredi"},
			text: map[string]string{
				"reminders_title": "<b>Rappels</b> 🔔", "enabled": "activé", "disabled": "désactivé", "fasting_schedule": "La veille · 20:00", "kahf_schedule": "Vendredi · 09:00",
				"pre_prayer_reminder": "Rappel avant la prière", "pre_reminder_off": "À l’heure de la prière uniquement", "minutes_before": "%d minutes avant",
				"choose_pre_reminder": "<b>Choisissez le rappel préalable</b> ⏳\n\nVous recevrez un message avant chaque prière obligatoire, puis un autre à l’heure de la prière.",
				"hijri_date":          "Date hégirienne", "hijri_era": "AH", "hijri_setting": "Correction Umm al-Qura : %+d jour(s)", "hijri_note": "Umm al-Qura · calculée", "choose_hijri": "<b>Corrigez la date hégirienne calculée</b> 🌙\n\nL'observation locale de la lune peut différer. Choisissez de −2 à +2 jours ; la valeur actuelle est marquée ✓.",
				"reminder_fasting": "<b>Rappel de jeûne</b> 🌙\nDemain est lundi ou jeudi, un jour de jeûne volontaire. Qu'Allah l'accepte.", "reminder_kahf": "<b>Rappel du vendredi</b> 📖\nPrenez le temps de lire la sourate Al-Kahf aujourd'hui.",
			},
			hijriMonths: []string{"Mouharram", "Safar", "Rabi al-Awwal", "Rabi al-Thani", "Joumada al-Awwal", "Joumada al-Thani", "Rajab", "Chaabane", "Ramadan", "Chawwal", "Dhou al-Qi'dah", "Dhou al-Hijjah"},
		},
		"ru": {
			buttons: map[string]string{"hijri": "🌙 Поправка даты Хиджры", "prayer_reminders": "Время намаза", "fasting_reminders": "Пост в понедельник и четверг", "kahf_reminders": "Аль-Кахф в пятницу"},
			text: map[string]string{
				"reminders_title": "<b>Напоминания</b> 🔔", "enabled": "включено", "disabled": "выключено", "fasting_schedule": "Накануне · 20:00", "kahf_schedule": "Пятница · 09:00",
				"pre_prayer_reminder": "Напоминание перед намазом", "pre_reminder_off": "Только при наступлении намаза", "minutes_before": "За %d мин.",
				"choose_pre_reminder": "<b>Выберите предварительное напоминание</b> ⏳\n\nВы получите сообщение перед каждым обязательным намазом, а затем — при наступлении его времени.",
				"hijri_date":          "Дата Хиджры", "hijri_era": "г. х.", "hijri_setting": "Поправка Умм аль-Кура: %+d дн.", "hijri_note": "Умм аль-Кура · расчёт", "choose_hijri": "<b>Поправка расчётной даты Хиджры</b> 🌙\n\nМестное наблюдение луны может отличаться. Выберите от −2 до +2 дней; текущее значение отмечено ✓.",
				"reminder_fasting": "<b>Напоминание о посте</b> 🌙\nЗавтра понедельник или четверг — день добровольного поста. Пусть Аллах примет его.", "reminder_kahf": "<b>Пятничное напоминание</b> 📖\nНайдите время прочитать суру «Аль-Кахф» сегодня.",
			},
			hijriMonths: []string{"Мухаррам", "Сафар", "Раби аль-авваль", "Раби ас-сани", "Джумада аль-уля", "Джумада ас-сания", "Раджаб", "Шаабан", "Рамадан", "Шавваль", "Зуль-када", "Зуль-хиджа"},
		},
		"tr": {
			buttons: map[string]string{"hijri": "🌙 Hicri tarih düzeltmesi", "prayer_reminders": "Namaz vakitleri", "fasting_reminders": "Pazartesi ve Perşembe orucu", "kahf_reminders": "Cuma Kehf Suresi"},
			text: map[string]string{
				"reminders_title": "<b>Hatırlatıcılar</b> 🔔", "enabled": "açık", "disabled": "kapalı", "fasting_schedule": "Önceki akşam · 20:00", "kahf_schedule": "Cuma · 09:00",
				"pre_prayer_reminder": "Namaz öncesi hatırlatma", "pre_reminder_off": "Yalnızca namaz vaktinde", "minutes_before": "%d dakika önce",
				"choose_pre_reminder": "<b>Namaz öncesi hatırlatmayı seçin</b> ⏳\n\nHer farz namazdan önce bir mesaj, vakit geldiğinde de ikinci bir bildirim alırsınız.",
				"hijri_date":          "Hicri tarih", "hijri_era": "H", "hijri_setting": "Ümmü'l-Kurâ düzeltmesi: %+d gün", "hijri_note": "Ümmü'l-Kurâ · hesaplandı", "choose_hijri": "<b>Hesaplanan Hicri tarihi düzeltin</b> 🌙\n\nYerel hilal gözlemi farklı olabilir. −2 ile +2 gün arasında seçim yapın; geçerli değer ✓ ile işaretlidir.",
				"reminder_fasting": "<b>Oruç hatırlatıcısı</b> 🌙\nYarın Pazartesi veya Perşembe, nafile oruç günüdür. Allah kabul etsin.", "reminder_kahf": "<b>Cuma hatırlatıcısı</b> 📖\nBugün Kehf Suresi'ni okumaya vakit ayırın.",
			},
			hijriMonths: []string{"Muharrem", "Safer", "Rebiülevvel", "Rebiülahir", "Cemaziyelevvel", "Cemaziyelahir", "Recep", "Şaban", "Ramazan", "Şevval", "Zilkade", "Zilhicce"},
		},
		"uz": {
			buttons: map[string]string{"hijri": "🌙 Hijriy sana tuzatishi", "prayer_reminders": "Namoz vaqtlari", "fasting_reminders": "Dushanba va payshanba ro‘zasi", "kahf_reminders": "Juma kuni Kahf surasi"},
			text: map[string]string{
				"reminders_title": "<b>Eslatmalar</b> 🔔", "enabled": "yoqilgan", "disabled": "o‘chirilgan", "fasting_schedule": "Oldingi oqshom · 20:00", "kahf_schedule": "Juma · 09:00",
				"pre_prayer_reminder": "Namozdan oldingi eslatma", "pre_reminder_off": "Faqat namoz vaqtida", "minutes_before": "%d daqiqa oldin",
				"choose_pre_reminder": "<b>Namozdan oldingi eslatmani tanlang</b> ⏳\n\nHar bir farz namozidan oldin va namoz vaqti kirganda alohida xabar olasiz.",
				"hijri_date":          "Hijriy sana", "hijri_era": "h.", "hijri_setting": "Umm al-Qura tuzatishi: %+d kun", "hijri_note": "Umm al-Qura · hisoblangan", "choose_hijri": "<b>Hisoblangan hijriy sanani tuzating</b> 🌙\n\nMahalliy hilol kuzatuvi farq qilishi mumkin. −2 dan +2 kungacha tanlang; joriy qiymat ✓ bilan belgilangan.",
				"reminder_fasting": "<b>Ro‘za eslatmasi</b> 🌙\nErtaga dushanba yoki payshanba — nafl ro‘za kuni. Alloh qabul qilsin.", "reminder_kahf": "<b>Juma eslatmasi</b> 📖\nBugun Kahf surasini o‘qishga vaqt ajrating.",
			},
			hijriMonths: []string{"Muharram", "Safar", "Rabi’ ul-avval", "Rabi’ us-soniy", "Jumodul avval", "Jumodus soniy", "Rajab", "Sha’bon", "Ramazon", "Shavvol", "Zulqa’da", "Zulhijja"},
		},
		"tt": {
			buttons: map[string]string{"hijri": "🌙 Һиҗри дата төзәтмәсе", "prayer_reminders": "Намаз вакытлары", "fasting_reminders": "Дүшәмбе һәм пәнҗешәмбе уразасы", "kahf_reminders": "Җомга Кәһф сүрәсе"},
			text: map[string]string{
				"reminders_title": "<b>Искәртүләр</b> 🔔", "enabled": "кабызылган", "disabled": "сүндерелгән", "fasting_schedule": "Алдагы кич · 20:00", "kahf_schedule": "Җомга · 09:00",
				"pre_prayer_reminder": "Намаз алдыннан искәртү", "pre_reminder_off": "Намаз вакыты җиткәч кенә", "minutes_before": "%d минут алдан",
				"choose_pre_reminder": "<b>Намаз алдыннан искәртүне сайлагыз</b> ⏳\n\nҺәр фарыз намаз алдыннан һәм намаз вакыты җиткәч аерым хәбәр алырсыз.",
				"hijri_date":          "Һиҗри дата", "hijri_era": "һ.", "hijri_setting": "Умм әл-Кура төзәтмәсе: %+d көн", "hijri_note": "Умм әл-Кура · исәпләнгән", "choose_hijri": "<b>Исәпләнгән Һиҗри датаны төзәтегез</b> 🌙\n\nҖирле ай күренеше аерылырга мөмкин. −2 дән +2 көнгә кадәр сайлагыз; хәзерге кыйммәт ✓ белән билгеләнгән.",
				"reminder_fasting": "<b>Ураза искәртүе</b> 🌙\nИртәгә дүшәмбе яки пәнҗешәмбе — нәфел ураза көне. Аллаһ кабул итсен.", "reminder_kahf": "<b>Җомга искәртүе</b> 📖\nБүген Кәһф сүрәсен укырга вакыт табыгыз.",
			},
			hijriMonths: []string{"Мөхәррәм", "Сәфәр", "Рабигыль-әүвәл", "Рабигыль-ахыр", "Җөмадиәл-әүвәл", "Җөмадиәл-ахыр", "Рәҗәб", "Шәгъбан", "Рамазан", "Шәүвәл", "Зөлкагдә", "Зөлхиҗҗә"},
		},
	}
	for code, extension := range extensions {
		locale := locales[code]
		for key, value := range extension.buttons {
			locale.Buttons[key] = value
		}
		for key, value := range extension.text {
			locale.Text[key] = value
		}
		locale.HijriMonths = extension.hijriMonths
		locales[code] = locale
	}
}
