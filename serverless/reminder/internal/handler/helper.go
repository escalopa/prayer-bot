package handler

import (
	"fmt"

	"github.com/escalopa/prayer-bot/domain"
)

const (
	prayerDayFormat  = "02.01.2006"
	prayerTimeFormat = "15:04"

	prayerText = `
🗓 %s

🕊 %s — %s
🌤 %s — %s
☀️ %s — %s
🌇 %s — %s
🌅 %s — %s
🌙 %s — %s
`
)

func (h *Handler) formatPrayerDay(botID int64, prayerDay *domain.PrayerDay, languageCode string) string {
	loc := h.cfg[botID].Location.V()
	text := h.lp.GetText(languageCode)

	return fmt.Sprintf(prayerText,
		prayerDay.Date.Format(prayerDayFormat),
		text.Prayer[int(domain.PrayerIDFajr)], prayerDay.Fajr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDShuruq)], prayerDay.Shuruq.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDDhuhr)], prayerDay.Dhuhr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDAsr)], prayerDay.Asr.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDMaghrib)], prayerDay.Maghrib.In(loc).Format(prayerTimeFormat),
		text.Prayer[int(domain.PrayerIDIsha)], prayerDay.Isha.In(loc).Format(prayerTimeFormat),
	)
}
