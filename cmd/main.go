package main

import (
	"github.com/advent-calendar-backend/src/api"
	"github.com/advent-calendar-backend/src/configs"
	"github.com/advent-calendar-backend/src/database"
	"github.com/advent-calendar-backend/src/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	logger := logger.InitLogger()
	defer logger.Sync()

	cfg := configs.LoadConfig()
	databaseConn := database.ConnectPostgreSql(cfg)

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{cfg.Origin.OriginFront}
	r.Use(cors.New(config))
	r.Run()
	api.Router(r, databaseConn, cfg)
}
