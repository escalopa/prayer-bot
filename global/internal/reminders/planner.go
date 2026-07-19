package reminders

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/escalopa/prayer-bot/global/internal/domain"
	"github.com/escalopa/prayer-bot/global/internal/occasions"
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
	if rule.Kind.Weekly() {
		return nextWeekly(profile, rule, after, location)
	}
	if rule.Kind.Occasion() {
		return nextOccasion(profile, rule, after, location)
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

func nextOccasion(profile domain.PrayerProfile, rule domain.ReminderRule, after time.Time, location *time.Location) (domain.ReminderSchedule, error) {
	category, ok := occasionCategory(rule.Kind)
	if !ok {
		return domain.ReminderSchedule{}, fmt.Errorf("unsupported occasion reminder kind %q", rule.Kind)
	}
	hour, minute, err := parseLocalTime(rule.LocalTime)
	if err != nil {
		return domain.ReminderSchedule{}, err
	}
	localAfter := after.In(location)
	upcoming, err := occasions.Between(localAfter, 400, profile.HijriAdjustment)
	if err != nil {
		return domain.ReminderSchedule{}, err
	}
	for _, occurrence := range upcoming {
		if occurrence.Definition.Category != category {
			continue
		}
		target := time.Date(
			occurrence.Date.Year(), occurrence.Date.Month(), occurrence.Date.Day(),
			0, 0, 0, 0, location,
		)
		previous := target.AddDate(0, 0, -1)
		nextRun := time.Date(previous.Year(), previous.Month(), previous.Day(), hour, minute, 0, 0, location)
		if !nextRun.After(after) {
			continue
		}
		return domain.ReminderSchedule{
			RuleID: rule.ID, ChatID: rule.ChatID, ProfileVersion: profile.Version,
			LocalDate: target.Format("2006-01-02"), PrayerAt: target,
			NextRunAt: nextRun.UTC(), State: "pending",
		}, nil
	}
	return domain.ReminderSchedule{}, fmt.Errorf("no valid occasion found in the next 400 days")
}

func occasionCategory(kind domain.ReminderKind) (occasions.Category, bool) {
	switch kind {
	case domain.ReminderOccasionMajor:
		return occasions.CategoryMajor, true
	case domain.ReminderOccasionFasting:
		return occasions.CategoryFasting, true
	case domain.ReminderOccasionObserved:
		return occasions.CategoryObserved, true
	default:
		return "", false
	}
}

func nextWeekly(profile domain.PrayerProfile, rule domain.ReminderRule, after time.Time, location *time.Location) (domain.ReminderSchedule, error) {
	hour, minute, err := parseLocalTime(rule.LocalTime)
	if err != nil {
		return domain.ReminderSchedule{}, err
	}
	localAfter := after.In(location)
	for dayOffset := 0; dayOffset < 15; dayOffset++ {
		candidate := localAfter.AddDate(0, 0, dayOffset)
		target := time.Date(candidate.Year(), candidate.Month(), candidate.Day(), 0, 0, 0, 0, location)
		var nextRun time.Time
		switch rule.Kind {
		case domain.ReminderWeeklyFasting:
			if target.Weekday() != time.Monday && target.Weekday() != time.Thursday {
				continue
			}
			previous := target.AddDate(0, 0, -1)
			nextRun = time.Date(previous.Year(), previous.Month(), previous.Day(), hour, minute, 0, 0, location)
		case domain.ReminderWeeklyKahf:
			if target.Weekday() != time.Friday {
				continue
			}
			nextRun = time.Date(target.Year(), target.Month(), target.Day(), hour, minute, 0, 0, location)
		default:
			return domain.ReminderSchedule{}, fmt.Errorf("unsupported weekly reminder kind %q", rule.Kind)
		}
		if !nextRun.After(after) {
			continue
		}
		prayerAt := target
		if rule.Kind == domain.ReminderWeeklyKahf {
			prayerAt = nextRun
		}
		return domain.ReminderSchedule{
			RuleID: rule.ID, ChatID: rule.ChatID, ProfileVersion: profile.Version,
			LocalDate: target.Format("2006-01-02"), PrayerAt: prayerAt,
			NextRunAt: nextRun.UTC(), State: "pending",
		}, nil
	}
	return domain.ReminderSchedule{}, fmt.Errorf("no valid weekly occurrence found in the next fifteen days")
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
