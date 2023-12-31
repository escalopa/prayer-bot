package parser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/escalopa/gopray/telegram/internal/application"
)

var (
	dayFormat   = "2/1/2006"
	clockFormat = "15:04"
)

// PrayerParser is responsible for parsing the prayer schedule.
// It also saves the schedule to the database.
type PrayerParser struct {
	path string // data path
	pr   application.PrayerRepository
}

// NewPrayerParser returns a new PrayerParser.
// @param path: path to the data file.
// @param pr: prayer repository.
func NewPrayerParser(path string, opts ...func(*PrayerParser)) *PrayerParser {
	p := &PrayerParser{path: path}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithPrayerRepository sets the prayer repository for the parser.
func WithPrayerRepository(pr application.PrayerRepository) func(*PrayerParser) {
	return func(p *PrayerParser) {
		p.pr = pr
	}
}

// ParseSchedule parses the prayer schedule and saves it to the database.
func (p *PrayerParser) ParseSchedule(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.Open(p.path)
	if err != nil {
		return err
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Printf("failed to close csv file, %s", err)
		}
	}()

	schedule, err := p.parseSchedule(file)
	if err != nil {
		return fmt.Errorf("failed to parse schedule: %v", err)
	}
	err = p.saveSchedule(ctx, schedule)
	if err != nil {
		return fmt.Errorf("failed to save schedule to database: %v", err)
	}

	return nil
}

// parseSchedule parses all prayers from file
func (p *PrayerParser) parseSchedule(file io.Reader) (schedule []core.PrayerTime, err error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 7 // 7 fields per record (date, fajr, sunrise, dhuhr, asr, maghrib, isha)
	reader.TrimLeadingSpace = true

	// skip header
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	// parse data
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		if len(record) != 7 {
			return nil, fmt.Errorf("invalid record length, expected 7, got %d", len(record))
		}

		day, err := p.parseDate(record[0])
		if err != nil {
			return nil, fmt.Errorf("failed to read day: %v", err)
		}

		// Parse prayers times and convert into time.Time
		prayers, err := p.parsePrayer(record[1:], day) // skip first record since it was date
		if err != nil {
			return nil, fmt.Errorf("failed to read prayer for day: %v", err)
		}

		// Add 20 min  since Dohaa is 20 min after sunrise
		prayers[1] = prayers[1].Add(20 * time.Minute)

		schedule = append(schedule, core.NewPrayerTime(day,
			prayers[0], // Fajr
			prayers[1], // Dohaa
			prayers[2], // Dhuhr
			prayers[3], // Asr
			prayers[4], // Maghrib
			prayers[5], // Isha
		))
	}

	return
}

// parseDate get day date from string
func (p *PrayerParser) parseDate(line string) (time.Time, error) {
	t, err := time.Parse(dayFormat, line)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse day date: %v", err)
	}
	return core.DefaultTime(t.Day(), int(t.Month()), t.Year()), nil
}

// parsePrayer parses all day's prayers
func (p *PrayerParser) parsePrayer(prayersStr []string, day time.Time) ([]time.Time, error) {
	if len(prayersStr) != 6 {
		return nil, fmt.Errorf("exptected len of 6 for the day prayers")
	}
	prayers := make([]time.Time, 6)
	for i := 0; i <= 5; i++ {
		prayer, err := p.convertToTime(prayersStr[i], day)
		if err != nil {
			return nil, err
		}
		prayers[i] = prayer
	}
	return prayers, nil
}

// convertToTime converts a string from format `hh:mm` to time.Time.
func (p *PrayerParser) convertToTime(str string, day time.Time) (time.Time, error) {
	t, err := time.Parse(clockFormat, str)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get time in hours and minute for %s: %v", str, err)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, core.GetLocation()), nil
}

// saveSchedule saves the schedule to the database.
func (p *PrayerParser) saveSchedule(ctx context.Context, schedule []core.PrayerTime) error {
	// Loop through all days of the schedule and save them to the database
	for _, prayers := range schedule {
		err := p.pr.StorePrayer(ctx, prayers)
		if err != nil {
			return err
		}
	}
	return nil
}
