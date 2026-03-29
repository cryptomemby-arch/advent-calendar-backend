package main

import (
	"strings"

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
	origin := cfg.Origin.OriginFront
	if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
		origin = "http://" + origin
	}
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{origin}
	r.Use(cors.New(config))
	api.Router(r, databaseConn, cfg)
	r.Run(cfg.Origin.OriginBack)
}
