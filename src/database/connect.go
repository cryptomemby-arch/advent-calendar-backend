package database

import (
	"fmt"

	"github.com/advent-calendar-backend/src/configs"
	"go.uber.org/zap"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgreSql(cfg *configs.Config) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.Database.Host, cfg.Database.Username, cfg.Database.Password, cfg.Database.Dname, cfg.Database.Port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Could not connect to database via GORM", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Fatal("Could not get underlying sql.DB instance", zap.Error(err))
	}

	err = sqlDB.Ping()
	if err != nil {
		zap.L().Fatal("Error while pinging database", zap.Error(err))
	}

	zap.L().Info("Successfully connected to database via GORM")
	return db
}
