package config

import (
	"strconv"
	"time"
)

type timeLoc time.Location

func (t *timeLoc) UnmarshalJSON(data []byte) error {
	value, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(value)
	if err != nil {
		return err
	}

	*t = timeLoc(*loc)
	return nil
}

func (t *timeLoc) Get() *time.Location {
	return (*time.Location)(t)
}

func (t *timeLoc) String() string {
	tl := time.Location(*t)
	return tl.String()
}
