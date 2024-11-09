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
	sleepDuration     = 30 * time.Second
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
	notifySoon func(ctx context.Context, userIDs []int, prayer string, time string),
	notifyStart func(ctx context.Context, userIDs []int, prayer string),
) {
	// Note that ticker is reset every time a prayer is about to start or has started.
	// So we can create a single ticker with any value and reuse it
	tick := time.NewTicker(time.Hour)
	for {
		prayerName, prayerAfter, err := s.getClosestPrayer(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get closest prayer: %s", err)
			time.Sleep(sleepDuration)
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

		// Wait until the prayer is about to start, & notify subscribers about the upcoming prayer.
		// If upcomingAt is 0, then the prayer is about to start in `startAt` minutes.
		if upcomingAt > 0 {
			tick.Reset(upcomingAt)
			<-tick.C
		}

		userIDs, err := s.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get subscribers: %s", err)
			time.Sleep(sleepDuration)
			continue
		}
		notifySoon(ctx, userIDs, prayerName, strconv.Itoa(int(startsAt.Minutes())))

		////////////////////////////////////////////////////////////////
		/// Notify subscribers when the prayer has started.
		////////////////////////////////////////////////////////////////

		tick.Reset(startsAt)
		<-tick.C
		// Get the subscribers
		userIDs, err = s.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get subscribers, %s", err)
			time.Sleep(sleepDuration)
			continue
		}
		notifyStart(ctx, userIDs, prayerName)
	}
}

func (s *Scheduler) notifyJummah(ctx context.Context, notifyGomaa func(context.Context, []int, string)) {
	// Note that ticker is reset every time a prayer is about to start or has started.
	// So we can create a single ticker with any value and reuse it
	tick := time.NewTicker(1 * time.Hour)
	for {
		gomaa := s.getClosestGomaa()
		log.Printf("Gomaa: %s", gomaa.Format(defaultTimeFormat))
		// Wait until the gomaa is about to start
		wait := gomaa.Sub(s.now())
		tick.Reset(wait)
		<-tick.C
		// Get the subscribers
		ids, err := s.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyGomma: failed to get subscribers: %s", err)
			time.Sleep(sleepDuration)
			continue
		}
		// Get the prayer time for the gomaa
		prayers, err := s.pr.GetPrayer(ctx, gomaa)
		if err != nil {
			log.Printf("notifyGomma: failed to get prayers for gomaa: %s", err)
			time.Sleep(sleepDuration)
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

// getClosestGomaa returns the time of the next gomaa
// @return day time.Time - the time of the next gomaa
func (s *Scheduler) getClosestGomaa() time.Time {
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
// @param t time.Time - the prayer time
// @return upcomingAt time.Duration - the time left in minutes until the prayer starts subtracted from the user reminder time `UPCOMING_REMINDER`
// if the time left is less than the reminder time, then the `upcomingAt` is 0
// @return startsAt time.Duration - the time to wait in minutes until the prayer starts, after the upcoming reminder has passed
func (s *Scheduler) timeLeft(t time.Time) (upcomingAt, startsAt time.Duration) {
	left := t.Sub(s.now())
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

// now returns the current time in the notifier's location.
func (s *Scheduler) now() time.Time {
	return time.Now().In(s.loc)
}
