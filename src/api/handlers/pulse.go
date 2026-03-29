package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PulseResponseDto struct {
	Date             string `json:"date"`
	TotalUsers       int64  `json:"totalUsers"`
	CompletedCount   int64  `json:"completedCount"`
	LowMoodCount     int64  `json:"lowMoodCount"`
	NeutralMoodCount int64  `json:"neutralMoodCount"`
	HighMoodCount    int64  `json:"highMoodCount"`
}

type PulseService struct {
	db *gorm.DB
}

func NewPulseService(db *gorm.DB) *PulseService {
	return &PulseService{db: db}
}

func (s *PulseService) GetTodayPulse(c *gin.Context) {
	today := time.Now().UTC()
	startOfToday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	endOfToday := startOfToday.Add(24*time.Hour - time.Nanosecond)

	var totalUsers int64
	var completedCount int64
	var lowMoodCount int64
	var neutralMoodCount int64
	var highMoodCount int64

	if err := s.db.Table("user_challenges").Distinct("user_id").
		Where("updated_at BETWEEN ? AND ?", startOfToday, endOfToday).
		Count(&totalUsers).Error; err != nil {
		zap.L().Error("Error counting users for today", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting users for today"})
		return
	}

	if err := s.db.Table("user_challenges").
		Where("status = ? AND updated_at BETWEEN ? AND ?", "COMPLETED", startOfToday, endOfToday).Count(&completedCount).Error; err != nil {
		zap.L().Error("Error counting completed challenges today", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting completed challenges today"})
		return
	}

	if err := s.db.Table("user_challenges").
		Where("mood = ? AND updated_at BETWEEN ? AND ?", "LOW", startOfToday, endOfToday).Count(&lowMoodCount).Error; err != nil {
		zap.L().Error("Error counting low mood today", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting low mood today"})
		return
	}

	if err := s.db.Table("user_challenges").
		Where("mood = ? AND updated_at BETWEEN ? AND ?", "NEUTRAL", startOfToday, endOfToday).Count(&neutralMoodCount).Error; err != nil {
		zap.L().Error("Error counting neutral mood today", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting neutral mood today"})
		return
	}

	if err := s.db.Table("user_challenges").
		Where("mood = ? AND updated_at BETWEEN ? AND ?", "HIGH", startOfToday, endOfToday).Count(&highMoodCount).Error; err != nil {
		zap.L().Error("Error counting high mood today", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting high mood today"})
		return
	}

	if totalUsers == 0 {
		zap.L().Info("No users interacted today", zap.String("date", startOfToday.Format("2006-01-02")))
		c.JSON(http.StatusOK, PulseResponseDto{})
		return
	}

	zap.L().Info("Pulse data retrieved", zap.String("date", startOfToday.Format("2006-01-02")), zap.Int64("totalUsers", totalUsers), zap.Int64("completed", completedCount))
	c.JSON(http.StatusOK, PulseResponseDto{
		Date:             startOfToday.Format("2006-01-02"),
		TotalUsers:       totalUsers,
		CompletedCount:   completedCount,
		LowMoodCount:     lowMoodCount,
		NeutralMoodCount: neutralMoodCount,
		HighMoodCount:    highMoodCount,
	})
}
