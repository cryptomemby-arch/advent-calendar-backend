package main

import (
	"github.com/advent-calendar-backend/src/api"
	"github.com/advent-calendar-backend/src/configs"
	"github.com/advent-calendar-backend/src/database"
	"github.com/advent-calendar-backend/src/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	logger := logger.InitLogger()
	defer logger.Sync()

	cfg := configs.LoadConfig()
	databaseConn := database.ConnectPostgreSql(cfg)

	r := gin.Default()
	api.Router(r, databaseConn, cfg)
}
