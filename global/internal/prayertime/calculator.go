package prayertime

import (
	"context"
	"fmt"
	"sync"
	"time"
	_ "time/tzdata"

	prayer "github.com/hablullah/go-prayer"

	"github.com/escalopa/prayer-bot/global/internal/domain"
)

type Calculator interface {
	Day(context.Context, time.Time, domain.PrayerProfile) (domain.DaySchedule, error)
}

type LocalCalculator struct {
	mu    sync.Mutex
	cache map[string][]prayer.Schedule
}

func New() *LocalCalculator {
	return &LocalCalculator{cache: make(map[string][]prayer.Schedule)}
}

func (c *LocalCalculator) Day(_ context.Context, date time.Time, profile domain.PrayerProfile) (domain.DaySchedule, error) {
	if err := profile.Validate(); err != nil {
		return domain.DaySchedule{}, fmt.Errorf("validate profile: %w", err)
	}
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return domain.DaySchedule{}, fmt.Errorf("load timezone: %w", err)
	}
	localDate := date.In(location)
	year := localDate.Year()
	key := cacheKey(profile, year)

	c.mu.Lock()
	schedules, ok := c.cache[key]
	c.mu.Unlock()
	if !ok {
		schedules, err = prayer.Calculate(toLibraryConfig(profile, location), year)
		if err != nil {
			return domain.DaySchedule{}, fmt.Errorf("calculate prayer times: %w", err)
		}
		c.mu.Lock()
		if len(c.cache) >= 256 {
			clear(c.cache)
		}
		c.cache[key] = schedules
		c.mu.Unlock()
	}

	wanted := localDate.Format("2006-01-02")
	for _, schedule := range schedules {
		if schedule.Date != wanted {
			continue
		}
		return domain.DaySchedule{
			Date:     localDate,
			Timezone: profile.Timezone,
			Times: map[domain.Prayer]time.Time{
				domain.PrayerFajr:    schedule.Fajr,
				domain.PrayerSunrise: schedule.Sunrise,
				domain.PrayerDhuhr:   schedule.Zuhr,
				domain.PrayerAsr:     schedule.Asr,
				domain.PrayerMaghrib: schedule.Maghrib,
				domain.PrayerIsha:    schedule.Isha,
			},
		}, nil
	}
	return domain.DaySchedule{}, fmt.Errorf("no prayer schedule for %s", wanted)
}

func toLibraryConfig(profile domain.PrayerProfile, location *time.Location) prayer.Config {
	minute := time.Minute
	cfg := prayer.Config{
		Latitude:            profile.Latitude,
		Longitude:           profile.Longitude,
		Timezone:            location,
		TwilightConvention:  convention(profile.Method),
		AsrConvention:       prayer.Shafii,
		HighLatitudeAdapter: highLatitudeAdapter(profile.HighLatitudeRule),
		Corrections: prayer.ScheduleCorrections{
			Fajr:    time.Duration(profile.Adjustments.Fajr) * minute,
			Sunrise: time.Duration(profile.Adjustments.Sunrise) * minute,
			Zuhr:    time.Duration(profile.Adjustments.Dhuhr) * minute,
			Asr:     time.Duration(profile.Adjustments.Asr) * minute,
			Maghrib: time.Duration(profile.Adjustments.Maghrib) * minute,
			Isha:    time.Duration(profile.Adjustments.Isha) * minute,
		},
	}
	if profile.Madhab == domain.MadhabHanafi {
		cfg.AsrConvention = prayer.Hanafi
	}
	return cfg
}

func convention(method domain.Method) *prayer.TwilightConvention {
	switch method {
	case domain.MethodEgyptian:
		return prayer.Egypt()
	case domain.MethodUmmAlQura:
		return prayer.UmmAlQura()
	case domain.MethodKarachi:
		return prayer.Karachi()
	case domain.MethodISNA:
		return prayer.ISNA()
	case domain.MethodDiyanet:
		return prayer.Diyanet()
	case domain.MethodKemenag:
		return prayer.Kemenag()
	case domain.MethodMUIS:
		return prayer.MUIS()
	case domain.MethodJAKIM:
		return prayer.JAKIM()
	default:
		return prayer.MWL()
	}
}

func highLatitudeAdapter(rule domain.HighLatitudeRule) prayer.HighLatitudeAdapter {
	switch rule {
	case domain.HighLatitudeMiddleNight:
		return prayer.MiddleNight()
	case domain.HighLatitudeSeventhNight:
		return prayer.OneSeventhNight()
	default:
		return prayer.AngleBased()
	}
}

func cacheKey(profile domain.PrayerProfile, year int) string {
	return fmt.Sprintf("%.3f|%.3f|%s|%s|%s|%s|%+v|%d",
		profile.Latitude, profile.Longitude, profile.Timezone, profile.Method,
		profile.Madhab, profile.HighLatitudeRule, profile.Adjustments, year)
}
