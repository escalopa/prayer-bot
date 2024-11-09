package scheduler

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	app "github.com/escalopa/gopray/telegram/internal/application"
)

// Scheduler is responsible for notifying subscribers about the upcoming prayer.
// It also notifies subscribers when the prayer has started.
type Scheduler struct {
	ur  time.Duration // upcoming reminder in minutes
	jr  time.Duration // gomaa notify hour in hours
	loc *time.Location
	pr  app.PrayerRepository
	sr  app.SubscriberRepository

	once sync.Once
}

const (
	defaultTimeFormat = "2006-01-02 15:04:05"
	errSleepDuration  = 30 * time.Second
)

func New(
	upcomingReminder time.Duration,
	jummahReminder time.Duration,
	location *time.Location,
	prayerRepository app.PrayerRepository,
	subscriberRepository app.SubscriberRepository,
) *Scheduler {
	return &Scheduler{
		ur:  upcomingReminder,
		jr:  jummahReminder,
		loc: location,
		pr:  prayerRepository,
		sr:  subscriberRepository,
	}
}

func (s *Scheduler) Run(ctx context.Context, notifier app.Notifier) {
	s.once.Do(func() {
		go s.notifyPrayers(ctx, notifier.PrayerSoon, notifier.PrayerNow)
		go s.notifyJummah(ctx, notifier.PrayerJummah)
	})
}

func (s *Scheduler) notifyPrayers(
	ctx context.Context,
	notifySoon func(ctx context.Context, chatIDs []int, prayer string, time string),
	notifyStart func(ctx context.Context, chatIDs []int, prayer string),
) {
	for {
		prayerName, prayerAfter, err := s.getClosestPrayer(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get closest prayer: %s", err)
			sleep(errSleepDuration)
			continue
		}

		upcomingAt, startsAt := s.timeLeft(prayerAfter)
		// logs for debugging
		log.Printf("Prayer: %s | upcoming: %s(%d) | starts: %s(%d)\n", prayerName,
			s.now().Add(upcomingAt).Format(defaultTimeFormat), int(upcomingAt.Minutes()), // upcoming status
			s.now().Add(startsAt).Add(upcomingAt).Format(defaultTimeFormat), int((startsAt + upcomingAt).Minutes()), // start status
		)

		////////////////////////////////////////////////////////////////
		/// Notify subscribers about the upcoming prayer.
		////////////////////////////////////////////////////////////////
		sleep(upcomingAt)

		chatIDs, err := s.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get subscribers: %s", err)
			sleep(errSleepDuration)
			continue
		}
		notifySoon(ctx, chatIDs, prayerName, strconv.Itoa(int(startsAt.Minutes())))

		////////////////////////////////////////////////////////////////
		/// Notify subscribers when the prayer has started.
		////////////////////////////////////////////////////////////////
		sleep(startsAt)

		chatIDs, err = s.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get subscribers, %s", err)
			sleep(errSleepDuration)
			continue
		}
		notifyStart(ctx, chatIDs, prayerName)
	}
}

func (s *Scheduler) notifyJummah(ctx context.Context, notifyGomaa func(context.Context, []int, string)) {
	for {
		jummah := s.getClosestJummah()
		log.Printf("Gomaa: %s", jummah.Format(defaultTimeFormat))

		// Wait until the jummah is about to start
		sleep(jummah.Sub(s.now()))

		// Get the subscribers
		ids, err := s.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyGomma: failed to get subscribers: %s", err)
			sleep(errSleepDuration)
			continue
		}

		// Get the prayer time for the jummah
		prayers, err := s.pr.GetPrayer(ctx, jummah)
		if err != nil {
			log.Printf("notifyGomma: failed to get prayers for jummah: %s", err)
			sleep(errSleepDuration)
			continue
		}
		// Notify the subscribers
		notifyGomaa(ctx, ids, prayers.Dhuhr.Format("15:04"))
	}
}

// getClosestPrayer returns the name of the closest prayer and the time left until it starts
// @return prayerName string - the name of the closest prayer
// @return prayerAfter int - the time left in minutes until the closest prayer starts
// @return err error - any error that might have occurred
func (s *Scheduler) getClosestPrayer(ctx context.Context) (prayerName string, prayerTime time.Time, err error) {
	// Get the prayer times for today.
	now := s.now()
	p, err := s.pr.GetPrayer(ctx, now)
	if err != nil {
		return "", time.Time{}, err
	}
	// Get the closest prayer.
	// To get time left until the prayer starts, we subtract the current time from the prayer time
	// and convert the result to minutes.
	if p.Fajr.After(now) {
		return "Fajr", p.Fajr, nil
	} else if p.Dohaa.After(now) {
		return "Dohaa", p.Dohaa, nil
	} else if p.Dhuhr.After(now) {
		return "Dhuhr", p.Dhuhr, nil
	} else if p.Asr.After(now) {
		return "Asr", p.Asr, nil
	} else if p.Maghrib.After(now) {
		return "Maghrib", p.Maghrib, nil
	} else if p.Isha.After(now) {
		return "Isha", p.Isha, nil
	}

	// If reach this block, it means that the current time is after Isha.
	// Get the first prayer time for the next day(Fajr).
	tomorrow := now.AddDate(0, 0, 1)
	p, err = s.pr.GetPrayer(ctx, tomorrow)
	if err != nil {
		return "", time.Time{}, err
	}
	return "Fajr", p.Fajr, nil
}

// getClosestJummah returns the time of the next gomaa
// @return day time.Time - the time of the next gomaa
func (s *Scheduler) getClosestJummah() time.Time {
	// `day` defaults to tomorrow. So that if the current date is Friday,
	// we add 1 day to get the next Friday (the next gomaa).
	day := s.now().AddDate(0, 0, 1)
	for day.Weekday() != time.Friday {
		day = day.AddDate(0, 0, 1)
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), int(s.jr.Hours()), 0, 0, 0, s.loc)
	return day
}

// timeLeft calculates the time left until the prayer starts
func (s *Scheduler) timeLeft(prayerTime time.Time) (upcomingAt, startsAt time.Duration) {
	left := prayerTime.Sub(s.now())

	// if `left` is less than the `UPCOMING_REMINDER`, then set `upcomingAt` to 0 & `startsAt` to the `left`
	// else, set the `upcomingAt` to `left` - `UPCOMING_REMINDER`, and `startsAt` to `UPCOMING_REMINDER`
	if left < s.ur {
		upcomingAt = 0
		startsAt = left
	} else {
		upcomingAt = left - s.ur
		startsAt = s.ur
	}

	return
}

func (s *Scheduler) now() time.Time {
	return time.Now().In(s.loc)
}

func sleep(d time.Duration) {
	time.Sleep(d)
}
