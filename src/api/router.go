package api

import (
	"database/sql"
	"net/http"

	"github.com/advent-calendar-backend/src/api/handlers"
	"github.com/advent-calendar-backend/src/configs"
	"github.com/gin-gonic/gin"
)

func Router(r *gin.Engine, databaseConn *sql.DB, cfg *configs.Config) {

	r.Use(func(c *gin.Context) {
		c.Set("db", databaseConn)
		c.Next()
	})

	jwtByteKey := []byte(cfg.Jwt.My_super_secret_key)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", handlers.Login(jwtByteKey))
		authGroup.POST("/register", handlers.Register)

		// authGroup.GET("/google/login", handlers.GoogleLogin)
		// authGroup.GET("/google/callback", handlers.GoogleCallback)
		// authGroup.GET("/microsoft/login", handlers.microsoftLogin)
		// authGroup.GET("/microsoft/callback", handlers.microsoftCallback)

	}

	protected := r.Group("/")
	protected.Use(handlers.AuthMiddleware(jwtByteKey))

	apiGroup := protected.Group("/api")
	{
		// Пример того, как это будет выглядеть:
		// apiGroup.GET("/profile", handlers.GetProfile)
		apiGroup.GET("/profile", func(c *gin.Context) {})
		apiGroup.PUT("/profile", func(c *gin.Context) {})
		apiGroup.PUT("/profile/theme", func(c *gin.Context) {})
		apiGroup.GET("/profile/badges", func(c *gin.Context) {})
	}

	challengesGroup := protected.Group("/challenges")
	{
		challengesGroup.POST("", func(c *gin.Context) {})
		challengesGroup.GET("", func(c *gin.Context) {})
		challengesGroup.GET("/category/:category", func(c *gin.Context) {})
		challengesGroup.GET("/today", func(c *gin.Context) {})
		challengesGroup.GET("/today/preview", func(c *gin.Context) {})
	}

	userChallengesGroup := protected.Group("/user-challenges")
	{
		userChallengesGroup.POST("/join", func(c *gin.Context) {})
		userChallengesGroup.GET("/user/:userId", func(c *gin.Context) {})
		userChallengesGroup.GET("/user/:userId/status", func(c *gin.Context) {})
		userChallengesGroup.GET("/user/:userId/progress", func(c *gin.Context) {})
		userChallengesGroup.GET("/daily", func(c *gin.Context) {})
		userChallengesGroup.POST("/daily/confirm", func(c *gin.Context) {})
		userChallengesGroup.POST("/start", func(c *gin.Context) {})
		userChallengesGroup.GET("/challenge/:challengeId", func(c *gin.Context) {})
		userChallengesGroup.GET("/:id", func(c *gin.Context) {})
		userChallengesGroup.PUT("/:id/complete", func(c *gin.Context) {})
		userChallengesGroup.PUT("/:id/status", func(c *gin.Context) {})
		userChallengesGroup.DELETE("/clear-pending", func(c *gin.Context) {})
	}

	usersGroup := protected.Group("/users")
	{
		usersGroup.POST("", func(c *gin.Context) {})
		usersGroup.GET("/:id", func(c *gin.Context) {})
		usersGroup.GET("", func(c *gin.Context) {})
	}

	photosGroup := protected.Group("/photos")
	{
		photosGroup.GET("/upload-signature", func(c *gin.Context) {})
		photosGroup.POST("", func(c *gin.Context) {})
		photosGroup.GET("", func(c *gin.Context) {})
		photosGroup.DELETE("/:photoId", func(c *gin.Context) {})
	}

	recapGroup := protected.Group("/recap")
	{
		recapGroup.GET("/monthly", func(c *gin.Context) {})
	}

	timeCapsulesGroup := protected.Group("/time-capsules")
	{
		timeCapsulesGroup.POST("", func(c *gin.Context) {})
		timeCapsulesGroup.GET("/revealed", func(c *gin.Context) {})
		timeCapsulesGroup.GET("/pending", func(c *gin.Context) {})
	}

	pulseGroup := protected.Group("/pulse")
	{
		pulseGroup.GET("/today", func(c *gin.Context) {})
	}

	// Тестовый роут, чтобы быстро проверить, работает ли токен
	protected.GET("/test-auth", func(c *gin.Context) {
		username, _ := c.Get("username")
		c.JSON(http.StatusOK, gin.H{
			"message": "Ура! Токен работает, доступ разрешен.",
			"user":    username,
		})
	})
}
