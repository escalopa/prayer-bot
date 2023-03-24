package parser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/escalopa/gopray/pkg/core"
	"github.com/escalopa/gopray/telegram/internal/application"
)

// PrayerParser is responsible for parsing the prayer schedule.
// It also saves the schedule to the database.
type PrayerParser struct {
	path string // data path
	pr   application.PrayerRepository
	loc  *time.Location
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

// WithTimeLocation sets the time location for the parser.
func WithTimeLocation(loc *time.Location) func(*PrayerParser) {
	return func(p *PrayerParser) {
		p.loc = loc
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

	// parse csv file
	var schedule []core.PrayerTimes
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 7 // 7 fields per record (date, fajr, sunrise, dhuhr, asr, maghrib, isha)
	reader.TrimLeadingSpace = true

	// skip header
	_, err = reader.Read()
	if err != nil {
		return err
	}
	// Parse data
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		if len(record) != 7 {
			return errors.New(fmt.Sprintf("invalid record length, expected 7, got %d", len(record)))
		}

		// Parse date DD/MM
		date := record[0]
		ss := strings.Split(date, "/")
		if len(ss) != 2 {
			return errors.New(fmt.Sprintf("invalid date format, expected DD/MM, got %s", date))
		}
		day, err := strconv.Atoi(ss[0])
		if err != nil {
			return err
		}
		month, err := strconv.Atoi(ss[1])
		if err != nil {
			return err
		}

		// Parse prayers times and convert into time.Time
		fajr, err := p.convertToTime(record[1], day, month)
		if err != nil {
			return err
		}
		sunrise, err := p.convertToTime(record[2], day, month)
		if err != nil {
			return err
		}
		dhuhr, err := p.convertToTime(record[3], day, month)
		if err != nil {
			return err
		}
		asr, err := p.convertToTime(record[4], day, month)
		if err != nil {
			return err
		}
		maghrib, err := p.convertToTime(record[5], day, month)
		if err != nil {
			return err
		}
		isha, err := p.convertToTime(record[6], day, month)
		if err != nil {
			return err
		}

		prayers := core.New(day, month, fajr, sunrise, dhuhr, asr, maghrib, isha)
		schedule = append(schedule, prayers)
	}

	// Save to database
	err = p.saveSchedule(ctx, schedule)
	if err != nil {
		return errors.Wrap(err, "failed to save schedule to database")
	}
	return nil
}

// saveSchedule saves the schedule to the database.
// @param schedule: prayer times for all days of the year.
// @return error: error if any.
func (p *PrayerParser) saveSchedule(ctx context.Context, schedule []core.PrayerTimes) error {
	// Loop through all days of the schedule and save them to the database
	for _, prayers := range schedule {
		err := p.pr.StorePrayer(ctx, prayers)
		if err != nil {
			return err
		}
	}
	return nil
}

// convertToTime converts a string from format `dd:mm` to time.Time.
// @param str: string to convert.
// @return time.Time: converted time.
func (p *PrayerParser) convertToTime(str string, day, month int) (time.Time, error) {
	ss := strings.Split(str, ":")
	if len(ss) != 2 {
		return time.Time{}, errors.New(fmt.Sprintf("invalid time format, expected HH:MM, got %s", str))
	}
	// Parse hour
	hour, err := strconv.Atoi(ss[0])
	if err != nil {
		return time.Time{}, errors.New(fmt.Sprintf("invalid hour, expected HH, got %s", ss[0]))
	}
	if hour < 0 || hour > 23 {
		return time.Time{}, errors.New(fmt.Sprintf("invalid hour, expected HH in range [0, 23], got %d", hour))
	}
	// Parse minute
	minute, err := strconv.Atoi(ss[1])
	if err != nil {
		return time.Time{}, errors.New(fmt.Sprintf("invalid minute, expected MM, got %s", ss[1]))
	}
	if minute < 0 || minute > 59 {
		return time.Time{}, errors.New(fmt.Sprintf("invalid minute, expected MM in range [0, 59], got %d", minute))
	}
	return time.Date(time.Now().Year(), time.Month(month), day, hour, minute, 0, 0, p.loc), nil
}
