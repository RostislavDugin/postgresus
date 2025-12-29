package intervals

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestInterval_ShouldTriggerBackup_Hourly(t *testing.T) {
	interval := &Interval{
		ID:       uuid.New(),
		Interval: IntervalHourly,
	}

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("No previous backup: Trigger backup immediately", func(t *testing.T) {
		should := interval.ShouldTriggerBackup(baseTime, nil)
		assert.True(t, should)
	})

	t.Run("Last backup 59 minutes ago: Do not trigger backup", func(t *testing.T) {
		lastBackup := baseTime.Add(-59 * time.Minute)
		should := interval.ShouldTriggerBackup(baseTime, &lastBackup)
		assert.False(t, should)
	})

	t.Run("Last backup exactly 1 hour ago: Trigger backup", func(t *testing.T) {
		lastBackup := baseTime.Add(-1 * time.Hour)
		should := interval.ShouldTriggerBackup(baseTime, &lastBackup)
		assert.True(t, should)
	})

	t.Run("Last backup 2 hours ago: Trigger backup", func(t *testing.T) {
		lastBackup := baseTime.Add(-2 * time.Hour)
		should := interval.ShouldTriggerBackup(baseTime, &lastBackup)
		assert.True(t, should)
	})
}

func TestInterval_ShouldTriggerBackup_Daily(t *testing.T) {
	timeOfDay := "09:00"
	interval := &Interval{
		ID:        uuid.New(),
		Interval:  IntervalDaily,
		TimeOfDay: &timeOfDay,
	}

	// Base time: January 15, 2024
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("No previous backup: Trigger backup immediately", func(t *testing.T) {
		now := baseDate.Add(10 * time.Hour) // 10:00 AM
		should := interval.ShouldTriggerBackup(now, nil)
		assert.True(t, should)
	})

	t.Run("Today 08:59, no backup today: Do not trigger backup", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 8, 59, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run("Today exactly 09:00, no backup today: Trigger backup", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should)
	})

	t.Run("Today 09:01, no backup today: Trigger backup", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 9, 1, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should)
	})

	t.Run("Backup earlier today at 09:00: Do not trigger another backup", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)       // 3 PM
		lastBackup := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC) // Today at 9 AM
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run(
		"Backup yesterday at correct time: Trigger backup today at or after 09:00",
		func(t *testing.T) {
			now := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
			lastBackup := time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC) // Yesterday at 9 AM
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"Backup yesterday at 15:00: Trigger backup today at 09:00",
		func(t *testing.T) {
			now := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
			lastBackup := time.Date(2024, 1, 14, 15, 0, 0, 0, time.UTC) // Yesterday at 15:00
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"Manual backup before scheduled time should not prevent scheduled backup",
		func(t *testing.T) {
			timeOfDay := "21:00"
			interval := &Interval{
				ID:        uuid.New(),
				Interval:  IntervalDaily,
				TimeOfDay: &timeOfDay,
			}

			manual := time.Date(2025, 6, 6, 16, 17, 0, 0, time.UTC)   // manual earlier
			scheduled := time.Date(2025, 6, 6, 21, 0, 0, 0, time.UTC) // scheduled time

			should := interval.ShouldTriggerBackup(scheduled, &manual)
			assert.True(t, should, "scheduled run should trigger even after earlier manual backup")
		},
	)

	t.Run("Catch up previous time slot", func(t *testing.T) {
		timeOfDay := "21:00"
		interval := &Interval{
			ID:        uuid.New(),
			Interval:  IntervalDaily,
			TimeOfDay: &timeOfDay,
		}

		// It's June-07 15:00 UTC, yesterday's scheduled backup was missed
		now := time.Date(2025, 6, 7, 15, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2025, 6, 6, 16, 0, 0, 0, time.UTC) // before yesterday's 21:00

		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should, "should catch up missed 21:00 backup the next day at 15:00")
	})
}

func TestInterval_ShouldTriggerBackup_Weekly(t *testing.T) {
	timeOfDay := "15:00"
	weekday := 3 // Wednesday (0=Sunday, 1=Monday, ..., 3=Wednesday)
	interval := &Interval{
		ID:        uuid.New(),
		Interval:  IntervalWeekly,
		TimeOfDay: &timeOfDay,
		Weekday:   &weekday,
	}

	// Base time: Wednesday, January 17, 2024 (to ensure we're on Wednesday)
	wednesday := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	t.Run("No previous backup: Trigger backup immediately", func(t *testing.T) {
		now := wednesday.Add(16 * time.Hour) // 4 PM Wednesday
		should := interval.ShouldTriggerBackup(now, nil)
		assert.True(t, should)
	})

	t.Run(
		"Today Wednesday at 14:59, no backup this week: Do not trigger backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 17, 14, 59, 0, 0, time.UTC)
			lastBackup := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.False(t, should)
		},
	)

	t.Run(
		"Today Wednesday at exactly 15:00, no backup this week: Trigger backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 17, 15, 0, 0, 0, time.UTC)
			lastBackup := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run("Today Wednesday at 15:01, no backup this week: Trigger backup", func(t *testing.T) {
		now := time.Date(2024, 1, 17, 15, 1, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should)
	})

	t.Run(
		"Backup already done at scheduled time (Wednesday 15:00): Do not trigger again",
		func(t *testing.T) {
			now := time.Date(2024, 1, 18, 10, 0, 0, 0, time.UTC) // Thursday

			// Wednesday this week at scheduled time
			lastBackup := time.Date(
				2024,
				1,
				17,
				15,
				0,
				0,
				0,
				time.UTC,
			)
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.False(t, should)
		},
	)

	t.Run(
		"Manual backup before scheduled time should not prevent scheduled backup",
		func(t *testing.T) {
			// Wednesday at scheduled time
			now := time.Date(
				2024,
				1,
				17,
				15,
				0,
				0,
				0,
				time.UTC,
			)
			// Manual backup same day, before scheduled time
			lastBackup := time.Date(
				2024,
				1,
				17,
				10,
				0,
				0,
				0,
				time.UTC,
			)
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"Manual backup after scheduled time should prevent another backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 18, 10, 0, 0, 0, time.UTC) // Thursday
			// Manual backup after scheduled time
			lastBackup := time.Date(
				2024,
				1,
				17,
				16,
				0,
				0,
				0,
				time.UTC,
			)
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.False(t, should)
		},
	)

	t.Run(
		"Backup missed completely: Trigger backup immediately after scheduled time",
		func(t *testing.T) {
			// Thursday after missed Wednesday
			now := time.Date(
				2024,
				1,
				18,
				10,
				0,
				0,
				0,
				time.UTC,
			)
			lastBackup := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"Backup last week: Trigger backup at this week's scheduled time",
		func(t *testing.T) {
			// Wednesday at scheduled time
			now := time.Date(
				2024,
				1,
				17,
				15,
				0,
				0,
				0,
				time.UTC,
			)
			lastBackup := time.Date(2024, 1, 10, 15, 0, 0, 0, time.UTC) // Previous week
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"User's scenario: Weekly Friday 00:00 backup should trigger even after Wednesday manual backup",
		func(t *testing.T) {
			timeOfDay := "00:00"
			weekday := 5 // Friday (0=Sunday, 1=Monday, ..., 5=Friday)
			fridayInterval := &Interval{
				ID:        uuid.New(),
				Interval:  IntervalWeekly,
				TimeOfDay: &timeOfDay,
				Weekday:   &weekday,
			}

			// Friday at 00:00 - scheduled backup time
			friday := time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC) // Friday Jan 19, 2024
			// Manual backup was done on Wednesday
			wednesdayBackup := time.Date(
				2024,
				1,
				17,
				21,
				0,
				0,
				0,
				time.UTC,
			) // Wednesday Jan 17, 2024 at 21:00

			should := fridayInterval.ShouldTriggerBackup(friday, &wednesdayBackup)
			assert.True(
				t,
				should,
				"Friday scheduled backup should trigger despite Wednesday manual backup",
			)
		},
	)
}

func TestInterval_ShouldTriggerBackup_Monthly(t *testing.T) {
	timeOfDay := "08:00"
	dayOfMonth := 10
	interval := &Interval{
		ID:         uuid.New(),
		Interval:   IntervalMonthly,
		TimeOfDay:  &timeOfDay,
		DayOfMonth: &dayOfMonth,
	}

	t.Run("No previous backup: Trigger backup immediately", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		should := interval.ShouldTriggerBackup(now, nil)
		assert.True(t, should)
	})

	t.Run(
		"Today is the 10th at 07:59, no backup this month: Do not trigger backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 10, 7, 59, 0, 0, time.UTC)
			lastBackup := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.False(t, should)
		},
	)

	t.Run(
		"Today is the 10th exactly 08:00, no backup this month: Trigger backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
			lastBackup := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"Today is the 10th after 08:00, no backup this month: Trigger backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 10, 8, 1, 0, 0, time.UTC)
			lastBackup := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"Today is the 11th, backup missed on the 10th: Trigger backup immediately",
		func(t *testing.T) {
			now := time.Date(2024, 1, 11, 10, 0, 0, 0, time.UTC)
			lastBackup := time.Date(2023, 12, 10, 8, 0, 0, 0, time.UTC) // Previous month
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run("Backup already performed at scheduled time: Do not trigger again", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC) // This month at scheduled time
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run(
		"Manual backup before scheduled time should not prevent scheduled backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC) // Scheduled time
			// Manual backup earlier this month
			lastBackup := time.Date(
				2024,
				1,
				5,
				10,
				0,
				0,
				0,
				time.UTC,
			)
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)

	t.Run(
		"Manual backup after scheduled time should prevent another backup",
		func(t *testing.T) {
			now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
			// Manual backup after scheduled time
			lastBackup := time.Date(
				2024,
				1,
				10,
				9,
				0,
				0,
				0,
				time.UTC,
			)
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.False(t, should)
		},
	)

	t.Run(
		"Backup performed last month on schedule: Trigger backup this month at scheduled time",
		func(t *testing.T) {
			now := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
			// Previous month at scheduled time
			lastBackup := time.Date(
				2023,
				12,
				10,
				8,
				0,
				0,
				0,
				time.UTC,
			)
			should := interval.ShouldTriggerBackup(now, &lastBackup)
			assert.True(t, should)
		},
	)
}

func TestInterval_ShouldTriggerBackup_Cron(t *testing.T) {
	cronExpr := "0 2 * * *" // Daily at 2:00 AM
	interval := &Interval{
		ID:             uuid.New(),
		Interval:       IntervalCron,
		CronExpression: &cronExpr,
	}

	t.Run("No previous backup: Trigger backup immediately", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		should := interval.ShouldTriggerBackup(now, nil)
		assert.True(t, should)
	})

	t.Run("Before scheduled cron time: Do not trigger backup", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 1, 59, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 2, 0, 0, 0, time.UTC) // Yesterday at 2 AM
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run("Exactly at scheduled cron time: Trigger backup", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 2, 0, 0, 0, time.UTC) // Yesterday at 2 AM
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should)
	})

	t.Run("After scheduled cron time: Trigger backup", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 2, 0, 0, 0, time.UTC) // Yesterday at 2 AM
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should)
	})

	t.Run("Backup already done after scheduled time: Do not trigger again", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 15, 2, 5, 0, 0, time.UTC) // Today at 2:05 AM
		should := interval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run("Weekly cron expression: 0 3 * * 1 (Monday at 3 AM)", func(t *testing.T) {
		weeklyCron := "0 3 * * 1" // Every Monday at 3 AM
		weeklyInterval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &weeklyCron,
		}

		// Monday Jan 15, 2024 at 3:00 AM
		monday := time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC)
		// Last backup was previous Monday
		lastBackup := time.Date(2024, 1, 8, 3, 0, 0, 0, time.UTC)

		should := weeklyInterval.ShouldTriggerBackup(monday, &lastBackup)
		assert.True(t, should)
	})

	t.Run("Complex cron expression: 30 4 1,15 * * (1st and 15th at 4:30 AM)", func(t *testing.T) {
		complexCron := "30 4 1,15 * *" // 1st and 15th of each month at 4:30 AM
		complexInterval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &complexCron,
		}

		// Jan 15, 2024 at 4:30 AM
		now := time.Date(2024, 1, 15, 4, 30, 0, 0, time.UTC)
		// Last backup was Jan 1
		lastBackup := time.Date(2024, 1, 1, 4, 30, 0, 0, time.UTC)

		should := complexInterval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should)
	})

	t.Run("Every 6 hours cron expression: 0 */6 * * *", func(t *testing.T) {
		sixHourlyCron := "0 */6 * * *" // Every 6 hours (0:00, 6:00, 12:00, 18:00)
		sixHourlyInterval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &sixHourlyCron,
		}

		// 12:00 - next trigger after 6:00
		now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		// Last backup was at 6:00
		lastBackup := time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC)

		should := sixHourlyInterval.ShouldTriggerBackup(now, &lastBackup)
		assert.True(t, should)
	})

	t.Run("Invalid cron expression returns false", func(t *testing.T) {
		invalidCron := "invalid cron"
		invalidInterval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &invalidCron,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)

		should := invalidInterval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run("Empty cron expression returns false", func(t *testing.T) {
		emptyCron := ""
		emptyInterval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &emptyCron,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)

		should := emptyInterval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run("Nil cron expression returns false", func(t *testing.T) {
		nilInterval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: nil,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)

		should := nilInterval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})
}

func TestInterval_Validate(t *testing.T) {
	t.Run("Daily interval requires time of day", func(t *testing.T) {
		interval := &Interval{
			ID:       uuid.New(),
			Interval: IntervalDaily,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "time of day is required")
	})

	t.Run("Weekly interval requires weekday", func(t *testing.T) {
		timeOfDay := "09:00"
		interval := &Interval{
			ID:        uuid.New(),
			Interval:  IntervalWeekly,
			TimeOfDay: &timeOfDay,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "weekday is required")
	})

	t.Run("Monthly interval requires day of month", func(t *testing.T) {
		timeOfDay := "09:00"
		interval := &Interval{
			ID:        uuid.New(),
			Interval:  IntervalMonthly,
			TimeOfDay: &timeOfDay,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "day of month is required")
	})

	t.Run("Hourly interval is valid without additional fields", func(t *testing.T) {
		interval := &Interval{
			ID:       uuid.New(),
			Interval: IntervalHourly,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Valid weekly interval", func(t *testing.T) {
		timeOfDay := "09:00"
		weekday := 1
		interval := &Interval{
			ID:        uuid.New(),
			Interval:  IntervalWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Valid monthly interval", func(t *testing.T) {
		timeOfDay := "09:00"
		dayOfMonth := 15
		interval := &Interval{
			ID:         uuid.New(),
			Interval:   IntervalMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Cron interval requires cron expression", func(t *testing.T) {
		interval := &Interval{
			ID:       uuid.New(),
			Interval: IntervalCron,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cron expression is required")
	})

	t.Run("Cron interval with empty expression is invalid", func(t *testing.T) {
		emptyCron := ""
		interval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &emptyCron,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cron expression is required")
	})

	t.Run("Cron interval with invalid expression is invalid", func(t *testing.T) {
		invalidCron := "invalid cron"
		interval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &invalidCron,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("Valid cron interval with daily expression", func(t *testing.T) {
		cronExpr := "0 2 * * *" // Daily at 2 AM
		interval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &cronExpr,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Valid cron interval with complex expression", func(t *testing.T) {
		cronExpr := "30 4 1,15 * *" // 1st and 15th of each month at 4:30 AM
		interval := &Interval{
			ID:             uuid.New(),
			Interval:       IntervalCron,
			CronExpression: &cronExpr,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})
}
