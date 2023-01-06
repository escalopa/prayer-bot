package parser

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	gpe "github.com/escalopa/gopray/pkg/error"
	"github.com/escalopa/gopray/pkg/prayer"
	"github.com/escalopa/gopray/telegram/internal/application"
)

// Parser is responsible for parsing the prayer schedule.
// It also saves the schedule to the database.
type Parser struct {
	path string // data path
	pr   application.PrayerRepository
}

// New returns a new Parser.
// @param path: path to the data file.
// @param pr: prayer repository.
func New(path string, pr application.PrayerRepository) *Parser {
	return &Parser{
		path: path,
		pr:   pr,
	}
}

// ParseSchedule returns prayer times for all days of the year.
func (p *Parser) ParseSchedule() error {
	file, err := os.Open(p.path)
	if err != nil {
		return err
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Printf("Error: %s, Failed to close file", err)
		}
	}()

	// parse csv file
	var schedule []prayer.PrayerTimes
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 7 // 7 fields per record (date, fajr, sunrise, dhuhr, asr, maghrib, isha)
	reader.TrimLeadingSpace = true

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return err
	}
	// Parse data
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		// Parse date DD/MM
		date := record[0]
		ss := strings.Split(date, "/")
		day, err := strconv.Atoi(ss[0])
		gpe.CheckError(err)
		month, err := strconv.Atoi(ss[1])
		gpe.CheckError(err)

		// Parse prayers times
		fajr, err := convertToTime(record[1], day, month)
		gpe.CheckError(err)
		sunrise, err := convertToTime(record[2], day, month)
		gpe.CheckError(err)
		dhuhr, err := convertToTime(record[3], day, month)
		gpe.CheckError(err)
		asr, err := convertToTime(record[4], day, month)
		gpe.CheckError(err)
		maghrib, err := convertToTime(record[5], day, month)
		gpe.CheckError(err)
		isha, err := convertToTime(record[6], day, month)
		gpe.CheckError(err)

		prayers := prayer.New(record[0],
			fajr, sunrise, dhuhr, asr, maghrib, isha,
		)
		schedule = append(schedule, prayers)
	}
	log.Println("Parsed prayers schedule")

	// Save to database
	err = p.saveSchedule(schedule)
	if err != nil {
		return err
	}
	log.Println("Saved prayers schedule")
	return nil
}

// saveSchedule saves the schedule to the database.
// @param schedule: prayer times for all days of the year.
// @return error: error if any.
func (p *Parser) saveSchedule(schedule []prayer.PrayerTimes) error {
	// Loop through all days of the schedule and save them to the database
	for _, prayers := range schedule {
		err := p.pr.StorePrayer(prayers.Date, prayers)
		if err != nil {
			return err
		}
	}
	return nil
}

// convertToTime converts a string from format `dd:mm` to time.Time.
// @param str: string to convert.
// @return time.Time: converted time.
func convertToTime(str string, day, month int) (time.Time, error) {
	ss := strings.Split(str, ":")
	hour, err := strconv.Atoi(ss[0])
	if err != nil {
		return time.Time{}, err
	}
	minute, err := strconv.Atoi(ss[1])
	if err != nil {
		return time.Time{}, err
	}

	// Get location for Moscow
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(time.Now().Year(), time.Month(month), day, hour, minute, 0, 0, loc), nil
}
