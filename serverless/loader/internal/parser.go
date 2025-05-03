package internal

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/escalopa/prayer-bot/domain"
)

var (
	dayFormat   = "2/1/2006"
	clockFormat = "15:04"

	loc = time.UTC
)

const (
	prayersCount = 6
	columnsCount = 7 // (date, fajr, shuruq, dhuhr, asr, maghrib, isha)
)

func parsePrayers(file io.Reader) (schedule []*domain.PrayerTimes, err error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = columnsCount
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

		prayer, err := parseRecord(record)
		if err != nil {
			return nil, err
		}

		schedule = append(schedule, prayer)
	}

	return
}

// parseRecord parses a record from the file
func parseRecord(record []string) (*domain.PrayerTimes, error) {
	if len(record) != columnsCount {
		return nil, fmt.Errorf("parseRecord: invalid number of fields, expected 7, got %d", len(record))
	}

	// parse day date
	day, err := parseDate(record[0])
	if err != nil {
		return nil, err
	}

	// parse prayers times and convert into time.Time
	prayers, err := parsePrayer(record[1:], day) // skip first record since it was date
	if err != nil {
		return nil, err
	}

	// add 20 min since `shuruq` is 20 min after sunrise
	prayers[1] = prayers[1].Add(20 * time.Minute)

	prayer := domain.NewPrayerTimes(
		day,
		prayers[0], // fajr
		prayers[1], // shuruq
		prayers[2], // dhuhr
		prayers[3], // asr
		prayers[4], // maghrib
		prayers[5], // isha
	)

	return prayer, nil
}

// parseDate get day date from string
func parseDate(line string) (time.Time, error) {
	t, err := time.Parse(dayFormat, line)
	if err != nil {
		return time.Time{}, fmt.Errorf("parseDate[%s]: %v", line, err)
	}
	return domain.Date(t.Day(), t.Month(), t.Year(), loc), nil
}

// parsePrayer parses all day's prayers
func parsePrayer(prayersStr []string, day time.Time) ([]time.Time, error) {
	if len(prayersStr) != prayersCount {
		return nil, fmt.Errorf("parsePrayer: unexpected number of prayers, expected 6, got %d", len(prayersStr))
	}

	// convert prayers array to []time.Time
	prayers := make([]time.Time, prayersCount)
	for i, prayerTimeStr := range prayersStr {
		prayer, err := convertToTime(prayerTimeStr, day, loc)
		if err != nil {
			return nil, err
		}
		prayers[i] = prayer
	}

	return prayers, nil
}

// convertToTime converts a string from format `hh:mm` to time.Time.
func convertToTime(str string, day time.Time, loc *time.Location) (time.Time, error) {
	clock, err := time.Parse(clockFormat, str)
	if err != nil {
		return time.Time{}, fmt.Errorf("convertToTime[%s]: %v", str, err)
	}
	return domain.DateTime(day, clock, loc), nil
}
