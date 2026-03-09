package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func Router(r *gin.Engine, databaseConn *sql.DB) {
	authGroup := r.Group("/auth")
	{
		authGroup.GET("/me")
		authGroup.POST("/ensure-user")
	}

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/profile")
		apiGroup.PUT("/profile")
		apiGroup.PUT("/profile/theme")
		apiGroup.GET("/profile/badges")
	}

	challengesGroup := r.Group("/challenges")
	{
		challengesGroup.POST("")
		challengesGroup.GET("")
		challengesGroup.GET("/category/:category")
		challengesGroup.GET("/today")
		challengesGroup.GET("/today/preview")
	}

	userChallengesGroup := r.Group("/user-challenges")
	{
		userChallengesGroup.POST("/join")
		userChallengesGroup.GET("/user/:userId")
		userChallengesGroup.GET("/user/:userId/status")
		userChallengesGroup.GET("/user/:userId/progress")
		userChallengesGroup.GET("/daily")
		userChallengesGroup.POST("/daily/confirm")
		userChallengesGroup.POST("/start")
		userChallengesGroup.GET("/challenge/:challengeId")
		userChallengesGroup.GET("/:id")
		userChallengesGroup.PUT("/:id/complete")
		userChallengesGroup.PUT("/:id/status")
		userChallengesGroup.DELETE("/clear-pending")
	}

	usersGroup := r.Group("/users")
	{
		usersGroup.POST("")
		usersGroup.GET("/:id")
		usersGroup.GET("")
	}

	photosGroup := r.Group("/photos")
	{
		photosGroup.GET("/upload-signature")
		photosGroup.POST("")
		photosGroup.GET("")
		photosGroup.DELETE("/:photoId")
	}

	recapGroup := r.Group("/recap")
	{
		recapGroup.GET("/monthly")
	}

	timeCapsulesGroup := r.Group("/time-capsules")
	{
		timeCapsulesGroup.POST("")
		timeCapsulesGroup.GET("/revealed")
		timeCapsulesGroup.GET("/pending")
	}

	pulseGroup := r.Group("/pulse")
	{
		pulseGroup.GET("/today")
	}
}
