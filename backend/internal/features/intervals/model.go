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

// GetNextTrigger reports when the scheduler will first treat a backup as due.
// Returns nil when the interval cannot be evaluated. Passing Validate is what lets
// the next*Trigger helpers dereference fields and ignore parse errors.
func (i *Interval) GetNextTrigger(now time.Time, lastBackupTime *time.Time) *NextTrigger {
	if i.Validate() != nil {
		return nil
	}

	if i.ShouldTriggerBackup(now, lastBackupTime) {
		return &NextTrigger{ScheduledAt: now, IsDue: true}
	}

	switch i.Type {
	case IntervalHourly:
		return &NextTrigger{ScheduledAt: lastBackupTime.Add(time.Hour)}

	case IntervalDaily:
		return &NextTrigger{ScheduledAt: i.nextDailyTrigger(now)}

	case IntervalWeekly:
		return &NextTrigger{ScheduledAt: i.nextWeeklyTrigger(now)}

	case IntervalMonthly:
		return &NextTrigger{ScheduledAt: i.nextMonthlyTrigger(now)}

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

// A missed slot is caught up: the backup fires on the first check after it.
func (i *Interval) shouldTriggerDaily(now, lastBackup time.Time) bool {
	if i.TimeOfDay == nil {
		return !isSameDay(lastBackup, now)
	}

	todaySlot, isParsed := i.getTodaySlot(now)
	if !isParsed {
		return false // malformed ⇒ play safe
	}

	latestSlot := todaySlot
	if now.Before(todaySlot) {
		latestSlot = todaySlot.AddDate(0, 0, -1)
	}

	return !now.Before(latestSlot) && lastBackup.Before(latestSlot)
}

// Only this week's slot counts, so a slot missed in an earlier week is not caught up.
// Without a weekday the interval is a plain 7 days.
func (i *Interval) shouldTriggerWeekly(now, lastBackup time.Time) bool {
	if i.Weekday == nil {
		return now.Sub(lastBackup) >= 7*24*time.Hour
	}

	thisWeekSlot := i.getThisWeekSlot(now)

	return !now.Before(thisWeekSlot) && lastBackup.Before(thisWeekSlot)
}

// Only this month's slot counts, so a slot missed in an earlier month is not caught up.
// A day the month lacks overflows into the next month and is never reached, so that
// month gets no backup. Without a day of month, the first check of a new month fires.
func (i *Interval) shouldTriggerMonthly(now, lastBackup time.Time) bool {
	if i.DayOfMonth == nil {
		return lastBackup.Before(getStartOfMonth(now))
	}

	thisMonthSlot := i.getMonthSlot(now)

	return !now.Before(thisMonthSlot) && lastBackup.Before(thisMonthSlot)
}

func (i *Interval) getTodaySlot(now time.Time) (time.Time, bool) {
	t, err := time.Parse("15:04", *i.TimeOfDay)
	if err != nil {
		return time.Time{}, false
	}

	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), true
}

// The weekly slot is at midnight when TimeOfDay is missing or malformed.
// Weekday 7 lands on Sunday like 0.
func (i *Interval) getThisWeekSlot(now time.Time) time.Time {
	weekday := time.Weekday(*i.Weekday)

	daysFromMonday := int(weekday) - 1
	if weekday == time.Sunday {
		daysFromMonday = 6
	}

	return i.atTimeOfDay(getStartOfWeek(now).AddDate(0, 0, daysFromMonday))
}

// The monthly slot is at midnight when TimeOfDay is missing or malformed. A day the
// month lacks is normalized into the following month, like time.Date does.
func (i *Interval) getMonthSlot(month time.Time) time.Time {
	return i.atTimeOfDay(time.Date(month.Year(), month.Month(), *i.DayOfMonth, 0, 0, 0, 0, month.Location()))
}

func (i *Interval) atTimeOfDay(day time.Time) time.Time {
	if i.TimeOfDay == nil {
		return day
	}

	t, err := time.Parse("15:04", *i.TimeOfDay)
	if err != nil {
		return day
	}

	return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, day.Location())
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

func (i *Interval) shouldTriggerCron(now, lastBackup time.Time) bool {
	if i.CronExpression == nil || *i.CronExpression == "" {
		return false
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(*i.CronExpression)
	if err != nil {
		return false
	}

	return !now.Before(schedule.Next(lastBackup))
}

func (i *Interval) nextDailyTrigger(now time.Time) time.Time {
	todaySlot, _ := i.getTodaySlot(now)

	if now.Before(todaySlot) {
		return todaySlot
	}

	return todaySlot.AddDate(0, 0, 1)
}

func (i *Interval) nextWeeklyTrigger(now time.Time) time.Time {
	thisWeekSlot := i.getThisWeekSlot(now)

	if now.Before(thisWeekSlot) {
		return thisWeekSlot
	}

	return thisWeekSlot.AddDate(0, 0, 7)
}

// Months without the configured day are skipped, as in shouldTriggerMonthly.
func (i *Interval) nextMonthlyTrigger(now time.Time) time.Time {
	for month := getStartOfMonth(now); ; month = month.AddDate(0, 1, 0) {
		monthSlot := i.getMonthSlot(month)

		if monthSlot.Month() == month.Month() && monthSlot.After(now) {
			return monthSlot
		}
	}
}

func (i *Interval) nextCronTrigger(lastBackup time.Time) *NextTrigger {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(*i.CronExpression)
	if err != nil {
		return nil
	}

	return &NextTrigger{ScheduledAt: schedule.Next(lastBackup)}
}

func (i *Interval) validateCronExpression(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	if err != nil {
		return errors.New("invalid cron expression: " + err.Error())
	}
	return nil
}
