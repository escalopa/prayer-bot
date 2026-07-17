package calendarfile

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/i18n"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
)

const eventDuration = 15 * time.Minute

var calendarPrayers = []domain.Prayer{
	domain.PrayerFajr,
	domain.PrayerSunrise,
	domain.PrayerDhuhr,
	domain.PrayerAsr,
	domain.PrayerMaghrib,
	domain.PrayerIsha,
}

func Generate(
	ctx context.Context,
	calculator prayertime.Calculator,
	profile domain.PrayerProfile,
	locale i18n.Locale,
	start time.Time,
	days int,
	createdAt time.Time,
	uidNamespace string,
) ([]byte, error) {
	if days < 1 || days > 31 {
		return nil, fmt.Errorf("calendar range must be between 1 and 31 days")
	}
	if !validUIDNamespace(uidNamespace) {
		return nil, fmt.Errorf("invalid calendar UID namespace")
	}
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	start = start.In(location)

	var calendar bytes.Buffer
	writeLine(&calendar, "BEGIN:VCALENDAR")
	writeLine(&calendar, "VERSION:2.0")
	writeLine(&calendar, "PRODID:-//Global Prayer Times//Prayer Calendar//EN")
	writeLine(&calendar, "CALSCALE:GREGORIAN")
	writeLine(&calendar, "METHOD:PUBLISH")
	writeLine(&calendar, "X-WR-CALNAME:"+escapeText(locale.BotName))
	writeLine(&calendar, "X-WR-TIMEZONE:"+escapeText(profile.Timezone))
	writeLine(&calendar, "X-PUBLISHED-TTL:PT12H")
	writeLine(&calendar, "REFRESH-INTERVAL;VALUE=DURATION:PT12H")

	for day := 0; day < days; day++ {
		schedule, err := calculator.Day(ctx, start.AddDate(0, 0, day), profile)
		if err != nil {
			return nil, fmt.Errorf("calculate day %d: %w", day+1, err)
		}
		for _, prayer := range calendarPrayers {
			at, ok := schedule.At(prayer)
			if !ok {
				continue
			}
			writeEvent(&calendar, profile, locale, prayer, at, createdAt, uidNamespace)
		}
	}
	writeLine(&calendar, "END:VCALENDAR")
	return calendar.Bytes(), nil
}

func writeEvent(
	calendar *bytes.Buffer,
	profile domain.PrayerProfile,
	locale i18n.Locale,
	prayer domain.Prayer,
	at time.Time,
	createdAt time.Time,
	uidNamespace string,
) {
	description := fmt.Sprintf("%s · %s · %s", locale.BotName, locale.Method(profile.Method), profile.Timezone)
	uid := fmt.Sprintf(
		"%s-%s-%s@global-prayer-bot",
		uidNamespace,
		at.In(mustLocation(profile.Timezone)).Format("20060102"),
		prayer,
	)

	writeLine(calendar, "BEGIN:VEVENT")
	writeLine(calendar, "UID:"+uid)
	writeLine(calendar, "DTSTAMP:"+createdAt.UTC().Format("20060102T150405Z"))
	writeLine(calendar, "DTSTART:"+at.UTC().Format("20060102T150405Z"))
	writeLine(calendar, "DTEND:"+at.Add(eventDuration).UTC().Format("20060102T150405Z"))
	writeLine(calendar, "SUMMARY:"+escapeText(locale.Prayer(prayer)))
	writeLine(calendar, "DESCRIPTION:"+escapeText(description))
	writeLine(calendar, "CATEGORIES:Prayer Times")
	writeLine(calendar, "END:VEVENT")
}

func validUIDNamespace(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func mustLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return location
}

func escapeText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\r\n", `\n`)
	value = strings.ReplaceAll(value, "\r", `\n`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, ";", `\;`)
	return strings.ReplaceAll(value, ",", `\,`)
}

// RFC 5545 content lines are limited to 75 octets. Continuation lines begin
// with a single space, so they contain at most 74 octets of content.
func writeLine(target *bytes.Buffer, line string) {
	remaining := line
	limit := 75
	for len(remaining) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(remaining[cut]) {
			cut--
		}
		target.WriteString(remaining[:cut])
		target.WriteString("\r\n ")
		remaining = remaining[cut:]
		limit = 74
	}
	target.WriteString(remaining)
	target.WriteString("\r\n")
}
