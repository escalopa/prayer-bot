// Package hijri converts local Gregorian dates to the calculated Umm al-Qura
// calendar used by the bot.
package hijri

import (
	"fmt"
	"time"

	gohijri "github.com/hablullah/go-hijri"
)

type Date struct {
	Day   int
	Month int
	Year  int
}

// FromGregorian returns a calculated Umm al-Qura date. Adjustment is a user
// supplied regional moon-sighting correction from -2 to +2 days. Dates outside
// the published Umm al-Qura table fall back to the arithmetic Hijri calendar.
func FromGregorian(date time.Time, adjustment int) (Date, error) {
	if adjustment < -2 || adjustment > 2 {
		return Date{}, fmt.Errorf("hijri adjustment must be between -2 and 2")
	}
	adjusted := date.AddDate(0, 0, adjustment)
	ummAlQura, err := gohijri.CreateUmmAlQuraDate(adjusted)
	if err == nil {
		return Date{Day: int(ummAlQura.Day), Month: int(ummAlQura.Month), Year: int(ummAlQura.Year)}, nil
	}
	arithmetic, arithmeticErr := gohijri.CreateHijriDate(adjusted, gohijri.Default)
	if arithmeticErr != nil {
		return Date{}, fmt.Errorf("convert Gregorian date to Hijri: %w", arithmeticErr)
	}
	return Date{Day: int(arithmetic.Day), Month: int(arithmetic.Month), Year: int(arithmetic.Year)}, nil
}
