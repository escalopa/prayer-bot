package prayer

import (
	"fmt"
	"time"
)

type PrayerTimes struct {
	Fajr    time.Time `json:"fajr"`
	Dhuhr   time.Time `json:"dhuhr"`
	Asr     time.Time `json:"asr"`
	Maghrib time.Time `json:"maghrib"`
	Isha    time.Time `json:"isha"`
}

func (p *PrayerTimes) EnglishString() string {
	return fmt.Sprintf(
		"Fajr: %s\nDhuhr: %s\nAsr: %s\nMaghrib: %s\nIsha: %s",
		p.Fajr, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}

func (p *PrayerTimes) RussianString() string {
	return fmt.Sprintf(
		"Восход: %s\nЗухр: %s\nАср: %s\nМагриб: %s\nИша: %s",
		p.Fajr, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}

func (p *PrayerTimes) ArabicString() string {
	return fmt.Sprintf(
		"الفجر: %s\nالظهر: %s\nالعصر: %s\nالمغرب: %s\nالعشاء: %s",
		p.Fajr, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}
