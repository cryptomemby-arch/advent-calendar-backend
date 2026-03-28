package handlers

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/advent-calendar-backend/src/configs"
	"github.com/advent-calendar-backend/src/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const MonthlyPhotoLimit = 30

type PhotoHandler struct {
	DB     *gorm.DB
	Config *configs.Config
}

func NewPhotoHandler(db *gorm.DB, cfg *configs.Config) *PhotoHandler {
	return &PhotoHandler{DB: db, Config: cfg}
}

func (h *PhotoHandler) getUID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	uid, ok := val.(uint)
	return uid, ok
}

func (h *PhotoHandler) GetUploadSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		timestamp := time.Now().Unix()

		cloudName := h.Config.Photo.CloudName
		apiKey := h.Config.Photo.ApiKeyPhoto
		apiSecret := h.Config.Photo.ApiKeyPhoto
		folder := h.Config.Photo.FolderPhoto

		payload := fmt.Sprintf("folder=%s&timestamp=%d", folder, timestamp)

		hash := sha1.New()
		hash.Write([]byte(payload + apiSecret))
		signature := hex.EncodeToString(hash.Sum(nil))

		c.JSON(http.StatusOK, gin.H{
			"cloudName": cloudName,
			"apiKey":    apiKey,
			"folder":    folder,
			"timestamp": timestamp,
			"signature": signature,
		})
	}
}

func (h *PhotoHandler) CreatePhoto() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := h.getUID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not identified"})
			return
		}

		var count int64
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		h.DB.Model(&models.Photo{}).Where("user_id = ? AND created_at >= ?", userID, startOfMonth).Count(&count)

		if count >= MonthlyPhotoLimit {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Monthly limit exceeded",
				"limit": MonthlyPhotoLimit,
			})
			return
		}

		var input struct {
			PublicID  string    `json:"publicId" binding:"required"`
			SecureURL string    `json:"secureUrl" binding:"required,url"`
			Caption   string    `json:"caption"`
			Format    string    `json:"format"`
			Width     int       `json:"width"`
			Height    int       `json:"height"`
			Bytes     int64     `json:"bytes"`
			TakenAt   time.Time `json:"takenAt"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
			return
		}

		photo := models.Photo{
			UserID:    userID,
			PublicID:  strings.TrimSpace(input.PublicID),
			SecureURL: strings.TrimSpace(input.SecureURL),
			Caption:   strings.TrimSpace(input.Caption),
			Format:    input.Format,
			Width:     input.Width,
			Height:    input.Height,
			Bytes:     input.Bytes,
			TakenAt:   input.TakenAt,
			CreatedAt: time.Now(),
		}

		if err := h.DB.Create(&photo).Error; err != nil {
			zap.L().Error("DB Error saving photo", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save photo"})
			return
		}

		c.JSON(http.StatusCreated, photo)
	}
}

func (h *PhotoHandler) GetPhotos() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := h.getUID(c)
		monthParam := c.Query("month")

		var start time.Time
		if monthParam != "" {
			t, err := time.Parse("2006-01", monthParam)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format YYYY-MM"})
				return
			}
			start = t
		} else {
			now := time.Now()
			start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		}
		end := start.AddDate(0, 1, 0)

		var photos []models.Photo
		h.DB.Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, start, end).
			Order("created_at DESC").
			Find(&photos)

		c.JSON(http.StatusOK, photos)
	}
}

func (h *PhotoHandler) DeletePhoto() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := h.getUID(c)
		photoID := c.Param("photoId")

		result := h.DB.Where("id = ? AND user_id = ?", photoID, userID).Delete(&models.Photo{})

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found or access denied"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (h *PhotoHandler) GetLimitStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := h.getUID(c)

		var count int64
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		h.DB.Model(&models.Photo{}).Where("user_id = ? AND created_at >= ?", userID, startOfMonth).Count(&count)

		c.JSON(http.StatusOK, gin.H{
			"used":      count,
			"remaining": MonthlyPhotoLimit - int(count),
			"limit":     MonthlyPhotoLimit,
		})
	}
}
