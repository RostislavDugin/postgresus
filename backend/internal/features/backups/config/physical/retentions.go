package backups_config_physical

import (
	"errors"
	"fmt"
)

type Retention string

const (
	RetentionChains               Retention = "CHAINS"
	RetentionFullBackups          Retention = "FULL_BACKUPS"
	RetentionChainsAndFullBackups Retention = "CHAINS_AND_FULL_BACKUPS"
)

type FullBackupsRetentionPolicy string

const (
	FullBackupsRetentionPolicyLastN FullBackupsRetentionPolicy = "LAST_N"
	FullBackupsRetentionPolicyGfs   FullBackupsRetentionPolicy = "GFS"
)

type ChainsRetention struct {
	Count int `json:"count" gorm:"column:count;type:int;not null;default:0"`
}

func (r ChainsRetention) IsZero() bool { return r.Count == 0 }

type FullBackupsRetention struct {
	Policy FullBackupsRetentionPolicy `json:"policy" gorm:"column:policy;type:text;not null;default:''"`
	Count  int                        `json:"count"  gorm:"column:count;type:int;not null;default:0"`

	GfsHours  int `json:"gfsHours"  gorm:"column:gfs_hours;type:int;not null;default:0"`
	GfsDays   int `json:"gfsDays"   gorm:"column:gfs_days;type:int;not null;default:0"`
	GfsWeeks  int `json:"gfsWeeks"  gorm:"column:gfs_weeks;type:int;not null;default:0"`
	GfsMonths int `json:"gfsMonths" gorm:"column:gfs_months;type:int;not null;default:0"`
	GfsYears  int `json:"gfsYears"  gorm:"column:gfs_years;type:int;not null;default:0"`
}

func (r FullBackupsRetention) IsZero() bool {
	return r.Policy == "" &&
		r.Count == 0 &&
		!r.hasAnyGfsBucket()
}

func (r FullBackupsRetention) Validate() error {
	switch r.Policy {
	case FullBackupsRetentionPolicyLastN:
		if r.Count <= 0 {
			return errors.New("LAST_N policy requires count > 0")
		}

		if r.hasAnyGfsBucket() {
			return errors.New("LAST_N policy must not set GFS buckets")
		}

	case FullBackupsRetentionPolicyGfs:
		if r.Count != 0 {
			return errors.New("GFS policy must not set count")
		}

		if !r.hasAnyGfsBucket() {
			return errors.New("GFS policy requires at least one bucket > 0")
		}

	default:
		return fmt.Errorf("invalid full backups retention policy: %q", r.Policy)
	}

	return nil
}

func (r FullBackupsRetention) hasAnyGfsBucket() bool {
	return r.GfsHours > 0 ||
		r.GfsDays > 0 ||
		r.GfsWeeks > 0 ||
		r.GfsMonths > 0 ||
		r.GfsYears > 0
}
