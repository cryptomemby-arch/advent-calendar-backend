package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TimeCapsule struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	RevealDate time.Time `gorm:"not null" json:"revealDate"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
	Revealed   bool      `gorm:"default:false" json:"revealed"`
}

type TimeCapsuleRequestDto struct {
	Content         string     `json:"content" binding:"required"`
	RevealDate      *time.Time `json:"revealDate"`
	DaysUntilReveal *int       `json:"daysUntilReveal"`
}

type TimeCapsuleResponseDto struct {
	ID         uint      `json:"id"`
	Content    string    `json:"content"`
	RevealDate time.Time `json:"revealDate"`
	CreatedAt  time.Time `json:"createdAt"`
	Revealed   bool      `json:"revealed"`
	Revealable bool      `json:"revealable"`
}

type TimeCapsuleService struct {
	db *gorm.DB
}

func NewTimeCapsuleService(db *gorm.DB) *TimeCapsuleService {
	return &TimeCapsuleService{db: db}
}

func (s *TimeCapsuleService) validateUser(userId uint) error {
	var count int64
	if err := s.db.Table("users").Where("id = ?", userId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *TimeCapsuleService) CreateCapsule(c *gin.Context) {
	var req TimeCapsuleRequestDto
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind time capsule JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		zap.L().Warn("Empty content in time capsule creation")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content cannot be empty"})
		return
	}

	userIDVal, ok := c.Get("userID")
	if !ok {
		zap.L().Error("User not identified in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not identified"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		zap.L().Error("Invalid user id type in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id type"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("User not found for time capsule creation", zap.Uint("userId", userID))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			zap.L().Error("Database error validating user", zap.Uint("userId", userID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	revealDate := time.Now().UTC().Add(7 * 24 * time.Hour)
	if req.RevealDate != nil {
		revealDate = req.RevealDate.UTC()
	} else if req.DaysUntilReveal != nil {
		if *req.DaysUntilReveal < 0 {
			zap.L().Warn("Negative daysUntilReveal", zap.Uint("userId", userID), zap.Int("days", *req.DaysUntilReveal))
			c.JSON(http.StatusBadRequest, gin.H{"error": "daysUntilReveal must be non-negative"})
			return
		}
		revealDate = time.Now().UTC().Add(time.Duration(*req.DaysUntilReveal) * 24 * time.Hour)
	}

	if revealDate.Before(time.Now().UTC()) {
		zap.L().Warn("Reveal date in past", zap.Uint("userId", userID), zap.Time("revealDate", revealDate))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reveal date must be in the future"})
		return
	}

	capsule := TimeCapsule{
		UserID:     userID,
		Content:    strings.TrimSpace(req.Content),
		RevealDate: revealDate,
		Revealed:   false,
	}

	if err := s.db.Create(&capsule).Error; err != nil {
		zap.L().Error("Failed to create time capsule", zap.Uint("userId", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create time capsule"})
		return
	}

	zap.L().Info("Time capsule created", zap.Uint("userId", userID), zap.Uint("capsuleId", capsule.ID), zap.Time("revealDate", revealDate))
	c.JSON(http.StatusCreated, s.toResponseDto(capsule))
}

func (s *TimeCapsuleService) GetRevealedCapsules(c *gin.Context) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		zap.L().Error("User not identified in context for revealed capsules")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not identified"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		zap.L().Error("Invalid user id type in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id type"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("User not found for revealed capsules", zap.Uint("userId", userID))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			zap.L().Error("Database error validating user", zap.Uint("userId", userID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	now := time.Now().UTC()
	var capsules []TimeCapsule
	if err := s.db.Where("user_id = ? AND reveal_date <= ?", userID, now).Find(&capsules).Error; err != nil {
		zap.L().Error("Failed to load revealed capsules", zap.Uint("userId", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load revealed capsules"})
		return
	}

	for _, cap := range capsules {
		if !cap.Revealed {
			cap.Revealed = true
			if err := s.db.Save(&cap).Error; err != nil {
				zap.L().Error("Failed to update capsule status", zap.Uint("capsuleId", cap.ID), zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update capsule status"})
				return
			}
		}
	}

	response := make([]TimeCapsuleResponseDto, 0, len(capsules))
	for _, cap := range capsules {
		response = append(response, s.toResponseDto(cap))
	}

	zap.L().Info("Revealed capsules retrieved", zap.Uint("userId", userID), zap.Int("count", len(response)))
	c.JSON(http.StatusOK, response)
}

func (s *TimeCapsuleService) GetPendingCapsules(c *gin.Context) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		zap.L().Error("User not identified in context for pending capsules")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not identified"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		zap.L().Error("Invalid user id type in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id type"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("User not found for pending capsules", zap.Uint("userId", userID))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			zap.L().Error("Database error validating user", zap.Uint("userId", userID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	now := time.Now().UTC()
	var capsules []TimeCapsule
	if err := s.db.Where("user_id = ? AND reveal_date > ?", userID, now).Find(&capsules).Error; err != nil {
		zap.L().Error("Failed to load pending capsules", zap.Uint("userId", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load pending capsules"})
		return
	}

	response := make([]TimeCapsuleResponseDto, 0, len(capsules))
	for _, cap := range capsules {
		response = append(response, s.toResponseDto(cap))
	}

	zap.L().Info("Pending capsules retrieved", zap.Uint("userId", userID), zap.Int("count", len(response)))
	c.JSON(http.StatusOK, response)
}

func (s *TimeCapsuleService) toResponseDto(capsule TimeCapsule) TimeCapsuleResponseDto {
	now := time.Now().UTC()
	return TimeCapsuleResponseDto{
		ID:         capsule.ID,
		Content:    capsule.Content,
		RevealDate: capsule.RevealDate,
		CreatedAt:  capsule.CreatedAt,
		Revealed:   capsule.Revealed,
		Revealable: !capsule.Revealed && !capsule.RevealDate.After(now),
	}
}
