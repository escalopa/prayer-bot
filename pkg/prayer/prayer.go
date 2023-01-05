package prayer

import (
	"encoding/json"
	"fmt"
)

type Prayer string

type PrayerTimes struct {
	Date    string `json:"date"`
	Fajr    Prayer `json:"fajr"`
	Sunrise Prayer `json:"sunrise"`
	Dhuhr   Prayer `json:"dhuhr"`
	Asr     Prayer `json:"asr"`
	Maghrib Prayer `json:"maghrib"`
	Isha    Prayer `json:"isha"`
}

func NewPrayerTimes(date, fajr, sunrise, dhuhr, asr, maghrib, isha string) PrayerTimes {
	return PrayerTimes{
		Date:    date,
		Fajr:    Prayer(fajr),
		Sunrise: Prayer(sunrise),
		Dhuhr:   Prayer(dhuhr),
		Asr:     Prayer(asr),
		Maghrib: Prayer(maghrib),
		Isha:    Prayer(isha),
	}
}

func (p PrayerTimes) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p PrayerTimes) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}

func (p *PrayerTimes) EnString() string {
	return fmt.Sprintf(
		`
		Date    : %s
		Fajr    : %s
		Sunrise : %s
		Dhuhr   : %s
		Asr     : %s
		Maghrib : %s
		Isha    : %s
		`, p.Date, p.Fajr, p.Sunrise, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}

// HTML returns prayer times in HTML format
func (p *PrayerTimes) EnHTML() string {
	return fmt.Sprintf(
		`
		<b>Date</b>    : %s
		<b>Fajr</b>    : %s
		<b>Sunrise</b> : %s
		<b>Dhuhr</b>   : %s
		<b>Asr</b>     : %s
		<b>Maghrib</b> : %s
		<b>Isha</b>    : %s
		`, p.Date, p.Fajr, p.Sunrise, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}

func (p *PrayerTimes) ArString() string {
	return fmt.Sprintf(
		`
		التاريخ : %s
		الفجر  : %s
		الشروق : %s
		الظهر  : %s
		العصر  : %s
		المغرب : %s
		العشاء : %s
		`, p.Date, p.Fajr, p.Sunrise, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}

func (p *PrayerTimes) ArHTML() string {
	return fmt.Sprintf(
		`
		<b>التاريخ</b> : %s
		<b>الفجر</b>  : %s
		<b>الشروق</b> : %s
		<b>الظهر </b> : %s
		<b>العصر</b>  : %s
		<b>المغرب</b> : %s
		<b>العشاء</b> : %s
		
		`, p.Date, p.Fajr, p.Sunrise, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}

// Russian
func (p *PrayerTimes) RuString() string {
	return fmt.Sprintf(
		`
		Дата    : %s
		Фаджр  : %s
		Восход : %s
		Зухр   : %s
		Аср    : %s
		Магриб : %s
		Иша    : %s
		`, p.Date, p.Fajr, p.Sunrise, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}

func (p *PrayerTimes) RuHTML() string {
	return fmt.Sprintf(
		`
		%s
		<b>Фаджр</b>  : %s
		<b>Восход</b> : %s
		<b>Зухр </b>  : %s
		<b>Аср</b>    : %s
		<b>Магриб</b> : %s
		<b>Иша</b>    : %s
		`, p.Date, p.Fajr, p.Sunrise, p.Dhuhr, p.Asr, p.Maghrib, p.Isha,
	)
}
