package storage

import (
	"databasus-backend/internal/config"
	"databasus-backend/internal/util/logger"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var log = logger.GetLogger()

var (
	db     *gorm.DB
	dbOnce sync.Once
)

func GetDb() *gorm.DB {
	dbOnce.Do(loadDbs)
	return db
}

func loadDbs() {
	LoadMainDb()
}

func LoadMainDb() {
	dbDsn := config.GetEnv().DatabaseDsn

	log.Info("Connection to database...")

	database, err := gorm.Open(postgres.Open(dbDsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		log.Error("error on connecting to database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Error("error getting underlying sql.DB", "error", err)
		os.Exit(1)
	}

	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(20)

	db = database

	log.Info("Main database connected successfully!")
}
