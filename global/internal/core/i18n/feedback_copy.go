package i18n

type feedbackCopy struct {
	Button, Command, Prompt, Placeholder, Sent, Private string
}

var feedbackCopies = map[string]feedbackCopy{
	"en": {
		"💬 Feedback", "Send feedback or report a bug",
		"Tell us what went wrong or what could be improved. Reply with a message or screenshot. Your name, username and Telegram ID will be shared privately with the bot owner.",
		"Describe the problem", "Thank you — your feedback was sent to the bot owner ✅", "Please open a private chat with the bot to send feedback.",
	},
	"ar": {
		"💬 ملاحظات أو بلاغ", "إرسال ملاحظة أو الإبلاغ عن مشكلة",
		"أخبرنا بالمشكلة أو بما يمكن تحسينه. أرسل رسالة أو لقطة شاشة ردًا على هذه الرسالة. سيُشارك اسمك واسم المستخدم ومعرّف تيليجرام بشكل خاص مع مالك البوت.",
		"صف المشكلة", "شكرًا لك — تم إرسال ملاحظاتك إلى مالك البوت ✅", "يرجى فتح محادثة خاصة مع البوت لإرسال الملاحظات.",
	},
	"es": {
		"💬 Comentarios", "Enviar comentarios o informar un error",
		"Cuéntanos qué salió mal o qué podemos mejorar. Responde con un mensaje o una captura. Tu nombre, usuario e ID de Telegram se compartirán en privado con el propietario del bot.",
		"Describe el problema", "Gracias: tus comentarios se enviaron al propietario del bot ✅", "Abre un chat privado con el bot para enviar comentarios.",
	},
	"fr": {
		"💬 Avis et problème", "Envoyer un avis ou signaler un problème",
		"Dites-nous ce qui ne va pas ou ce qui peut être amélioré. Répondez avec un message ou une capture. Votre nom, identifiant et ID Telegram seront transmis en privé au propriétaire du bot.",
		"Décrivez le problème", "Merci — votre message a été envoyé au propriétaire du bot ✅", "Ouvrez une discussion privée avec le bot pour envoyer votre avis.",
	},
	"ru": {
		"💬 Отзыв или ошибка", "Отправить отзыв или сообщить об ошибке",
		"Расскажите, что не работает или что можно улучшить. Ответьте сообщением или снимком экрана. Ваше имя, username и Telegram ID будут переданы владельцу бота в личном сообщении.",
		"Опишите проблему", "Спасибо — сообщение отправлено владельцу бота ✅", "Чтобы отправить отзыв, откройте личный чат с ботом.",
	},
	"tr": {
		"💬 Geri bildirim", "Geri bildirim gönder veya hata bildir",
		"Neyin yanlış gittiğini veya neyin geliştirilebileceğini anlatın. Mesaj ya da ekran görüntüsüyle yanıtlayın. Adınız, kullanıcı adınız ve Telegram kimliğiniz bot sahibine özel olarak iletilir.",
		"Sorunu açıklayın", "Teşekkürler — geri bildiriminiz bot sahibine gönderildi ✅", "Geri bildirim için botla özel sohbet açın.",
	},
	"uz": {
		"💬 Fikr yoki xato", "Fikr yuborish yoki xato haqida xabar berish",
		"Nima ishlamagani yoki nimani yaxshilash mumkinligini yozing. Xabar yoki skrinshot bilan javob bering. Ismingiz, username va Telegram ID bot egasiga shaxsiy tarzda yuboriladi.",
		"Muammoni tasvirlang", "Rahmat — fikringiz bot egasiga yuborildi ✅", "Fikr yuborish uchun bot bilan shaxsiy chatni oching.",
	},
	"tt": {
		"💬 Фикер яки хата", "Фикер җибәрү яки хата турында хәбәр итү",
		"Нәрсә эшләмәгәнен яки нәрсәне яхшыртып булганын языгыз. Хәбәр яки экран рәсеме белән җавап бирегез. Исемегез, username һәм Telegram ID бот хуҗасына шәхси рәвештә җибәреләчәк.",
		"Проблеманы тасвирлагыз", "Рәхмәт — фикерегез бот хуҗасына җибәрелде ✅", "Фикер җибәрү өчен бот белән шәхси чат ачыгыз.",
	},
}

func init() {
	for code, copy := range feedbackCopies {
		locale := locales[code]
		locale.Buttons[ActionFeedback] = copy.Button
		locale.Commands[ActionFeedback] = copy.Command
		locale.Text["feedback_prompt"] = copy.Prompt
		locale.Text["feedback_placeholder"] = copy.Placeholder
		locale.Text["feedback_sent"] = copy.Sent
		locale.Text["feedback_private"] = copy.Private
	}
}
