package database

import (
	"database/sql"
	"fmt"

	"github.com/advent-calendar-backend/src/configs"
	"go.uber.org/zap"
)

func ConnectPostgreSql(cfg *configs.Config) *sql.DB {
	psqInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Username, cfg.Database.Password, cfg.Database.Dname)
	db, err := sql.Open("postgres", psqInfo)
	if err != nil {
		zap.L().Error("Error while connecting to database")
	}
	defer db.Close()

	return db
}
