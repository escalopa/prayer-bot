package notifier

import (
	"fmt"
	"time"

	"github.com/escalopa/gopray/pkg/prayer"
	app "github.com/escalopa/gopray/telegram/internal/application"
	"github.com/pkg/errors"
)

// Notifier is responsible for notifying subscribers about the upcoming prayer.
// It also notifies subscribers when the prayer has started.
type Notifier struct {
	pr app.PrayerRepository
	sr app.SubscriberRepository
	lr app.LanguageRepository
	ur uint
}

func New(pr app.PrayerRepository, sr app.SubscriberRepository, lr app.LanguageRepository, upcomingReminder uint) *Notifier {
	return &Notifier{
		pr: pr,
		sr: sr,
		lr: lr,
		ur: upcomingReminder,
	}
}

func (n *Notifier) Notify(notify func(ids []int, msg string)) error {
	var tick *time.Ticker

	for {
		prayerName, prayerAfter, err := n.getClosestPrayer()
		if err != nil {
			return errors.Wrap(err, "Failed to get closest prayer")
		}

		upcomingAt, startsAt := n.calculateLeftTime(prayerAfter)
		// logs for debugging
		// log.Println("Waiting for", prayerName, "to start in", prayerAfter, "minutes.")
		// log.Println("Notifying subscribers in", upcomingAt, "minutes.")
		// log.Println("Prayer start at ", startsAt, "minutes.")

		// Wait until the prayer is about to start, & notify subscribers about the upcoming prayer.
		tick = time.NewTicker(upcomingAt)
		<-tick.C
		ids, err := n.sr.GetSubscribers()
		if err != nil {
			return errors.Wrap(err, "Failed to get subscribers")
		}
		notify(ids, fmt.Sprintf("<b>%s</b> prayer is about to start in <b>%d</b> minutes.", prayerName, prayerAfter))

		// Wait until the prayer starts & notify subscribers that the prayer has started.
		t2 := time.NewTicker(startsAt)
		<-t2.C
		ids, err = n.sr.GetSubscribers()
		if err != nil {
			return errors.Wrap(err, "Failed to get subscribers")
		}
		notify(ids, fmt.Sprintf("<b>%s</b> prayer time has arrived.", prayerName))
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
func (n *Notifier) getClosestPrayer() (prayerName string, prayerAfter uint, err error) {
	// Get the prayer times for today.
	p, err := n.getPrayerTime(time.Now())
	if err != nil {
		return "", 0, err
	}

	// Get the current time.
	// Time is in UTC, so we need to convert it to the local time in Kazan Russia("Europe/Moscow").
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return "", 0, err
	}
	now := time.Now().In(loc)

	// Get the closest prayer.
	// To get time left until the prayer starts, we subtract the current time from the prayer time
	// and convert the result to minutes.
	if p.Fajr.After(now) {
		return "Fajr", uint(p.Fajr.Sub(now).Minutes()), nil
	} else if p.Dhuhr.After(now) {
		return "Dhuhr", uint(p.Dhuhr.Sub(now).Minutes()), nil
	} else if p.Asr.After(now) {
		return "Asr", uint(p.Asr.Sub(now).Minutes()), nil
	} else if p.Maghrib.After(now) {
		return "Maghrib", uint(p.Maghrib.Sub(now).Minutes()), nil
	} else if p.Isha.After(now) {
		return "Isha", uint(p.Isha.Sub(now).Minutes()), nil
	}

	// If reach this block, it means that the current time is after Isha.
	// Get the first prayer time for the next day(Fajr).
	tomorrow := time.Now().Add(time.Hour * 24)
	p, err = n.getPrayerTime(tomorrow)
	if err != nil {
		return "", 0, err
	}
	return "Fajr", uint(p.Fajr.Sub(now).Minutes()), nil
}

// getPrayerTime returns the prayer times for the given date.
// @param t time.Time - the date for which to get the prayer times
// @return prayer.PrayerTimes - the prayer times for the given date
// @return error - any error that might have occurred
func (n *Notifier) getPrayerTime(t time.Time) (prayer.PrayerTimes, error) {
	// Create the key for the prayer times. in the format of "day/month" without leading zeros.
	key := fmt.Sprintf("%d/%d", t.Day(), t.Month())
	p, err := n.pr.GetPrayer(key)
	if err != nil {
		return prayer.PrayerTimes{}, err
	}
	return p, nil
}

func (n *Notifier) calculateLeftTime(t uint) (upcomingAt, startsAt time.Duration) {
	upcomingAt = time.Duration((t - n.ur))
	// If the prayer is close, wait for 1 minute then notify.
	if upcomingAt <= 0 {
		upcomingAt = 1
	}
	startsAt = time.Duration(t)
	upcomingAt, startsAt = upcomingAt*time.Minute, startsAt*time.Minute
	return
}
