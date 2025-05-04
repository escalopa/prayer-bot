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

func parsePrayerDays(file io.Reader) (prayerDays []*domain.PrayerDay, err error) {
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

		prayerDay, err := parseRecord(record)
		if err != nil {
			return nil, err
		}

		prayerDays = append(prayerDays, prayerDay)
	}

	return
}

// parseRecord parses a record from the file
func parseRecord(record []string) (*domain.PrayerDay, error) {
	if len(record) != columnsCount {
		return nil, fmt.Errorf("parseRecord: invalid number of fields, expected 7, got %d", len(record))
	}

	// parse prayerDay's date
	date, err := parseDate(record[0])
	if err != nil {
		return nil, err
	}

	// parse prayerDay's times
	prayers, err := parsePrayer(record[1:], date) // skip first record since it was date
	if err != nil {
		return nil, err
	}

	// add 20 min since `shuruq` is 20 min after sunrise
	prayers[1] = prayers[1].Add(20 * time.Minute)

	prayerDay := domain.NewPrayerDay(
		date,
		prayers[0], // fajr
		prayers[1], // shuruq
		prayers[2], // dhuhr
		prayers[3], // asr
		prayers[4], // maghrib
		prayers[5], // isha
	)

	return prayerDay, nil
}

// parseDate get day date from string
func parseDate(line string) (time.Time, error) {
	t, err := time.Parse(dayFormat, line)
	if err != nil {
		return time.Time{}, fmt.Errorf("parseDate[%s]: %v", line, err)
	}
	return domain.Date(t.Day(), t.Month(), t.Year(), loc), nil
}

// parsePrayer parses prayerDay's times
func parsePrayer(prayerTimes []string, day time.Time) ([]time.Time, error) {
	if len(prayerTimes) != prayersCount {
		return nil, fmt.Errorf("unexpected number of prayers: expected: %d got: %d", prayersCount, len(prayerTimes))
	}

	// convert prayers array to []time.Time
	prayers := make([]time.Time, prayersCount)
	for i, prayerTime := range prayerTimes {
		prayer, err := convertToTime(prayerTime, day, loc)
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
