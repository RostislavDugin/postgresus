package storage

import (
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"databasus-backend/internal/config"
	"databasus-backend/internal/util/logger"
)

var log = logger.GetLogger()

var db *gorm.DB

var initDb = sync.OnceFunc(loadDbs)

func GetDb() *gorm.DB {
	initDb()
	return db
}

func loadDbs() {
	LoadMainDb()
}

func LoadMainDb() {
	dbDsn := config.GetEnv().DatabaseDsn

	log.Info("connection to database")

	database, err := gorm.Open(postgres.Open(dbDsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		log.Error("error on connecting to database", "error", err)
		logger.ExitAfterFlush(1)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Error("error getting underlying sql.DB", "error", err)
		logger.ExitAfterFlush(1)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(10)

	db = database

	log.Info("main database connected successfully")
}
