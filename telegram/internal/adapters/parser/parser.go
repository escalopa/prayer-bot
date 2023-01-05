package parser

import (
	"encoding/csv"
	"os"

	"github.com/escalopa/gopray/pkg/prayer"
	"github.com/escalopa/gopray/telegram/internal/application"
)

type Parser struct {
	p  string // data path
	pr application.PrayerRepository
}

// New returns a new Parser.
// @param path: path to the data file.
// @param pr: prayer repository.
func New(path string, pr application.PrayerRepository) *Parser {
	return &Parser{
		p:  path,
		pr: pr,
	}
}

// ParseSchedule returns prayer times for all days of the year.
func (p *Parser) ParseSchedule() error {
	file, err := os.Open(p.p)
	if err != nil {
		return err
	}

	defer file.Close()
	// parse csv file
	var schedule []prayer.PrayerTimes
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 7
	reader.TrimLeadingSpace = true

	// Parse data (Including header)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		prayers := prayer.NewPrayerTimes(record[0],
			record[1], record[2], record[3],
			record[4], record[5], record[6],
		)
		schedule = append(schedule, prayers)
	}

	err = p.saveSchedule(schedule)
	if err != nil {
		return err
	}
	return nil
}

// saveSchedule saves the schedule to the database.
// @param schedule: prayer times for all days of the year.
func (p *Parser) saveSchedule(schedule []prayer.PrayerTimes) error {
	for _, prayers := range schedule {
		err := p.pr.StorePrayer(prayers.Date, prayers)
		if err != nil {
			return err
		}
	}
	return nil
}
