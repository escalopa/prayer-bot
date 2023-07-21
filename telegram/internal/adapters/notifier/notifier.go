package notifier

import (
	"context"
	"log"
	"strconv"
	"time"

	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/pkg/errors"
)

// Notifier is responsible for notifying subscribers about the upcoming prayer.
// It also notifies subscribers when the prayer has started.
type Notifier struct {
	pr  app.PrayerRepository
	sr  app.SubscriberRepository
	lr  app.LanguageRepository
	ur  time.Duration // upcoming reminder in minutes
	gnh time.Duration // gomaa notify hour in hours
	loc *time.Location
}

const (
	defaultTimeFormat = "2006-01-02 15:04:05"
	sleepDuration     = 30 * time.Second
)

// New creates a new Notifier.
// @param upcomingReminder is the number of minutes before the prayer starts to notify subscribers.
// @param gomaaNotifyHour is the hour of the day to notify subscribers about the gomaa prayer.
func New(upcomingReminder, gomaaNotifyHour time.Duration, opts ...func(*Notifier)) (*Notifier, error) {
	n := &Notifier{ur: upcomingReminder, gnh: gomaaNotifyHour}
	for _, opt := range opts {
		opt(n)
	}
	if n.ur.Minutes() <= 0 || n.ur.Minutes() >= 60 {
		return nil, errors.New("UPCOMING_REMINDER must be between 1 and 59")
	}
	if n.gnh.Hours() <= 0 || n.gnh.Hours() >= 12 {
		return nil, errors.New("GOMAA_NOTIFY_HOUR must be between 0 and 11")
	}
	if n.loc == nil {
		return nil, errors.New("location is nil")
	}
	if n.pr == nil {
		return nil, errors.New("prayer repository is nil")
	}
	if n.sr == nil {
		return nil, errors.New("subscriber repository is nil")
	}
	if n.lr == nil {
		return nil, errors.New("language repository is nil")
	}
	return n, nil
}

func WithPrayerRepository(pr app.PrayerRepository) func(*Notifier) {
	return func(n *Notifier) {
		n.pr = pr
	}
}

func WithSubscriberRepository(sr app.SubscriberRepository) func(*Notifier) {
	return func(n *Notifier) {
		n.sr = sr
	}
}

func WithLanguageRepository(lr app.LanguageRepository) func(*Notifier) {
	return func(n *Notifier) {
		n.lr = lr
	}
}

func WithTimeLocation(loc *time.Location) func(*Notifier) {
	return func(n *Notifier) {
		n.loc = loc
	}
}

func (n *Notifier) NotifyPrayers(ctx context.Context, notifySoon func([]int, string, string), notifyStart func([]int, string)) {
	// Note that ticker is reset every time a prayer is about to start or has started.
	// So we can create a single ticker with any value and reuse it
	tick := time.NewTicker(time.Hour)
	for {
		prayerName, prayerAfter, err := n.getClosestPrayer(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get closest prayer: %s", err)
			time.Sleep(sleepDuration)
			continue
		}

		upcomingAt, startsAt := n.timeLeft(prayerAfter)
		// logs for debugging
		log.Printf("Prayer: %s | upcoming: %s(%d) | starts: %s(%d)\n", prayerName,
			n.now().Add(upcomingAt).Format(defaultTimeFormat), int(upcomingAt.Minutes()), // upcoming status
			n.now().Add(startsAt).Add(upcomingAt).Format(defaultTimeFormat), int((startsAt + upcomingAt).Minutes()), // start status
		)

		////////////////////////////////////////////////////////////////
		/// Notify subscribers about the upcoming prayer.
		////////////////////////////////////////////////////////////////

		// Wait until the prayer is about to start, & notify subscribers about the upcoming prayer.
		if upcomingAt > 0 { // if upcomingAt is 0, then the prayer is about to start in `startAt` minutes.
			tick.Reset(upcomingAt)
			<-tick.C
		}
		ids, err := n.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get subscribers: %s", err)
			time.Sleep(sleepDuration)
			continue
		}
		notifySoon(ids, prayerName, strconv.Itoa(int(startsAt.Minutes())))

		////////////////////////////////////////////////////////////////
		/// Notify subscribers when the prayer has started.
		////////////////////////////////////////////////////////////////

		tick.Reset(startsAt)
		<-tick.C
		// Get the subscribers
		ids, err = n.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyPrayer: failed to get subscribers, %s", err)
			time.Sleep(sleepDuration)
			continue
		}
		notifyStart(ids, prayerName)
	}
}

func (n *Notifier) NotifyGomaa(ctx context.Context, notifyGomaa func([]int, string)) {
	// Note that ticker is reset every time a prayer is about to start or has started.
	// So we can create a single ticker with any value and reuse it
	tick := time.NewTicker(1 * time.Hour)
	for {
		gomaa := n.getClosestGomaa()
		log.Printf("Gomaa: %s", gomaa.Format(defaultTimeFormat))
		// Wait until the gomaa is about to start
		wait := gomaa.Sub(n.now())
		tick.Reset(wait)
		<-tick.C
		// Get the subscribers
		ids, err := n.sr.GetSubscribers(ctx)
		if err != nil {
			log.Printf("notifyGomma: failed to get subscribers: %s", err)
			time.Sleep(sleepDuration)
			continue
		}
		// Get the prayer time for the gomaa
		prayers, err := n.pr.GetPrayer(ctx, gomaa.Day(), int(gomaa.Month()))
		if err != nil {
			log.Printf("notifyGomma: failed to get prayers for gomaa: %s", err)
			time.Sleep(sleepDuration)
			continue
		}
		// Notify the subscribers
		notifyGomaa(ids, prayers.Dhuhr.Format("15:04"))
	}
}

// getClosestPrayer returns the name of the closest prayer and the time left until it starts
// @return prayerName string - the name of the closest prayer
// @return prayerAfter int - the time left in minutes until the closest prayer starts
// @return err error - any error that might have occurred
func (n *Notifier) getClosestPrayer(ctx context.Context) (prayerName string, prayerTime time.Time, err error) {
	// Get the prayer times for today.
	now := n.now()
	p, err := n.pr.GetPrayer(ctx, now.Day(), int(now.Month()))
	if err != nil {
		return "", time.Time{}, err
	}
	// Get the closest prayer.
	// To get time left until the prayer starts, we subtract the current time from the prayer time
	// and convert the result to minutes.
	if p.Fajr.After(now) {
		return "Fajr", p.Fajr, nil
	} else if p.Sunrise.After(now) {
		return "Sunrise", p.Sunrise, nil
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
	p, err = n.pr.GetPrayer(ctx, tomorrow.Day(), int(tomorrow.Month()))
	if err != nil {
		return "", time.Time{}, err
	}
	return "Fajr", p.Fajr, nil
}

// getClosestGomaa returns the time of the next gomaa
// @return day time.Time - the time of the next gomaa
func (n *Notifier) getClosestGomaa() time.Time {
	// `day` defaults to tomorrow. So that if the current date is Friday, we add 1 day to get the next Friday (the next gomaa).
	day := n.now().AddDate(0, 0, 1)
	for day.Weekday() != time.Friday {
		day = day.AddDate(0, 0, 1)
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), int(n.gnh.Hours()), 0, 0, 0, n.loc)
	return day
}

// timeLeft calculates the time left until the prayer starts
// @param t time.Time - the prayer time
// @return upcomingAt time.Duration - the time left in minutes until the prayer starts subtracted from the user reminder time `UPCOMING_REMINDER`
// if the time left is less than the reminder time, then the `upcomingAt` is 0
// @return startsAt time.Duration - the time to wait in minutes until the prayer starts, after the upcoming reminder has passed
func (n *Notifier) timeLeft(t time.Time) (upcomingAt, startsAt time.Duration) {
	left := t.Sub(n.now())
	// if `left` is less than the `UPCOMING_REMINDER`, then set `upcomingAt` to 0 & `startsAt` to the `left`
	// else, set the `upcomingAt` to `left` - `UPCOMING_REMINDER`, and `startsAt` to `UPCOMING_REMINDER`
	if left < n.ur {
		upcomingAt = 0
		startsAt = left
	} else {
		upcomingAt = left - n.ur
		startsAt = n.ur
	}
	return
}

// now returns the current time in the notifier's location.
func (n *Notifier) now() time.Time {
	now := time.Now().In(n.loc)
	return now
}
