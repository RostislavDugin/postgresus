package clusters

import (
	"postgresus-backend/internal/storage"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClusterRepository struct{}

func (r *ClusterRepository) Save(cluster *Cluster) (*Cluster, error) {
	db := storage.GetDb()

	isNew := cluster.ID == uuid.Nil
	if isNew {
		cluster.ID = uuid.New()
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// ensure nested object has correct FK
		cluster.Postgresql.ClusterID = cluster.ID

		// handle backup interval object
		if cluster.BackupInterval != nil {
			if cluster.BackupInterval.ID == uuid.Nil {
				if err := tx.Create(cluster.BackupInterval).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(cluster.BackupInterval).Error; err != nil {
					return err
				}
			}
			// assign FK
			cluster.BackupIntervalID = &cluster.BackupInterval.ID
		}

		if isNew {
			if err := tx.Create(cluster).Omit("Postgresql", "Notifiers", "ExcludedDatabases", "BackupInterval").Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(cluster).Omit("Postgresql", "Notifiers", "ExcludedDatabases", "BackupInterval").Error; err != nil {
				return err
			}
		}

		// save pg settings
		if cluster.Postgresql.ID == uuid.Nil {
			cluster.Postgresql.ID = uuid.New()
			if err := tx.Create(&cluster.Postgresql).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(&cluster.Postgresql).Error; err != nil {
				return err
			}
		}

		// save notifiers relation
		if err := tx.Model(cluster).Association("Notifiers").Replace(cluster.Notifiers); err != nil {
			return err
		}

		// replace excluded databases manually to ensure ClusterID is set
		if err := tx.Where("cluster_id = ?", cluster.ID).Delete(&ClusterExcludedDatabase{}).Error; err != nil {
			return err
		}
		if len(cluster.ExcludedDatabases) > 0 {
			items := make([]ClusterExcludedDatabase, 0, len(cluster.ExcludedDatabases))
			for _, ed := range cluster.ExcludedDatabases {
				if ed.Name == "" {
					continue
				}
				ed.ClusterID = cluster.ID
				if ed.ID == uuid.Nil {
					ed.ID = uuid.New()
				}
				items = append(items, ed)
			}
			if len(items) > 0 {
				if err := tx.Create(&items).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (r *ClusterRepository) FindByID(id uuid.UUID) (*Cluster, error) {
	var cluster Cluster

	if err := storage.GetDb().
		Preload("Postgresql").
		Preload("Notifiers").
		Preload("BackupInterval").
		Preload("ExcludedDatabases").
		Where("id = ?", id).
		First(&cluster).Error; err != nil {
		return nil, err
	}

	return &cluster, nil
}

func (r *ClusterRepository) FindByWorkspaceID(workspaceID uuid.UUID) ([]*Cluster, error) {
	var clusters []*Cluster

	if err := storage.GetDb().
		Preload("Postgresql").
		Preload("Notifiers").
		Preload("BackupInterval").
		Preload("ExcludedDatabases").
		Where("workspace_id = ?", workspaceID).
		Order("name ASC").
		Find(&clusters).Error; err != nil {
		return nil, err
	}

	return clusters, nil
}

func (r *ClusterRepository) Delete(id uuid.UUID) error {
	db := storage.GetDb()

	return db.Transaction(func(tx *gorm.DB) error {
		var cluster Cluster
		if err := tx.Where("id = ?", id).First(&cluster).Error; err != nil {
			return err
		}

		if err := tx.Model(&cluster).Association("Notifiers").Clear(); err != nil {
			return err
		}

		if err := tx.Where("cluster_id = ?", id).Delete(&PostgresqlCluster{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&Cluster{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *ClusterRepository) IsNotifierUsing(notifierID uuid.UUID) (bool, error) {
	var count int64
	if err := storage.GetDb().
		Table("cluster_notifiers").
		Where("notifier_id = ?", notifierID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ClusterRepository) FindAll() ([]*Cluster, error) {
	var clusters []*Cluster

	if err := storage.GetDb().
		Preload("Postgresql").
		Preload("Notifiers").
		Preload("BackupInterval").
		Preload("ExcludedDatabases").
		Order("name ASC").
		Find(&clusters).Error; err != nil {
		return nil, err
	}

	return clusters, nil
}

func (r *ClusterRepository) UpdateLastRunAt(id uuid.UUID, t time.Time) error {
	return storage.GetDb().Model(&Cluster{}).Where("id = ?", id).Update("last_run_at", t).Error
}
