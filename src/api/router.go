package api

import (
	"github.com/advent-calendar-backend/src/api/handlers"
	"github.com/advent-calendar-backend/src/configs"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Router(r *gin.Engine, databaseConn *gorm.DB, cfg *configs.Config) {

	r.Use(func(c *gin.Context) {
		c.Set("db", databaseConn)
		c.Next()
	})

	jwtByteKey := []byte(cfg.Jwt.My_super_secret_key)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", handlers.Login(jwtByteKey))
		authGroup.POST("/register", handlers.Register)

		authGroup.GET("/google/login", handlers.GoogleLogin(cfg))
		authGroup.GET("/google/callback", handlers.GoogleCallback(cfg, jwtByteKey))
		authGroup.GET("/microsoft/login", handlers.MicrosoftLogin(cfg))
		authGroup.GET("/microsoft/callback", handlers.MicrosoftCallback(cfg, jwtByteKey))

	}

	protected := r.Group("/")
	protected.Use(handlers.AuthMiddleware(jwtByteKey))

	apiGroup := protected.Group("/api")
	{
		profileService := handlers.NewProfileService(databaseConn)
		apiGroup.GET("/profile", profileService.GetProfile)
		apiGroup.PUT("/profile", profileService.UpdateProfile)
		apiGroup.PUT("/profile/theme", profileService.UpdateThemePreference)
		apiGroup.GET("/profile/badges", profileService.GetProfileBadges)
	}

	challengesGroup := protected.Group("/challenges")
	{
		challengeService := handlers.NewChallengeService(databaseConn)
		challengesGroup.POST("", challengeService.CreateChallengeHandler)
		challengesGroup.GET("", challengeService.GetAllChallengesHandler)
		challengesGroup.GET("/category/:category", challengeService.GetActiveChallengesByCategoryHandler)
		challengesGroup.GET("/today", challengeService.GetTodayChallengeHandler)
		challengesGroup.GET("/today/preview", challengeService.GetTodayChallengePreviewHandler)
	}

	userChallengesService := handlers.NewUserChallengeService(databaseConn, handlers.NewBadgeService())
	userChallengesGroup := protected.Group("/user-challenges")
	{
		userChallengesGroup.POST("/join", userChallengesService.JoinChallenge)
		userChallengesGroup.GET("/user/:userId", userChallengesService.GetUserChallenges)
		userChallengesGroup.GET("/user/:userId/status", userChallengesService.GetUserChallengesByStatus)
		userChallengesGroup.GET("/user/:userId/progress", userChallengesService.GetUserProgress)
		userChallengesGroup.GET("/daily", userChallengesService.GetOrAssignDailyChallenge)
		userChallengesGroup.POST("/daily/confirm", userChallengesService.ConfirmDailyChallenge)
		userChallengesGroup.POST("/start", userChallengesService.StartChallenge)
		userChallengesGroup.GET("/challenge/:challengeId", userChallengesService.GetChallengeParticipants)
		userChallengesGroup.GET("/:id", userChallengesService.GetUserChallengeByID)
		userChallengesGroup.PUT("/:id/complete", userChallengesService.MarkAsCompleted)
		userChallengesGroup.PUT("/:id/status", userChallengesService.UpdateStatus)
		userChallengesGroup.DELETE("/clear-pending", userChallengesService.ClearPendingChallenges)
	}

	usersGroup := protected.Group("/users")
	{
		userService := handlers.NewUserService(databaseConn)
		usersGroup.POST("", userService.CreateUserHandler)
		usersGroup.GET("/:id", userService.GetUserByIDHandler)
		usersGroup.GET("", userService.GetUsersHandler)
	}

	photoH := handlers.NewPhotoHandler(databaseConn, cfg)

	photosGroup := protected.Group("/photos")
	{
		photosGroup.GET("/upload-signature", photoH.GetUploadSignature())
		photosGroup.GET("", photoH.GetPhotos())
		photosGroup.GET("/limit-status", photoH.GetLimitStatus())
		photosGroup.POST("", photoH.CreatePhoto())
		photosGroup.DELETE("/:photoId", photoH.DeletePhoto())
	}

	recapGroup := protected.Group("/recap")
	{
		recapService := handlers.NewRecapService(databaseConn)
		recapGroup.GET("/monthly", recapService.GetMonthlyRecap)
	}

	timeCapsulesGroup := protected.Group("/time-capsules")
	{
		timeCapsuleService := handlers.NewTimeCapsuleService(databaseConn)
		timeCapsulesGroup.POST("", timeCapsuleService.CreateCapsule)
		timeCapsulesGroup.GET("/revealed", timeCapsuleService.GetRevealedCapsules)
		timeCapsulesGroup.GET("/pending", timeCapsuleService.GetPendingCapsules)
	}

	pulseGroup := protected.Group("/pulse")
	{
		pulseService := handlers.NewPulseService(databaseConn)
		pulseGroup.GET("/today", pulseService.GetTodayPulse)
	}
}
