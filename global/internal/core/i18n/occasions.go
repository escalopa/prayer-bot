package i18n

type OccasionCopy struct {
	Title   string
	Summary string
	Action  string
}

func (l Locale) Occasion(id string) OccasionCopy {
	if value, ok := occasionCopy[l.Code][id]; ok {
		return value
	}
	return occasionCopy["en"][id]
}

func (l Locale) OccasionCategory(category string) string {
	if value := occasionCategories[l.Code][category]; value != "" {
		return value
	}
	return occasionCategories["en"][category]
}

func (l Locale) OccasionUI(key string) string {
	if value := occasionUI[l.Code][key]; value != "" {
		return value
	}
	return occasionUI["en"][key]
}

var occasionUI = map[string]map[string]string{
	"en": {
		"title": "Upcoming Islamic dates", "help": "Calculated from your corrected Hijri calendar.",
		"disclaimer":  "Local moon sighting and scholarly practice may differ.",
		"recommended": "Recommended", "sources": "Sources",
		"major_reminders": "Major Islamic occasions", "fasting_reminders": "Special fasting days",
		"observed_reminders": "Commonly observed dates", "schedule": "Evening before · 20:00",
	},
	"ar": {
		"title": "المناسبات الإسلامية القادمة", "help": "محسوبة وفق تقويمك الهجري المصحح.",
		"disclaimer":  "قد يختلف ثبوت الهلال والعمل الفقهي محليًا.",
		"recommended": "المقترح", "sources": "المصادر",
		"major_reminders": "المناسبات الإسلامية الكبرى", "fasting_reminders": "أيام الصيام الخاصة",
		"observed_reminders": "المناسبات الشائعة", "schedule": "مساء اليوم السابق · 20:00",
	},
	"es": {
		"title": "Próximas fechas islámicas", "help": "Calculadas con tu calendario hiyri corregido.",
		"disclaimer":  "El avistamiento lunar y la práctica académica local pueden variar.",
		"recommended": "Recomendado", "sources": "Fuentes",
		"major_reminders": "Ocasiones islámicas principales", "fasting_reminders": "Días especiales de ayuno",
		"observed_reminders": "Fechas habitualmente observadas", "schedule": "Víspera · 20:00",
	},
	"fr": {
		"title": "Prochaines dates islamiques", "help": "Calculées selon votre calendrier hégirien corrigé.",
		"disclaimer":  "L’observation lunaire et les pratiques savantes locales peuvent varier.",
		"recommended": "Recommandé", "sources": "Sources",
		"major_reminders": "Grandes occasions islamiques", "fasting_reminders": "Jours de jeûne particuliers",
		"observed_reminders": "Dates couramment observées", "schedule": "La veille · 20:00",
	},
	"ru": {
		"title": "Ближайшие исламские даты", "help": "Рассчитаны по вашему скорректированному календарю Хиджры.",
		"disclaimer":  "Местное наблюдение луны и мнения учёных могут различаться.",
		"recommended": "Рекомендуется", "sources": "Источники",
		"major_reminders": "Важные исламские даты", "fasting_reminders": "Особые дни поста",
		"observed_reminders": "Распространённые даты", "schedule": "Накануне · 20:00",
	},
	"tr": {
		"title": "Yaklaşan İslami tarihler", "help": "Düzeltilmiş Hicri takviminize göre hesaplanır.",
		"disclaimer":  "Yerel hilal gözlemi ve ilmî uygulamalar farklı olabilir.",
		"recommended": "Önerilen", "sources": "Kaynaklar",
		"major_reminders": "Önemli İslami günler", "fasting_reminders": "Özel oruç günleri",
		"observed_reminders": "Yaygın anma tarihleri", "schedule": "Önceki akşam · 20:00",
	},
	"uz": {
		"title": "Yaqin Islomiy sanalar", "help": "Tuzatilgan Hijriy taqvimingiz bo‘yicha hisoblanadi.",
		"disclaimer":  "Mahalliy hilol kuzatuvi va ulamolar amaliyoti farq qilishi mumkin.",
		"recommended": "Tavsiya", "sources": "Manbalar",
		"major_reminders": "Muhim Islomiy sanalar", "fasting_reminders": "Maxsus ro‘za kunlari",
		"observed_reminders": "Keng nishonlanadigan sanalar", "schedule": "Oldingi oqshom · 20:00",
	},
	"tt": {
		"title": "Якын Ислам даталары", "help": "Төзәтелгән Һиҗри календарегыз буенча исәпләнә.",
		"disclaimer":  "Җирле ай күренеше һәм галимнәр практикасы аерылырга мөмкин.",
		"recommended": "Киңәш", "sources": "Чыганаклар",
		"major_reminders": "Мөһим Ислам көннәре", "fasting_reminders": "Махсус ураза көннәре",
		"observed_reminders": "Киң билгеләп үтелгән даталар", "schedule": "Алдагы кич · 20:00",
	},
}

var occasionCategories = map[string]map[string]string{
	"en": {"major": "Major occasion", "fasting": "Fasting opportunity", "observed": "Commonly observed"},
	"ar": {"major": "مناسبة كبرى", "fasting": "فرصة للصيام", "observed": "مناسبة شائعة"},
	"es": {"major": "Ocasión principal", "fasting": "Oportunidad de ayuno", "observed": "Conmemoración habitual"},
	"fr": {"major": "Grande occasion", "fasting": "Jeûne recommandé", "observed": "Date couramment observée"},
	"ru": {"major": "Важная дата", "fasting": "Возможность поста", "observed": "Распространённая дата"},
	"tr": {"major": "Önemli gün", "fasting": "Oruç fırsatı", "observed": "Yaygın anma"},
	"uz": {"major": "Muhim sana", "fasting": "Ro‘za imkoniyati", "observed": "Keng nishonlanadigan sana"},
	"tt": {"major": "Мөһим көн", "fasting": "Ураза мөмкинлеге", "observed": "Киң билгеләп үтелә"},
}

var occasionCopy = map[string]map[string]OccasionCopy{
	"en": {
		"ashura":            {Title: "Day of Ashura", Summary: "The tenth of Muharram is a day of gratitude and remembrance.", Action: "Consider fasting Ashura and an adjacent day."},
		"mawlid":            {Title: "Mawlid al-Nabi", Summary: "Commonly observed as the Prophet’s birth date; the exact historical date and observance differ among Muslims.", Action: "Send blessings and peace upon the Prophet ﷺ and study his character."},
		"isra_miraj":        {Title: "Isra and Mi’raj", Summary: "A commonly observed date recalling the Night Journey and Ascension; its precise calendar date is not established.", Action: "Read the opening of Surah Al-Isra and reflect on the gift of prayer."},
		"mid_shaban":        {Title: "Mid-Sha’ban", Summary: "A night commonly observed in some Muslim communities; practices and evidentiary assessments differ.", Action: "Use the night for general worship without treating a particular practice as obligatory."},
		"ramadan_start":     {Title: "Beginning of Ramadan", Summary: "The month of fasting and Quran begins according to the calculated Hijri calendar.", Action: "Prepare your intention, worship plan, and local moon-sighting confirmation."},
		"last_ten_nights":   {Title: "Last ten nights of Ramadan", Summary: "Laylat al-Qadr is sought in the odd nights of Ramadan’s final ten nights.", Action: "Increase prayer, Quran, charity, and the dua for pardon."},
		"eid_fitr":          {Title: "Eid al-Fitr", Summary: "The celebration completing Ramadan and its prescribed fast.", Action: "Confirm the local date, give Zakat al-Fitr, and join the Eid prayer."},
		"dhul_hijjah_start": {Title: "First ten days of Dhu al-Hijjah", Summary: "These are especially virtuous days for righteous deeds.", Action: "Increase dhikr, charity, prayer, and other good deeds."},
		"arafah":            {Title: "Day of Arafah", Summary: "The ninth of Dhu al-Hijjah is the central day of Hajj and a recommended fast for non-pilgrims.", Action: "If you are not performing Hajj, consider fasting and making abundant dua."},
		"eid_adha":          {Title: "Eid al-Adha", Summary: "The festival of sacrifice begins on the tenth of Dhu al-Hijjah.", Action: "Join the Eid prayer and follow your local guidance for udhiyah."},
	},
	"ar": {
		"ashura":            {Title: "يوم عاشوراء", Summary: "العاشر من المحرم يوم شكر وذكر.", Action: "يُستحب صيام عاشوراء مع يوم قبله أو بعده."},
		"mawlid":            {Title: "المولد النبوي", Summary: "يوافق تاريخًا شائعًا لمولد النبي ﷺ، مع اختلاف المسلمين في التاريخ الدقيق وطريقة إحيائه.", Action: "أكثر من الصلاة والسلام على النبي ﷺ وتعلّم من سيرته."},
		"isra_miraj":        {Title: "الإسراء والمعراج", Summary: "تاريخ شائع لتذكر رحلة الإسراء والمعراج، أما تعيين الليلة بدقة فغير ثابت.", Action: "اقرأ بداية سورة الإسراء وتأمل في نعمة الصلاة."},
		"mid_shaban":        {Title: "ليلة النصف من شعبان", Summary: "ليلة يحييها بعض المسلمين، مع اختلاف العلماء في الأعمال والأدلة الخاصة بها.", Action: "اغتنمها في العبادة العامة دون اعتقاد وجوب عمل مخصوص."},
		"ramadan_start":     {Title: "بداية رمضان", Summary: "يبدأ شهر الصيام والقرآن وفق التاريخ الهجري المحسوب.", Action: "استعد بالنية وخطة العبادة وتحقق من ثبوت الهلال محليًا."},
		"last_ten_nights":   {Title: "العشر الأواخر من رمضان", Summary: "تُتحرى ليلة القدر في الليالي الوترية من العشر الأواخر.", Action: "أكثر من الصلاة والقرآن والصدقة ودعاء العفو."},
		"eid_fitr":          {Title: "عيد الفطر", Summary: "فرحة إتمام رمضان وصيامه المفروض.", Action: "تحقق من التاريخ المحلي وأدِّ زكاة الفطر وصلِّ العيد."},
		"dhul_hijjah_start": {Title: "العشر الأوائل من ذي الحجة", Summary: "أيام فاضلة يُستحب فيها العمل الصالح.", Action: "أكثر من الذكر والصدقة والصلاة وسائر الخير."},
		"arafah":            {Title: "يوم عرفة", Summary: "تاسع ذي الحجة وأعظم أيام الحج، ويُستحب صيامه لغير الحاج.", Action: "إن لم تكن حاجًا ففكر في الصيام وأكثر من الدعاء."},
		"eid_adha":          {Title: "عيد الأضحى", Summary: "يبدأ عيد النحر في العاشر من ذي الحجة.", Action: "صلِّ العيد واتبع الإرشادات المحلية للأضحية."},
	},
	"es": {
		"ashura":            {Title: "Día de Ashura", Summary: "El diez de Muharram es un día de gratitud y recuerdo.", Action: "Considera ayunar Ashura y un día adyacente."},
		"mawlid":            {Title: "Mawlid al-Nabi", Summary: "Fecha comúnmente observada como nacimiento del Profeta; la fecha histórica y su celebración difieren.", Action: "Envía bendiciones al Profeta ﷺ y estudia su carácter."},
		"isra_miraj":        {Title: "Isra y Mi’raj", Summary: "Fecha habitual para recordar el Viaje Nocturno; su fecha exacta no está establecida.", Action: "Lee el inicio de Al-Isra y reflexiona sobre la oración."},
		"mid_shaban":        {Title: "Mitad de Sha’ban", Summary: "Noche observada en algunas comunidades; las prácticas y sus evidencias difieren.", Action: "Dedícala a la adoración general sin considerar obligatoria una práctica concreta."},
		"ramadan_start":     {Title: "Comienzo de Ramadán", Summary: "Comienza el mes del ayuno y del Corán según el calendario calculado.", Action: "Prepara tu intención y confirma el avistamiento lunar local."},
		"last_ten_nights":   {Title: "Últimas diez noches de Ramadán", Summary: "Laylat al-Qadr se busca en las noches impares de las últimas diez.", Action: "Aumenta la oración, el Corán, la caridad y la súplica por el perdón."},
		"eid_fitr":          {Title: "Eid al-Fitr", Summary: "La celebración que completa Ramadán y su ayuno.", Action: "Confirma la fecha local, entrega Zakat al-Fitr y reza el Eid."},
		"dhul_hijjah_start": {Title: "Primeros diez días de Dhu al-Hijjah", Summary: "Días especialmente virtuosos para las buenas obras.", Action: "Aumenta el dhikr, la caridad, la oración y el bien."},
		"arafah":            {Title: "Día de Arafah", Summary: "El nueve de Dhu al-Hijjah es central en el Hajj y se recomienda ayunarlo a quien no peregrina.", Action: "Si no haces el Hajj, considera ayunar y hacer abundante dua."},
		"eid_adha":          {Title: "Eid al-Adha", Summary: "La fiesta del sacrificio comienza el diez de Dhu al-Hijjah.", Action: "Reza el Eid y sigue la orientación local sobre la udhiyah."},
	},
	"fr": {
		"ashura":            {Title: "Jour de Achoura", Summary: "Le dix Mouharram est un jour de gratitude et de rappel.", Action: "Envisagez de jeûner Achoura avec un jour adjacent."},
		"mawlid":            {Title: "Mawlid an-Nabi", Summary: "Date souvent associée à la naissance du Prophète ; la date historique et sa commémoration divergent.", Action: "Priez sur le Prophète ﷺ et étudiez son comportement."},
		"isra_miraj":        {Title: "Isra et Mi’raj", Summary: "Date courante rappelant le Voyage nocturne ; sa date exacte n’est pas établie.", Action: "Lisez le début d’Al-Isra et méditez sur le don de la prière."},
		"mid_shaban":        {Title: "Mi-Chaabane", Summary: "Nuit observée dans certaines communautés ; les pratiques et les preuves divergent.", Action: "Consacrez-la au culte général sans rendre une pratique particulière obligatoire."},
		"ramadan_start":     {Title: "Début du Ramadan", Summary: "Le mois du jeûne et du Coran commence selon le calendrier calculé.", Action: "Préparez votre intention et vérifiez l’observation lunaire locale."},
		"last_ten_nights":   {Title: "Dix dernières nuits du Ramadan", Summary: "Laylat al-Qadr est recherchée durant les nuits impaires des dix dernières.", Action: "Multipliez prière, Coran, aumône et invocation du pardon."},
		"eid_fitr":          {Title: "Aïd al-Fitr", Summary: "La fête qui conclut le Ramadan et son jeûne.", Action: "Confirmez la date locale, donnez Zakat al-Fitr et priez l’Aïd."},
		"dhul_hijjah_start": {Title: "Dix premiers jours de Dhou al-Hijjah", Summary: "Des jours particulièrement vertueux pour les bonnes œuvres.", Action: "Multipliez dhikr, aumône, prière et bonnes actions."},
		"arafah":            {Title: "Jour de Arafat", Summary: "Le neuf Dhou al-Hijjah est central au Hajj et son jeûne est recommandé aux non-pèlerins.", Action: "Si vous ne faites pas le Hajj, envisagez de jeûner et invoquez abondamment."},
		"eid_adha":          {Title: "Aïd al-Adha", Summary: "La fête du sacrifice commence le dix Dhou al-Hijjah.", Action: "Priez l’Aïd et suivez les indications locales pour l’oudhiya."},
	},
	"ru": {
		"ashura":            {Title: "День Ашура", Summary: "Десятый день Мухаррама — день благодарности и поминания.", Action: "Рассмотрите пост в Ашура и соседний день."},
		"mawlid":            {Title: "Маулид ан-Наби", Summary: "Распространённая дата рождения Пророка; точная дата и форма её отмечания различаются.", Action: "Произносите салават Пророку ﷺ и изучайте его нрав."},
		"isra_miraj":        {Title: "Исра и Мирадж", Summary: "Распространённая дата Ночного путешествия; её точное календарное определение не установлено.", Action: "Прочитайте начало суры «Аль-Исра» и размышляйте о даре молитвы."},
		"mid_shaban":        {Title: "Середина Шаабана", Summary: "Ночь отмечается в некоторых общинах; практики и оценка доказательств различаются.", Action: "Посвятите время общему поклонению, не считая отдельную практику обязательной."},
		"ramadan_start":     {Title: "Начало Рамадана", Summary: "Начинается месяц поста и Корана по расчётному календарю.", Action: "Подготовьте намерение и подтвердите местное наблюдение луны."},
		"last_ten_nights":   {Title: "Последние десять ночей Рамадана", Summary: "Ляйлят аль-Кадр ищут в нечётные ночи последней декады.", Action: "Усильте молитву, чтение Корана, милостыню и дуа о прощении."},
		"eid_fitr":          {Title: "Ид аль-Фитр", Summary: "Праздник завершения Рамадана и обязательного поста.", Action: "Уточните местную дату, выплатите закят аль-фитр и совершите праздничную молитву."},
		"dhul_hijjah_start": {Title: "Первые десять дней Зуль-хиджи", Summary: "Особо благословенные дни для праведных дел.", Action: "Увеличьте зикр, милостыню, молитву и добрые дела."},
		"arafah":            {Title: "День Арафа", Summary: "Девятый Зуль-хиджи — главный день хаджа; не паломникам рекомендуется пост.", Action: "Если вы не в хадже, рассмотрите пост и больше обращайтесь с дуа."},
		"eid_adha":          {Title: "Ид аль-Адха", Summary: "Праздник жертвоприношения начинается десятого Зуль-хиджи.", Action: "Совершите праздничную молитву и следуйте местным правилам удхии."},
	},
	"tr": {
		"ashura":            {Title: "Aşure Günü", Summary: "Muharrem’in onu şükür ve hatırlama günüdür.", Action: "Aşure günüyle birlikte önceki veya sonraki günü oruçlu geçirmeyi düşünün."},
		"mawlid":            {Title: "Mevlid-i Nebi", Summary: "Peygamber’in doğumu olarak yaygın anılan tarihtir; kesin tarih ve anma şekli konusunda farklılık vardır.", Action: "Peygamber’e ﷺ salavat getirin ve ahlakını öğrenin."},
		"isra_miraj":        {Title: "İsra ve Miraç", Summary: "Gece Yolculuğu’nun yaygın anma tarihidir; kesin takvim tarihi sabit değildir.", Action: "İsra Suresi’nin başını okuyun ve namaz nimetini düşünün."},
		"mid_shaban":        {Title: "Şaban’ın Ortası", Summary: "Bazı topluluklarda ihya edilen bir gecedir; uygulamalar ve delil değerlendirmeleri farklıdır.", Action: "Belirli bir ameli zorunlu görmeden genel ibadetle değerlendirin."},
		"ramadan_start":     {Title: "Ramazan’ın Başlangıcı", Summary: "Hesaplanan takvime göre oruç ve Kur’an ayı başlar.", Action: "Niyetinizi hazırlayın ve yerel hilal duyurusunu doğrulayın."},
		"last_ten_nights":   {Title: "Ramazan’ın Son On Gecesi", Summary: "Kadir Gecesi son on gecenin tek gecelerinde aranır.", Action: "Namazı, Kur’an’ı, sadakayı ve af duasını artırın."},
		"eid_fitr":          {Title: "Ramazan Bayramı", Summary: "Ramazan ve farz orucun tamamlanmasını kutlar.", Action: "Yerel tarihi doğrulayın, fitre verin ve bayram namazına katılın."},
		"dhul_hijjah_start": {Title: "Zilhicce’nin İlk On Günü", Summary: "Salih ameller için özellikle faziletli günlerdir.", Action: "Zikri, sadakayı, namazı ve iyiliği artırın."},
		"arafah":            {Title: "Arefe Günü", Summary: "Zilhicce’nin dokuzu haccın ana günüdür; hacda olmayanlara oruç tavsiye edilir.", Action: "Hacda değilseniz oruç tutmayı ve çokça dua etmeyi düşünün."},
		"eid_adha":          {Title: "Kurban Bayramı", Summary: "Kurban bayramı Zilhicce’nin onunda başlar.", Action: "Bayram namazına katılın ve kurban için yerel rehberliği izleyin."},
	},
	"uz": {
		"ashura":            {Title: "Ashuro kuni", Summary: "Muharramning o‘ninchi kuni shukr va eslash kunidir.", Action: "Ashuro va unga qo‘shni bir kunda ro‘za tutishni o‘ylab ko‘ring."},
		"mawlid":            {Title: "Mavlid an-Nabiy", Summary: "Payg‘ambar tug‘ilgan kun sifatida keng tarqalgan sana; aniq tarix va nishonlash borasida farq bor.", Action: "Payg‘ambarimizga ﷺ salavot ayting va u zotning xulqini o‘rganing."},
		"isra_miraj":        {Title: "Isro va Me’roj", Summary: "Tungi sayohatni eslash uchun keng tarqalgan sana; aniq kalendar kuni sobit emas.", Action: "Isro surasining boshini o‘qing va namoz ne’matini tafakkur qiling."},
		"mid_shaban":        {Title: "Sha’bon o‘rtasi", Summary: "Ba’zi jamoalarda e’zozlanadigan tun; amallar va dalillar bahosi turlicha.", Action: "Muayyan amalni majburiy sanamasdan umumiy ibodat bilan o‘tkazing."},
		"ramadan_start":     {Title: "Ramazon boshlanishi", Summary: "Hisoblangan taqvim bo‘yicha ro‘za va Qur’on oyi boshlanadi.", Action: "Niyat va ibodat rejangizni tayyorlab, mahalliy hilol xabarini tekshiring."},
		"last_ten_nights":   {Title: "Ramazonning so‘nggi o‘n kechasi", Summary: "Qadr kechasi so‘nggi o‘n kechaning toq kechalarida izlanadi.", Action: "Namoz, Qur’on, sadaqa va afv duosini ko‘paytiring."},
		"eid_fitr":          {Title: "Ramazon hayiti", Summary: "Ramazon va farz ro‘zaning tugash bayrami.", Action: "Mahalliy sanani tasdiqlang, fitr zakotini bering va hayit namoziga boring."},
		"dhul_hijjah_start": {Title: "Zulhijjaning ilk o‘n kuni", Summary: "Yaxshi amallar uchun alohida fazilatli kunlar.", Action: "Zikr, sadaqa, namoz va ezgu ishlarni ko‘paytiring."},
		"arafah":            {Title: "Arafa kuni", Summary: "Zulhijjaning to‘qqizinchi kuni hajning asosiy kuni; hojilarga bo‘lmaganlarga ro‘za tavsiya etiladi.", Action: "Hajda bo‘lmasangiz, ro‘za va ko‘p duo qilishni o‘ylab ko‘ring."},
		"eid_adha":          {Title: "Qurbon hayiti", Summary: "Qurbon bayrami Zulhijjaning o‘ninchi kuni boshlanadi.", Action: "Hayit namoziga boring va qurbonlikda mahalliy ko‘rsatmaga amal qiling."},
	},
	"tt": {
		"ashura":            {Title: "Гашура көне", Summary: "Мөхәррәмнең унынчы көне — шөкер һәм искә алу көне.", Action: "Гашура һәм аңа күрше бер көндә ураза тотуны уйлагыз."},
		"mawlid":            {Title: "Мәүлид ән-Нәби", Summary: "Пәйгамбәрнең туган көне буларак киң билгеләнгән дата; төгәл тарих һәм үткәрү төрлечә.", Action: "Пәйгамбәргә ﷺ салават әйтегез һәм аның әхлагын өйрәнегез."},
		"isra_miraj":        {Title: "Исра һәм Мигъраҗ", Summary: "Төнге сәяхәтне искә алу өчен киң дата; төгәл календарь көне нык билгеләнмәгән.", Action: "Исра сүрәсенең башын укыгыз һәм намаз нигъмәте турында уйланыгыз."},
		"mid_shaban":        {Title: "Шәгъбан уртасы", Summary: "Кайбер җәмгыятьләрдә билгеләнә; гамәлләр һәм дәлилләргә бәя төрле.", Action: "Аерым гамәлне мәҗбүри санамыйча гомуми гыйбадәт кылыгыз."},
		"ramadan_start":     {Title: "Рамазан башлануы", Summary: "Исәпләнгән календарь буенча ураза һәм Коръән ае башлана.", Action: "Ниятегезне әзерләгез һәм җирле ай күренү хәбәрен тикшерегез."},
		"last_ten_nights":   {Title: "Рамазанның соңгы ун төне", Summary: "Кадер кичәсе соңгы ун төннең так кичләрендә эзләнә.", Action: "Намаз, Коръән, сәдака һәм гафу догасын арттырыгыз."},
		"eid_fitr":          {Title: "Ураза бәйрәме", Summary: "Рамазан һәм фарыз уразаның тәмамлану бәйрәме.", Action: "Җирле датаны раслагыз, фитыр сәдакасын бирегез һәм бәйрәм намазына барыгыз."},
		"dhul_hijjah_start": {Title: "Зөлхиҗҗәнең беренче ун көне", Summary: "Изге гамәлләр өчен аеруча фазыйләтле көннәр.", Action: "Зикер, сәдака, намаз һәм яхшылыкны арттырыгыз."},
		"arafah":            {Title: "Гарәфә көне", Summary: "Зөлхиҗҗәнең тугызынчы көне — хаҗның төп көне; хаҗда булмаганнарга ураза киңәш ителә.", Action: "Хаҗда булмасагыз, ураза һәм күп дога кылуны уйлагыз."},
		"eid_adha":          {Title: "Корбан бәйрәме", Summary: "Корбан бәйрәме Зөлхиҗҗәнең унынчы көнендә башлана.", Action: "Бәйрәм намазына барыгыз һәм корбанлыкта җирле күрсәтмәне үтәгез."},
	},
}
