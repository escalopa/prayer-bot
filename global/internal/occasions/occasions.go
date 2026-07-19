// Package occasions provides the curated Islamic-date catalog used by the
// Mini App, calendar feed, and reminder planner.
package occasions

import (
	"fmt"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/hijri"
)

type Category string

const (
	CategoryMajor    Category = "major"
	CategoryFasting  Category = "fasting"
	CategoryObserved Category = "observed"
)

type Source struct {
	Label string
	URL   string
}

type Definition struct {
	ID       string
	Month    int
	Day      int
	Category Category
	Emoji    string
	Sources  []Source
}

type Occurrence struct {
	Definition Definition
	Date       time.Time
	Hijri      hijri.Date
}

var catalog = []Definition{
	{
		ID: "ashura", Month: 1, Day: 10, Category: CategoryFasting, Emoji: "🌊",
		Sources: []Source{{Label: "Sahih Muslim 1162a", URL: "https://sunnah.com/muslim:1162a"}},
	},
	{
		ID: "mawlid", Month: 3, Day: 12, Category: CategoryObserved, Emoji: "ﷺ",
		Sources: []Source{
			{Label: "Quran 33:56", URL: "https://quran.com/33/56"},
			{Label: "Sahih Muslim 1162e", URL: "https://sunnah.com/muslim:1162e"},
		},
	},
	{
		ID: "isra_miraj", Month: 7, Day: 27, Category: CategoryObserved, Emoji: "✨",
		Sources: []Source{{Label: "Quran 17:1", URL: "https://quran.com/17/1"}},
	},
	{
		ID: "mid_shaban", Month: 8, Day: 15, Category: CategoryObserved, Emoji: "🌕",
	},
	{
		ID: "ramadan_start", Month: 9, Day: 1, Category: CategoryMajor, Emoji: "🌙",
		Sources: []Source{{Label: "Quran 2:185", URL: "https://quran.com/2/185"}},
	},
	{
		ID: "last_ten_nights", Month: 9, Day: 21, Category: CategoryMajor, Emoji: "🤲",
		Sources: []Source{
			{Label: "Surah Al-Qadr", URL: "https://quran.com/al-qadr"},
			{Label: "Sahih al-Bukhari 2017", URL: "https://sunnah.com/bukhari:2017"},
			{Label: "Jami at-Tirmidhi 3513", URL: "https://sunnah.com/tirmidhi/48/144"},
		},
	},
	{
		ID: "eid_fitr", Month: 10, Day: 1, Category: CategoryMajor, Emoji: "🎉",
		Sources: []Source{{Label: "Quran 2:185", URL: "https://quran.com/2/185"}},
	},
	{
		ID: "dhul_hijjah_start", Month: 12, Day: 1, Category: CategoryMajor, Emoji: "🕋",
		Sources: []Source{{Label: "Sahih al-Bukhari 969", URL: "https://sunnah.com/bukhari:969"}},
	},
	{
		ID: "arafah", Month: 12, Day: 9, Category: CategoryFasting, Emoji: "🤍",
		Sources: []Source{{Label: "Sahih Muslim 1162a", URL: "https://sunnah.com/muslim:1162a"}},
	},
	{
		ID: "eid_adha", Month: 12, Day: 10, Category: CategoryMajor, Emoji: "🐑",
		Sources: []Source{{Label: "Quran 22:36", URL: "https://quran.com/22/36"}},
	},
}

func Catalog() []Definition {
	result := make([]Definition, len(catalog))
	for index, definition := range catalog {
		result[index] = definition
		result[index].Sources = append([]Source(nil), definition.Sources...)
	}
	return result
}

func Between(start time.Time, days, adjustment int) ([]Occurrence, error) {
	if days < 1 || days > 400 {
		return nil, fmt.Errorf("occasion range must be between 1 and 400 days")
	}
	var result []Occurrence
	for offset := 0; offset < days; offset++ {
		date := start.AddDate(0, 0, offset)
		hijriDate, err := hijri.FromGregorian(date, adjustment)
		if err != nil {
			return nil, err
		}
		for _, definition := range catalog {
			if definition.Month == hijriDate.Month && definition.Day == hijriDate.Day {
				result = append(result, Occurrence{
					Definition: definition,
					Date:       date,
					Hijri:      hijriDate,
				})
			}
		}
	}
	return result, nil
}

func Next(start time.Time, adjustment int, category Category) (Occurrence, error) {
	upcoming, err := Between(start, 400, adjustment)
	if err != nil {
		return Occurrence{}, err
	}
	for _, occurrence := range upcoming {
		if occurrence.Definition.Category == category {
			return occurrence, nil
		}
	}
	return Occurrence{}, fmt.Errorf("no %s occasion found in the next 400 days", category)
}

func OnDate(date time.Time, adjustment int, category Category) (Occurrence, bool) {
	upcoming, err := Between(date, 1, adjustment)
	if err != nil {
		return Occurrence{}, false
	}
	for _, occurrence := range upcoming {
		if occurrence.Definition.Category == category {
			return occurrence, true
		}
	}
	return Occurrence{}, false
}
