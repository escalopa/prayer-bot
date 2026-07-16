package reminders

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/prayertime"
)

type PlanningStore interface {
	Profile(context.Context, int64) (domain.PrayerProfile, error)
	EnabledRules(context.Context, int64) ([]domain.ReminderRule, error)
	UpsertSchedule(context.Context, domain.ReminderSchedule) (domain.ReminderSchedule, error)
}

type Planner struct {
	store      PlanningStore
	calculator prayertime.Calculator
}

func NewPlanner(store PlanningStore, calculator prayertime.Calculator) *Planner {
	return &Planner{store: store, calculator: calculator}
}

func (p *Planner) RebuildChat(ctx context.Context, chatID int64, after time.Time) error {
	profile, err := p.store.Profile(ctx, chatID)
	if err != nil {
		return fmt.Errorf("load profile: %w", err)
	}
	rules, err := p.store.EnabledRules(ctx, chatID)
	if err != nil {
		return fmt.Errorf("load reminder rules: %w", err)
	}
	for _, rule := range rules {
		next, err := p.Next(ctx, profile, rule, after)
		if err != nil {
			return fmt.Errorf("plan rule %d: %w", rule.ID, err)
		}
		if _, err := p.store.UpsertSchedule(ctx, next); err != nil {
			return fmt.Errorf("save rule %d schedule: %w", rule.ID, err)
		}
	}
	return nil
}

func (p *Planner) Next(ctx context.Context, profile domain.PrayerProfile, rule domain.ReminderRule, after time.Time) (domain.ReminderSchedule, error) {
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return domain.ReminderSchedule{}, err
	}
	localAfter := after.In(location)
	for dayOffset := 0; dayOffset < 8; dayOffset++ {
		date := localAfter.AddDate(0, 0, dayOffset)
		schedule, err := p.calculator.Day(ctx, date, profile)
		if err != nil {
			return domain.ReminderSchedule{}, err
		}
		prayerAt, ok := schedule.At(rule.Prayer)
		if !ok {
			continue
		}

		nextRun := prayerAt.Add(-time.Duration(rule.OffsetMinutes) * time.Minute)
		if rule.Kind == domain.ReminderTomorrow {
			hour, minute, err := parseLocalTime(rule.LocalTime)
			if err != nil {
				return domain.ReminderSchedule{}, err
			}
			previousDay := prayerAt.In(location).AddDate(0, 0, -1)
			nextRun = time.Date(previousDay.Year(), previousDay.Month(), previousDay.Day(), hour, minute, 0, 0, location)
		}
		if !nextRun.After(after) {
			continue
		}
		return domain.ReminderSchedule{
			RuleID:         rule.ID,
			ChatID:         rule.ChatID,
			ProfileVersion: profile.Version,
			LocalDate:      prayerAt.In(location).Format("2006-01-02"),
			PrayerAt:       prayerAt,
			NextRunAt:      nextRun.UTC(),
			State:          "pending",
		}, nil
	}
	return domain.ReminderSchedule{}, fmt.Errorf("no valid occurrence found in the next eight days")
}

func parseLocalTime(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid local time %q", value)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid local time %q", value)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid local time %q", value)
	}
	return hour, minute, nil
}
