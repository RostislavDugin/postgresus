package intervals

import (
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Interval struct {
	Type IntervalType `json:"type" gorm:"column:interval_type;type:text;not null"`

	TimeOfDay      *string `json:"timeOfDay"                gorm:"column:time_of_day;type:text"`
	Weekday        *int    `json:"weekday,omitempty"        gorm:"column:weekday;type:int"`
	DayOfMonth     *int    `json:"dayOfMonth,omitempty"     gorm:"column:day_of_month;type:int"`
	CronExpression *string `json:"cronExpression,omitempty" gorm:"column:cron_expression;type:text"`
}

func (i *Interval) Validate() error {
	switch i.Type {
	case IntervalHourly, IntervalDaily, IntervalWeekly, IntervalMonthly, IntervalCron:
	default:
		return fmt.Errorf("invalid interval type: %q", i.Type)
	}

	if i.Type == IntervalDaily || i.Type == IntervalWeekly || i.Type == IntervalMonthly {
		if i.TimeOfDay == nil {
			return errors.New("time of day is required for daily, weekly and monthly intervals")
		}

		if _, err := time.Parse("15:04", *i.TimeOfDay); err != nil {
			return fmt.Errorf("invalid time of day: %w", err)
		}
	}

	if i.Type == IntervalWeekly {
		if i.Weekday == nil {
			return errors.New("weekday is required for weekly intervals")
		}

		// 0 and 7 both mean Sunday — see shouldTriggerWeekly for the alias handling
		if *i.Weekday < 0 || *i.Weekday > 7 {
			return errors.New("weekday must be between 0 and 7")
		}
	}

	if i.Type == IntervalMonthly {
		if i.DayOfMonth == nil {
			return errors.New("day of month is required for monthly intervals")
		}

		if *i.DayOfMonth < 1 || *i.DayOfMonth > 31 {
			return errors.New("day of month must be between 1 and 31")
		}
	}

	if i.Type == IntervalCron {
		if i.CronExpression == nil || *i.CronExpression == "" {
			return errors.New("cron expression is required for cron intervals")
		}

		if err := i.validateCronExpression(*i.CronExpression); err != nil {
			return err
		}
	}

	return nil
}

// ShouldTriggerBackup checks if a backup should be triggered based on the interval and last backup time
func (i *Interval) ShouldTriggerBackup(now time.Time, lastBackupTime *time.Time) bool {
	// If no backup has been made yet, trigger immediately
	if lastBackupTime == nil {
		return true
	}

	switch i.Type {
	case IntervalHourly:
		return now.Sub(*lastBackupTime) >= time.Hour
	case IntervalDaily:
		return i.shouldTriggerDaily(now, *lastBackupTime)
	case IntervalWeekly:
		return i.shouldTriggerWeekly(now, *lastBackupTime)
	case IntervalMonthly:
		return i.shouldTriggerMonthly(now, *lastBackupTime)
	case IntervalCron:
		return i.shouldTriggerCron(now, *lastBackupTime)
	default:
		return false
	}
}

// NextTriggerTime computes the next time a backup should trigger based on the interval and last backup time.
// Returns nil when a backup is due immediately (no previous backup exists).
func (i *Interval) NextTriggerTime(now time.Time, lastBackupTime *time.Time) *time.Time {
	if lastBackupTime == nil {
		return nil
	}

	switch i.Type {
	case IntervalHourly:
		next := lastBackupTime.Add(time.Hour)
		return &next

	case IntervalDaily:
		next := i.nextDailyTrigger(now)
		return &next

	case IntervalWeekly:
		next := i.nextWeeklyTrigger(now)
		return &next

	case IntervalMonthly:
		next := i.nextMonthlyTrigger(now)
		return &next

	case IntervalCron:
		return i.nextCronTrigger(*lastBackupTime)

	default:
		return nil
	}
}

// ApproxPeriod returns a coarse cadence for the interval, used only where an
// exact next-trigger is unnecessary (e.g. sizing a bounded wait window as a
// fraction of the cadence). Cron and unknown types return 0 so callers fall
// back to their own cap rather than guessing a period from an arbitrary
// expression.
func (i *Interval) ApproxPeriod() time.Duration {
	switch i.Type {
	case IntervalHourly:
		return time.Hour
	case IntervalDaily:
		return 24 * time.Hour
	case IntervalWeekly:
		return 7 * 24 * time.Hour
	case IntervalMonthly:
		return 30 * 24 * time.Hour
	default:
		return 0
	}
}

func (i *Interval) Copy() Interval {
	return Interval{
		Type:           i.Type,
		TimeOfDay:      i.TimeOfDay,
		Weekday:        i.Weekday,
		DayOfMonth:     i.DayOfMonth,
		CronExpression: i.CronExpression,
	}
}

// daily trigger: honour the TimeOfDay slot and catch up the previous one
func (i *Interval) shouldTriggerDaily(now, lastBackup time.Time) bool {
	if i.TimeOfDay == nil {
		return !isSameDay(lastBackup, now)
	}

	t, err := time.Parse("15:04", *i.TimeOfDay)
	if err != nil {
		return false // malformed ⇒ play safe
	}

	// Today's scheduled slot (todayTgt)
	todayTgt := time.Date(
		now.Year(), now.Month(), now.Day(),
		t.Hour(), t.Minute(), 0, 0, now.Location(),
	)

	// The last scheduled slot that should already have happened
	var lastScheduled time.Time
	if now.Before(todayTgt) {
		lastScheduled = todayTgt.AddDate(0, 0, -1)
	} else {
		lastScheduled = todayTgt
	}

	// Fire when we are past that slot AND no backup has been taken since it
	return (now.After(lastScheduled) || now.Equal(lastScheduled)) &&
		lastBackup.Before(lastScheduled)
}

// weekly trigger: on specified weekday/calendar week, otherwise ≥7 days
func (i *Interval) shouldTriggerWeekly(now, lastBackup time.Time) bool {
	if i.Weekday != nil {
		targetWd := time.Weekday(*i.Weekday)

		// Calculate the target datetime for this week
		startOfWeek := getStartOfWeek(now)

		// Convert Go weekday to days from Monday: Sunday=6, Monday=0, Tuesday=1, ..., Saturday=5
		var daysFromMonday int
		if targetWd == time.Sunday {
			daysFromMonday = 6
		} else {
			daysFromMonday = int(targetWd) - 1
		}

		targetThisWeek := startOfWeek.AddDate(0, 0, daysFromMonday)

		if i.TimeOfDay != nil {
			t, err := time.Parse("15:04", *i.TimeOfDay)
			if err == nil {
				targetThisWeek = time.Date(
					targetThisWeek.Year(),
					targetThisWeek.Month(),
					targetThisWeek.Day(),
					t.Hour(),
					t.Minute(),
					0,
					0,
					targetThisWeek.Location(),
				)
			}
		}

		// If current time is at or after the target time this week
		// and no backup has been made at or after the target time, trigger
		if now.After(targetThisWeek) || now.Equal(targetThisWeek) {
			return lastBackup.Before(targetThisWeek)
		}

		return false
	}

	// no Weekday: generic 7-day interval
	return now.Sub(lastBackup) >= 7*24*time.Hour
}

// monthly trigger: on specified day/calendar month, otherwise next calendar month
func (i *Interval) shouldTriggerMonthly(now, lastBackup time.Time) bool {
	if i.DayOfMonth != nil {
		day := *i.DayOfMonth

		// Calculate the target datetime for this month
		targetThisMonth := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, now.Location())

		if i.TimeOfDay != nil {
			t, err := time.Parse("15:04", *i.TimeOfDay)
			if err == nil {
				targetThisMonth = time.Date(
					targetThisMonth.Year(),
					targetThisMonth.Month(),
					targetThisMonth.Day(),
					t.Hour(),
					t.Minute(),
					0,
					0,
					targetThisMonth.Location(),
				)
			}
		}

		// If current time is at or after the target time this month
		// and no backup has been made at or after the target time, trigger
		if now.After(targetThisMonth) || now.Equal(targetThisMonth) {
			return lastBackup.Before(targetThisMonth)
		}

		return false
	}
	// no DayOfMonth: if we're in a new calendar month
	return lastBackup.Before(getStartOfMonth(now))
}

func isSameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func getStartOfWeek(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-wd+1, 0, 0, 0, 0, t.Location())
}

func getStartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// cron trigger: check if we've passed a scheduled cron time since last backup
func (i *Interval) shouldTriggerCron(now, lastBackup time.Time) bool {
	if i.CronExpression == nil || *i.CronExpression == "" {
		return false
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(*i.CronExpression)
	if err != nil {
		return false
	}

	// Find the next scheduled time after the last backup
	nextAfterLastBackup := schedule.Next(lastBackup)

	// If we're at or past that next scheduled time, trigger
	return now.After(nextAfterLastBackup) || now.Equal(nextAfterLastBackup)
}

func (i *Interval) nextDailyTrigger(now time.Time) time.Time {
	t, err := time.Parse("15:04", *i.TimeOfDay)
	if err != nil {
		return now
	}

	todaySlot := time.Date(
		now.Year(), now.Month(), now.Day(),
		t.Hour(), t.Minute(), 0, 0, now.Location(),
	)

	if now.Before(todaySlot) {
		return todaySlot
	}

	return todaySlot.AddDate(0, 0, 1)
}

func (i *Interval) nextWeeklyTrigger(now time.Time) time.Time {
	targetWd := time.Weekday(0)
	if i.Weekday != nil {
		targetWd = time.Weekday(*i.Weekday)
	}

	startOfWeek := getStartOfWeek(now)

	var daysFromMonday int
	if targetWd == time.Sunday {
		daysFromMonday = 6
	} else {
		daysFromMonday = int(targetWd) - 1
	}

	targetThisWeek := startOfWeek.AddDate(0, 0, daysFromMonday)

	if i.TimeOfDay != nil {
		t, err := time.Parse("15:04", *i.TimeOfDay)
		if err == nil {
			targetThisWeek = time.Date(
				targetThisWeek.Year(), targetThisWeek.Month(), targetThisWeek.Day(),
				t.Hour(), t.Minute(), 0, 0, targetThisWeek.Location(),
			)
		}
	}

	if now.Before(targetThisWeek) {
		return targetThisWeek
	}

	return targetThisWeek.AddDate(0, 0, 7)
}

func (i *Interval) nextMonthlyTrigger(now time.Time) time.Time {
	day := 1
	if i.DayOfMonth != nil {
		day = *i.DayOfMonth
	}

	targetThisMonth := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, now.Location())

	if i.TimeOfDay != nil {
		t, err := time.Parse("15:04", *i.TimeOfDay)
		if err == nil {
			targetThisMonth = time.Date(
				targetThisMonth.Year(), targetThisMonth.Month(), targetThisMonth.Day(),
				t.Hour(), t.Minute(), 0, 0, targetThisMonth.Location(),
			)
		}
	}

	if now.Before(targetThisMonth) {
		return targetThisMonth
	}

	return targetThisMonth.AddDate(0, 1, 0)
}

func (i *Interval) nextCronTrigger(lastBackup time.Time) *time.Time {
	if i.CronExpression == nil || *i.CronExpression == "" {
		return nil
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(*i.CronExpression)
	if err != nil {
		return nil
	}

	next := schedule.Next(lastBackup)

	return &next
}

func (i *Interval) validateCronExpression(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	if err != nil {
		return errors.New("invalid cron expression: " + err.Error())
	}
	return nil
}
