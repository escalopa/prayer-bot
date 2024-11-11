package parser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"

	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/escalopa/gopray/telegram/internal/domain"
	"github.com/pkg/errors"
)

var (
	dayFormat   = "2/1/2006"
	clockFormat = "15:04"
)

// PrayerParser is responsible for parsing the prayer schedule.
// It also saves the schedule to the database.
type PrayerParser struct {
	path string // data-path
	pr   app.PrayerRepository
	loc  *time.Location
}

func NewPrayerParser(
	path string,
	pr app.PrayerRepository,
	loc *time.Location,
) *PrayerParser {
	return &PrayerParser{
		path: path,
		pr:   pr,
		loc:  loc,
	}
}

// LoadSchedule parses the prayer schedule and saves it to the database.
func (p *PrayerParser) LoadSchedule(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := os.Open(p.path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	schedule, err := p.loadSchedule(file)
	if err != nil {
		return fmt.Errorf("PrayerParser.loadSchedule: %v", err)
	}

	err = p.saveSchedule(ctx, schedule)
	if err != nil {
		return fmt.Errorf("PrayerParser.saveSchedule: %v", err)
	}

	return nil
}

// loadSchedule parses all prayers from file
func (p *PrayerParser) loadSchedule(file io.Reader) (schedule []*domain.PrayerTime, err error) {
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
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		prayer, err := p.parseRecord(record)
		if err != nil {
			return nil, err
		}

		schedule = append(schedule, prayer)
	}

	return
}

// parseRecord parses a record from the file
func (p *PrayerParser) parseRecord(record []string) (*domain.PrayerTime, error) {
	if len(record) != 7 {
		return nil, errors.Errorf("PrayerParser.parseRecord: invalid number of fields, expected 7, got %d", len(record))
	}

	// Parse day date
	day, err := p.parseDate(record[0])
	if err != nil {
		return nil, err
	}

	// Parse prayers times and convert into time.Time
	prayers, err := p.parsePrayer(record[1:], day) // skip first record since it was date
	if err != nil {
		return nil, err
	}

	// Add 20 min since Dohaa is 20 min after sunrise
	prayers[1] = prayers[1].Add(20 * time.Minute)

	prayer := domain.NewPrayerTime(day,
		prayers[0], // Fajr
		prayers[1], // Dohaa
		prayers[2], // Dhuhr
		prayers[3], // Asr
		prayers[4], // Maghrib
		prayers[5], // Isha
	)

	return prayer, nil
}

// parseDate get day date from string
func (p *PrayerParser) parseDate(line string) (time.Time, error) {
	t, err := time.Parse(dayFormat, line)
	if err != nil {
		return time.Time{}, fmt.Errorf("PrayerParser.parseDate[%s]: %v", line, err)
	}
	return domain.Time(t.Day(), t.Month(), t.Year()), nil
}

// parsePrayer parses all day's prayers
func (p *PrayerParser) parsePrayer(prayersStr []string, day time.Time) ([]time.Time, error) {
	if len(prayersStr) != 6 {
		return nil, errors.Errorf("PrayerParser.parsePrayer: invalid number of prayers, expected 6, got %d", len(prayersStr))
	}

	// Convert prayers array to []time.Time
	prayers := make([]time.Time, 6)
	for i, prayerTimeStr := range prayersStr {
		prayer, err := p.convertToTime(prayerTimeStr, day)
		if err != nil {
			return nil, err
		}
		prayers[i] = prayer
	}

	return prayers, nil
}

// convertToTime converts a string from format `hh:mm` to time.Time.
func (p *PrayerParser) convertToTime(str string, day time.Time) (time.Time, error) {
	clock, err := time.Parse(clockFormat, str)
	if err != nil {
		return time.Time{}, errors.Errorf("PrayerParser.convertToTime[%s]: %v", str, err)
	}
	fmt.Printf(" %s | %s | %d:%d \n", day.String(), str, clock.Hour(), clock.Minute())
	return domain.Date(day, clock), nil
}

// saveSchedule saves the schedule to the database.
func (p *PrayerParser) saveSchedule(ctx context.Context, schedule []*domain.PrayerTime) error {
	for _, prayers := range schedule {
		if err := p.pr.StorePrayer(ctx, prayers); err != nil {
			return err
		}
	}
	return nil
}
