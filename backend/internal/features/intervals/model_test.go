package intervals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterval_ShouldTriggerBackup_Hourly(t *testing.T) {
	interval := &Interval{
		Type: IntervalHourly,
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
		Type:      IntervalDaily,
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
				Type:      IntervalDaily,
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
			Type:      IntervalDaily,
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
		Type:      IntervalWeekly,
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
				Type:      IntervalWeekly,
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
		Type:       IntervalMonthly,
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
		Type:           IntervalCron,
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
			Type:           IntervalCron,
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
			Type:           IntervalCron,
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
			Type:           IntervalCron,
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
			Type:           IntervalCron,
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
			Type:           IntervalCron,
			CronExpression: &emptyCron,
		}

		now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		lastBackup := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)

		should := emptyInterval.ShouldTriggerBackup(now, &lastBackup)
		assert.False(t, should)
	})

	t.Run("Nil cron expression returns false", func(t *testing.T) {
		nilInterval := &Interval{
			Type:           IntervalCron,
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
			Type: IntervalDaily,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "time of day is required")
	})

	t.Run("Weekly interval requires weekday", func(t *testing.T) {
		timeOfDay := "09:00"
		interval := &Interval{
			Type:      IntervalWeekly,
			TimeOfDay: &timeOfDay,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "weekday is required")
	})

	t.Run("Monthly interval requires day of month", func(t *testing.T) {
		timeOfDay := "09:00"
		interval := &Interval{
			Type:      IntervalMonthly,
			TimeOfDay: &timeOfDay,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "day of month is required")
	})

	t.Run("Hourly interval is valid without additional fields", func(t *testing.T) {
		interval := &Interval{
			Type: IntervalHourly,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Valid weekly interval", func(t *testing.T) {
		timeOfDay := "09:00"
		weekday := 1
		interval := &Interval{
			Type:      IntervalWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Daily interval with invalid time of day is invalid", func(t *testing.T) {
		timeOfDay := "25:00"
		interval := &Interval{
			Type:      IntervalDaily,
			TimeOfDay: &timeOfDay,
		}
		err := interval.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time of day")
	})

	t.Run("Daily interval with empty time of day is invalid", func(t *testing.T) {
		timeOfDay := ""
		interval := &Interval{
			Type:      IntervalDaily,
			TimeOfDay: &timeOfDay,
		}
		err := interval.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time of day")
	})

	t.Run("Weekly interval with invalid weekday is invalid", func(t *testing.T) {
		timeOfDay := "09:00"
		weekday := 8
		interval := &Interval{
			Type:      IntervalWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := interval.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "weekday must be between 0 and 7")
	})

	t.Run("Weekly interval with negative weekday is invalid", func(t *testing.T) {
		timeOfDay := "09:00"
		weekday := -1
		interval := &Interval{
			Type:      IntervalWeekly,
			TimeOfDay: &timeOfDay,
			Weekday:   &weekday,
		}
		err := interval.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "weekday must be between 0 and 7")
	})

	t.Run("Weekly interval with weekday 7 (Sunday alias) is valid", func(t *testing.T) {
		timeOfDay := "09:00"
		weekday := 7
		interval := &Interval{
			Type:      IntervalWeekly,
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
			Type:       IntervalMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Monthly interval with invalid day of month is invalid", func(t *testing.T) {
		timeOfDay := "09:00"
		dayOfMonth := 0
		interval := &Interval{
			Type:       IntervalMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := interval.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "day of month must be between 1 and 31")
	})

	t.Run("Monthly interval with day of month above 31 is invalid", func(t *testing.T) {
		timeOfDay := "09:00"
		dayOfMonth := 32
		interval := &Interval{
			Type:       IntervalMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := interval.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "day of month must be between 1 and 31")
	})

	t.Run("Monthly interval with day of month 31 is valid", func(t *testing.T) {
		timeOfDay := "09:00"
		dayOfMonth := 31
		interval := &Interval{
			Type:       IntervalMonthly,
			TimeOfDay:  &timeOfDay,
			DayOfMonth: &dayOfMonth,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Cron interval requires cron expression", func(t *testing.T) {
		interval := &Interval{
			Type: IntervalCron,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cron expression is required")
	})

	t.Run("Cron interval with empty expression is invalid", func(t *testing.T) {
		emptyCron := ""
		interval := &Interval{
			Type:           IntervalCron,
			CronExpression: &emptyCron,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cron expression is required")
	})

	t.Run("Cron interval with invalid expression is invalid", func(t *testing.T) {
		invalidCron := "invalid cron"
		interval := &Interval{
			Type:           IntervalCron,
			CronExpression: &invalidCron,
		}
		err := interval.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("Valid cron interval with daily expression", func(t *testing.T) {
		cronExpr := "0 2 * * *" // Daily at 2 AM
		interval := &Interval{
			Type:           IntervalCron,
			CronExpression: &cronExpr,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})

	t.Run("Valid cron interval with complex expression", func(t *testing.T) {
		cronExpr := "30 4 1,15 * *" // 1st and 15th of each month at 4:30 AM
		interval := &Interval{
			Type:           IntervalCron,
			CronExpression: &cronExpr,
		}
		err := interval.Validate()
		assert.NoError(t, err)
	})
}

func Test_GetNextTrigger_ForEveryIntervalType_MatchesShouldTriggerBackup(t *testing.T) {
	testCases := []struct {
		name            string
		interval        Interval
		now             time.Time
		lastBackupTime  *time.Time
		nextScheduledAt time.Time
		isDueNow        bool
	}{
		{
			name:            "Hourly with a recent backup",
			interval:        Interval{Type: IntervalHourly},
			now:             time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 15, 11, 30, 15, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 12, 30, 15, 0, time.UTC),
		},
		{
			name:            "Hourly with an hour elapsed",
			interval:        Interval{Type: IntervalHourly},
			now:             time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
			isDueNow:        true,
		},
		{
			name:            "Daily without any backup",
			interval:        Interval{Type: IntervalDaily, TimeOfDay: new("03:00")},
			now:             time.Date(2026, 1, 15, 1, 0, 0, 0, time.UTC),
			nextScheduledAt: time.Date(2026, 1, 15, 1, 0, 0, 0, time.UTC),
			isDueNow:        true,
		},
		{
			name:            "Daily before today's slot",
			interval:        Interval{Type: IntervalDaily, TimeOfDay: new("03:00")},
			now:             time.Date(2026, 1, 15, 1, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 14, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC),
		},
		{
			name:            "Daily after today's backup",
			interval:        Interval{Type: IntervalDaily, TimeOfDay: new("03:00")},
			now:             time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 15, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 16, 3, 0, 0, 0, time.UTC),
		},
		{
			name:            "Daily with a missed slot",
			interval:        Interval{Type: IntervalDaily, TimeOfDay: new("03:00")},
			now:             time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 13, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			isDueNow:        true,
		},
		{
			name:            "Daily exactly at the slot",
			interval:        Interval{Type: IntervalDaily, TimeOfDay: new("03:00")},
			now:             time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 14, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC),
			isDueNow:        true,
		},
		{
			name: "Weekly before this week's slot",
			interval: Interval{
				Type:      IntervalWeekly,
				TimeOfDay: new("03:00"),
				Weekday:   new(int(time.Wednesday)),
			},
			now:             time.Date(2026, 1, 13, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 7, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 14, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Weekly with a slot missed in an earlier week",
			interval: Interval{
				Type:      IntervalWeekly,
				TimeOfDay: new("03:00"),
				Weekday:   new(int(time.Wednesday)),
			},
			now:             time.Date(2026, 1, 13, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2025, 12, 24, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 14, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Weekly after this week's backup",
			interval: Interval{
				Type:      IntervalWeekly,
				TimeOfDay: new("03:00"),
				Weekday:   new(int(time.Wednesday)),
			},
			now:             time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 14, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 21, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Weekly with this week's slot missed",
			interval: Interval{
				Type:      IntervalWeekly,
				TimeOfDay: new("03:00"),
				Weekday:   new(int(time.Wednesday)),
			},
			now:             time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 7, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			isDueNow:        true,
		},
		{
			name: "Weekly on Sunday written as 7",
			interval: Interval{
				Type:      IntervalWeekly,
				TimeOfDay: new("03:00"),
				Weekday:   new(7),
			},
			now:             time.Date(2026, 1, 13, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 11, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 18, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly before this month's slot",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("08:00"),
				DayOfMonth: new(10),
			},
			now:             time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2025, 12, 10, 8, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly after this month's backup",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("08:00"),
				DayOfMonth: new(10),
			},
			now:             time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 10, 8, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly with this month's slot missed",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("08:00"),
				DayOfMonth: new(10),
			},
			now:             time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2025, 12, 10, 8, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			isDueNow:        true,
		},
		{
			name: "Monthly on day 31 skips February",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("03:00"),
				DayOfMonth: new(31),
			},
			now:             time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 31, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 3, 31, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly on day 31 skips a 30-day month",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("03:00"),
				DayOfMonth: new(31),
			},
			now:             time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 3, 31, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 5, 31, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly on day 30 skips February",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("03:00"),
				DayOfMonth: new(30),
			},
			now:             time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 30, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 3, 30, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly on day 29 skips February outside a leap year",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("03:00"),
				DayOfMonth: new(29),
			},
			now:             time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 29, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 3, 29, 3, 0, 0, 0, time.UTC),
		},
		{
			name: "Monthly on day 29 keeps February in a leap year",
			interval: Interval{
				Type:       IntervalMonthly,
				TimeOfDay:  new("03:00"),
				DayOfMonth: new(29),
			},
			now:             time.Date(2028, 2, 1, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2028, 1, 29, 3, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2028, 2, 29, 3, 0, 0, 0, time.UTC),
		},
		{
			name:            "Cron before the next run",
			interval:        Interval{Type: IntervalCron, CronExpression: new("0 */6 * * *")},
			now:             time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 15, 6, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:            "Cron with a missed run",
			interval:        Interval{Type: IntervalCron, CronExpression: new("0 */6 * * *")},
			now:             time.Date(2026, 1, 15, 13, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 15, 6, 0, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 13, 0, 0, 0, time.UTC),
			isDueNow:        true,
		},
		{
			name:            "Cron on the 1st and 15th",
			interval:        Interval{Type: IntervalCron, CronExpression: new("30 4 1,15 * *")},
			now:             time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
			lastBackupTime:  new(time.Date(2026, 1, 1, 4, 30, 20, 0, time.UTC)),
			nextScheduledAt: time.Date(2026, 1, 15, 4, 30, 0, 0, time.UTC),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.NoError(t, testCase.interval.Validate())

			nextTrigger := testCase.interval.GetNextTrigger(testCase.now, testCase.lastBackupTime)

			require.NotNil(t, nextTrigger)
			assert.Equal(t, testCase.nextScheduledAt, nextTrigger.ScheduledAt)
			assert.Equal(t, testCase.isDueNow, nextTrigger.IsDue)
			assertNextTriggerMatchesShouldTriggerBackup(
				t, &testCase.interval, testCase.now, testCase.lastBackupTime, nextTrigger)
		})
	}
}

func Test_GetNextTrigger_WhenIntervalCannotBeEvaluated_ReturnsNil(t *testing.T) {
	testCases := []struct {
		name     string
		interval Interval
	}{
		{name: "Unknown type", interval: Interval{Type: IntervalType("UNKNOWN")}},
		{name: "Daily without time of day", interval: Interval{Type: IntervalDaily}},
		{name: "Daily with malformed time of day", interval: Interval{Type: IntervalDaily, TimeOfDay: new("25:99")}},
		{name: "Weekly without weekday", interval: Interval{Type: IntervalWeekly, TimeOfDay: new("03:00")}},
		{name: "Weekly without time of day", interval: Interval{Type: IntervalWeekly, Weekday: new(3)}},
		{name: "Monthly without day of month", interval: Interval{Type: IntervalMonthly, TimeOfDay: new("03:00")}},
		{name: "Monthly with day of month out of range", interval: Interval{
			Type:       IntervalMonthly,
			TimeOfDay:  new("03:00"),
			DayOfMonth: new(32),
		}},
		{name: "Cron without expression", interval: Interval{Type: IntervalCron}},
		{name: "Cron with empty expression", interval: Interval{Type: IntervalCron, CronExpression: new("")}},
		{name: "Cron with unparsable expression", interval: Interval{
			Type:           IntervalCron,
			CronExpression: new("invalid cron"),
		}},
	}

	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	lastBackupTime := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Nil(t, testCase.interval.GetNextTrigger(now, &lastBackupTime))
			assert.Nil(t, testCase.interval.GetNextTrigger(now, nil))
		})
	}
}

// The scheduler acts on ShouldTriggerBackup, so a reported next trigger is only
// truthful when the due check stays false up to it and turns true exactly at it.
func assertNextTriggerMatchesShouldTriggerBackup(
	t *testing.T,
	interval *Interval,
	now time.Time,
	lastBackupTime *time.Time,
	nextTrigger *NextTrigger,
) {
	t.Helper()

	require.Equal(t, interval.ShouldTriggerBackup(now, lastBackupTime), nextTrigger.IsDue)

	if nextTrigger.IsDue {
		require.Equal(t, now, nextTrigger.ScheduledAt)

		return
	}

	for moment := now; moment.Before(nextTrigger.ScheduledAt); moment = moment.Add(time.Minute) {
		require.Falsef(t, interval.ShouldTriggerBackup(moment, lastBackupTime),
			"due check fired at %s, before the reported %s", moment, nextTrigger.ScheduledAt)
	}

	require.False(t, interval.ShouldTriggerBackup(nextTrigger.ScheduledAt.Add(-time.Second), lastBackupTime))
	require.True(t, interval.ShouldTriggerBackup(nextTrigger.ScheduledAt, lastBackupTime))
}
