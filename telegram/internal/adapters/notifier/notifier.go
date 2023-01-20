package notifier

import (
	"fmt"
	"log"
	"time"

	"github.com/escalopa/gopray/pkg/prayer"
	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/pkg/errors"
)

// Notifier is responsible for notifying subscribers about the upcoming prayer.
// It also notifies subscribers when the prayer has started.
type Notifier struct {
	pr  app.PrayerRepository
	sr  app.SubscriberRepository
	lr  app.LanguageRepository
	ur  int // upcoming reminder in minutes
	gnh int // gomaa notify hour
	loc *time.Location
}

// New creates a new Notifier.
// @param upcomingReminder is the number of minutes before the prayer starts to notify subscribers.
// @param gomaaNotifyHour is the hour of the day to notify subscribers about the gomaa prayer.
// @return a new Notifier.
// @return an error if the upcomingReminder is not between 1 and 60 or gomaaNotifyHour is not between 0 and 12 or if the location is not found.
func New(pr app.PrayerRepository, sr app.SubscriberRepository, lr app.LanguageRepository, upcomingReminder, gomaaNotifyHour int) (*Notifier, error) {
	n := &Notifier{
		pr:  pr,
		sr:  sr,
		lr:  lr,
		ur:  upcomingReminder,
		gnh: gomaaNotifyHour,
	}
	if n.ur <= 0 || n.ur >= 60 {
		return nil, errors.New("UPCOMING_REMINDER must be between 1 and 60")
	}
	if n.gnh <= 0 || n.gnh >= 12 {
		return nil, errors.New("GOMAA_NOTIFY_HOUR must be between 0 and 12")
	}
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load location")
	}
	n.loc = loc
	return n, nil
}

func (n *Notifier) NotifyPrayers(notify func(ids []int, msg string)) error {
	// Note that ticker is reset every time a prayer is about to start or has started.
	// So we can create a single ticker with any value and reuse it
	tick := time.NewTicker(time.Hour)
	for {
		prayerName, prayerAfter, err := n.getClosestPrayer()
		if err != nil {
			return errors.Wrap(err, "Failed to get closest prayer")
		}

		upcomingAt, startsAt, startsIn := n.calculateLeftTime(prayerAfter)
		// logs for debugging
		log.Printf("Prayer: %s,upcomingAt:%f,startsAt: %f,startsIn: %d", prayerName, upcomingAt.Minutes(), startsAt.Minutes(), startsIn)

		// Wait until the prayer is about to start, & notify subscribers about the upcoming prayer.
		if upcomingAt > 0 {
			tick.Reset(upcomingAt)
			<-tick.C
		}
		ids, err := n.sr.GetSubscribers()
		if err != nil {
			return errors.Wrap(err, "Failed to get subscribers")
		}
		notify(ids, fmt.Sprintf("<b>%s</b> prayer is about to start in <b>%d</b> minutes.", prayerName, startsIn))

		// Wait until the prayer starts & notify subscribers that the prayer has started.
		if startsAt > 0 {
			tick.Reset(startsAt)
			<-tick.C
		}
		ids, err = n.sr.GetSubscribers()
		if err != nil {
			return errors.Wrap(err, "Failed to get subscribers")
		}
		notify(ids, fmt.Sprintf("<b>%s</b> prayer time has arrived.", prayerName))
	}
}

func (n *Notifier) NotifyGomaa(notify func(ids []int, msg string)) error {
	// Note that ticker is reset every time a prayer is about to start or has started.
	// So we can create a single ticker with any value and reuse it
	tick := time.NewTicker(1 * time.Hour)
	for {
		gomaa, err := n.getClosestGomaa()
		if err != nil {
			return errors.Wrap(err, "Failed to get closest gomaa")
		}

		log.Printf("Gomaa: %s", gomaa.String())

		// Get the current time
		now, err := n.now()
		if err != nil {
			return errors.Wrap(err, "Failed to get current time")
		}

		// Wait until the gomaa is about to start, & notify subscribers about the upcoming gomaa.
		wait := gomaa.Sub(now)
		tick.Reset(wait)
		<-tick.C

		// Get the subscribers
		ids, err := n.sr.GetSubscribers()
		if err != nil {
			return errors.Wrap(err, "Failed to get subscribers")
		}

		// Get the prayer time for the gomaa
		prayers, err := n.pr.GetPrayer(gomaa.Day(), int(gomaa.Month()))
		if err != nil {
			return errors.Wrap(err, "Failed to get prayers for gomaa")
		}

		// Notify the subscribers
		message := fmt.Sprintf(
			"Assalamu Alaikum 👋!\nDon't forget today is <b>Gomaa</b> , make sure to attend prayers at the mosque! 🕌, Gomma today is at <b>%s</b>",
			prayers.Dhuhr.Format("15:04"))
		notify(ids, message)
	}
}

// Subscribe adds the subscriber with the given id to the list of subscribers.
// @param id int - the id of the subscriber to add
// @return error - any error that might have occurred
func (n *Notifier) Subscribe(id int) error {
	return n.sr.StoreSubscriber(id)
}

// Unsubscribe removes the subscriber with the given id from the list of subscribers.
// @param id int - the id of the subscriber to remove
// @return error - any error that might have occurred
func (n *Notifier) Unsubscribe(id int) error {
	return n.sr.RemoveSubscribe(id)
}

// getClosestPrayer returns the name of the closest prayer and the time left until it starts
// @return prayerName string - the name of the closest prayer
// @return prayerAfter int - the time left in minutes until the closest prayer starts
// @return err error - any error that might have occurred
func (n *Notifier) getClosestPrayer() (prayerName string, prayerTime time.Time, err error) {
	// Get the prayer times for today.
	p, err := n.getPrayerTime(time.Now())
	if err != nil {
		return "", time.Time{}, err
	}

	now, err := n.now()
	if err != nil {
		return "", time.Time{}, err
	}
	// Get the current time.
	// Time is in UTC, so we need to convert it to the local time in Kazan Russia("Europe/Moscow").

	// Get the closest prayer.
	// To get time left until the prayer starts, we subtract the current time from the prayer time
	// and convert the result to minutes.
	if p.Fajr.After(now) {
		return "Fajr", p.Fajr, nil
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
	tomorrow := time.Now().Add(time.Hour * 24)
	p, err = n.getPrayerTime(tomorrow)
	if err != nil {
		return "", time.Time{}, err
	}
	return "Fajr", p.Fajr, nil
}

func (n *Notifier) getClosestGomaa() (gomaa time.Time, err error) {
	now, err := n.now()
	if err != nil {
		return time.Time{}, err
	}

	friday := now
	// Get the next Friday.
	for {
		friday = friday.AddDate(0, 0, 1)
		if friday.Weekday() == time.Friday {
			break
		}
	}

	friday = time.Date(friday.Year(), friday.Month(), friday.Day(), n.gnh, 0, 0, 0, n.loc)
	return friday, nil
}

// getPrayerTime returns the prayer times for the given date.
// @param t time.Time - the date for which to get the prayer times
// @return prayer.PrayerTimes - the prayer times for the given date
// @return error - any error that might have occurred
func (n *Notifier) getPrayerTime(t time.Time) (prayer.PrayerTimes, error) {
	p, err := n.pr.GetPrayer(t.Day(), int(t.Month()))
	if err != nil {
		return prayer.PrayerTimes{}, err
	}
	return p, nil
}

// calculateLeftTime calculates the time left until the prayer starts
// @param t time.Time - the prayer time
// @return upcomingAt time.Duration - the time left in minutes until the prayer starts subtracted from the user reminder time `UPCOMING_REMINDER`
// @return startsIn time.Duration - the time left in minutes until the prayer starts after upcoming reminder has passed
// @return startsAt time.Duration - the time to wait in minutes until the prayer starts, after the upcoming reminder has passed
// Returns usage flow, `upcomingAt` >> `startsIn` >> `startsAt`
// The difference between `startsIn` and `startsAt` is that `startsIn` is `int` representing the time left in minutes
// while `startsAt` is `time.Duration.Minute()`  which is the time left in minutes as `float64` or more precisely in `nanoseconds`.
func (n *Notifier) calculateLeftTime(t time.Time) (upcomingAt, startsAt time.Duration, startsIn int) {
	// Get the current time, Error here is omitted because it's already handled in the `getClosestPrayer` function.
	// And it's not possible to get an error on this call, since the previous call to `now()` was successful.
	now, _ := n.now()

	// if time left until the prayer starts is less than the `UPCOMING_REMINDER`, then set the `upcomingAt` to 0
	// and set the `startsAt` to the `UPCOMING_REMINDER`
	// else, set the `upcomingAt` to the time left until the prayer starts subtracted from the `UPCOMING_REMINDER`
	left := int(t.Sub(now).Minutes())

	////////////////////////
	// NOTE: StartAt is increased by 1 minute to avoid sending the notification twice or many times.
	////////////////////////

	// The prayers start in time less than the `UPCOMING_REMINDER`.
	if left < n.ur {
		upcomingAt = 0
		startsIn = left
		startsAt = time.Duration(left + 1)
	} else {
		upcomingAt = time.Duration((left - n.ur))
		startsIn = n.ur
		startsAt = time.Duration(n.ur + 1)
	}
	upcomingAt, startsAt = upcomingAt*time.Minute, startsAt*time.Minute
	return
}

func (n *Notifier) now() (time.Time, error) {
	now := time.Now().In(n.loc)
	return now, nil
}
