package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/advent-calendar-backend/src/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RecapPhotoPreviewDto struct {
	ID        uint      `json:"id"`
	SecureURL string    `json:"secureUrl"`
	Caption   string    `json:"caption"`
	CreatedAt time.Time `json:"createdAt"`
}

type MonthlyRecapResponseDto struct {
	Month            string                 `json:"month"`
	MonthStart       time.Time              `json:"monthStart"`
	MonthEnd         time.Time              `json:"monthEnd"`
	TotalAssigned    int64                  `json:"totalAssigned"`
	TotalCompleted   int64                  `json:"totalCompleted"`
	CurrentStreak    int                    `json:"currentStreak"`
	LongestStreak    int                    `json:"longestStreak"`
	TopCategory      string                 `json:"topCategory"`
	TopCategoryCount int64                  `json:"topCategoryCount"`
	CapsulesCreated  int64                  `json:"capsulesCreated"`
	CapsulesUnlocked int64                  `json:"capsulesUnlocked"`
	PhotosAdded      int64                  `json:"photosAdded"`
	RecentPhotos     []RecapPhotoPreviewDto `json:"recentPhotos"`
	GeneratedAt      time.Time              `json:"generatedAt"`
}

type RecapService struct {
	db *gorm.DB
}

func NewRecapService(db *gorm.DB) *RecapService {
	return &RecapService{db: db}
}

func (s *RecapService) GetMonthlyRecap(c *gin.Context) {
	userIdStr := c.Query("userId")
	if strings.TrimSpace(userIdStr) == "" {
		zap.L().Warn("Monthly recap request missing userId")
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}
	userId, err := strconv.ParseUint(userIdStr, 10, 64)
	if err != nil || userId == 0 {
		zap.L().Warn("Invalid userId in monthly recap", zap.String("userIdStr", userIdStr))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userId"})
		return
	}

	var user User
	if err := s.db.First(&user, userId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("User not found for monthly recap", zap.Uint64("userId", userId))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			zap.L().Error("Database error fetching user for recap", zap.Uint64("userId", userId), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	monthParam := c.Query("month")
	now := time.Now().UTC()
	var targetMonth time.Time
	if strings.TrimSpace(monthParam) == "" {
		targetMonth = now
	} else {
		parsed, err := time.Parse("2006-01", monthParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Use YYYY-MM"})
			return
		}
		targetMonth = parsed
	}

	monthStart := time.Date(targetMonth.Year(), targetMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	var totalAssigned int64
	err = s.db.Table("user_challenges").Where("user_id = ? AND created_at BETWEEN ? AND ?", userId, monthStart, monthEnd).Count(&totalAssigned).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting assigned challenges"})
		return
	}

	var totalCompleted int64
	err = s.db.Table("user_challenges").Where("user_id = ? AND status = ? AND (completion_time BETWEEN ? AND ? OR updated_at BETWEEN ? AND ?)", userId, "COMPLETED", monthStart, monthEnd, monthStart, monthEnd).Count(&totalCompleted).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting completed challenges"})
		return
	}

	// Determine top category
	topCategory := ""
	topCategoryCount := int64(0)

	type categoryCountRow struct {
		Category string
		Count    int64
	}
	var categoryCounts []categoryCountRow
	err = s.db.Table("user_challenges").Select("category, COUNT(*) as count").
		Where("user_id = ? AND status = ? AND (completion_time BETWEEN ? AND ? OR updated_at BETWEEN ? AND ?)", userId, "COMPLETED", monthStart, monthEnd, monthStart, monthEnd).
		Group("category").
		Order("count DESC, category ASC").
		Scan(&categoryCounts).Error
	if err != nil {
		// if no category field exists, ignore this metric
		topCategory = ""
	} else if len(categoryCounts) > 0 {
		topCategory = categoryCounts[0].Category
		topCategoryCount = categoryCounts[0].Count
	}

	var capsulesCreated int64
	err = s.db.Table("time_capsules").Where("user_id = ? AND created_at BETWEEN ? AND ?", userId, monthStart, monthEnd).Count(&capsulesCreated).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting created capsules"})
		return
	}

	var capsulesUnlocked int64
	err = s.db.Table("time_capsules").Where("user_id = ? AND unlocked_at BETWEEN ? AND ?", userId, monthStart, monthEnd).Count(&capsulesUnlocked).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting unlocked capsules"})
		return
	}

	var photosAdded int64
	err = s.db.Table("photos").Where("user_id = ? AND created_at BETWEEN ? AND ?", userId, monthStart, monthEnd).Count(&photosAdded).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting photos"})
		return
	}

	var recentPhotos []models.Photo
	err = s.db.Where("user_id = ? AND created_at BETWEEN ? AND ?", userId, monthStart, monthEnd).
		Order("created_at desc").
		Limit(8).
		Find(&recentPhotos).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error loading recent photos"})
		return
	}

	recentPreviews := make([]RecapPhotoPreviewDto, 0, len(recentPhotos))
	for _, p := range recentPhotos {
		recentPreviews = append(recentPreviews, RecapPhotoPreviewDto{
			ID:        p.ID,
			SecureURL: p.SecureURL,
			Caption:   p.Caption,
			CreatedAt: p.CreatedAt,
		})
	}

	// Streak stats
	type challengeTimeRow struct {
		CompletionTime *time.Time
		StartTime      *time.Time
	}
	var challengeTimes []challengeTimeRow
	err = s.db.Table("user_challenges").Select("completion_time, start_time").
		Where("user_id = ? AND status = ?", userId, "COMPLETED").
		Find(&challengeTimes).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error loading completed challenges"})
		return
	}

	completedDays := map[time.Time]struct{}{}
	for _, row := range challengeTimes {
		var d time.Time
		if row.CompletionTime != nil {
			d = row.CompletionTime.UTC()
		} else if row.StartTime != nil {
			d = row.StartTime.UTC()
		} else {
			continue
		}
		completedDays[time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)] = struct{}{}
	}

	currentStreak, longestStreak := 0, 0
	if len(completedDays) > 0 {
		days := make([]time.Time, 0, len(completedDays))
		for dt := range completedDays {
			days = append(days, dt)
		}
		sort.Slice(days, func(i, j int) bool {
			return days[i].Before(days[j])
		})

		longestStreak = 1
		running := 1
		for i := 1; i < len(days); i++ {
			if days[i].Sub(days[i-1]) == 24*time.Hour {
				running++
			} else {
				running = 1
			}
			if running > longestStreak {
				longestStreak = running
			}
		}

		lastDay := days[len(days)-1]
		gap := int(time.Now().UTC().Sub(lastDay).Hours() / 24)
		if gap == 0 || gap == 1 {
			currentStreak = 1
			for i := len(days) - 1; i > 0; i-- {
				if days[i].Sub(days[i-1]) == 24*time.Hour {
					currentStreak++
				} else {
					break
				}
			}
		}
	}

	c.JSON(http.StatusOK, MonthlyRecapResponseDto{
		Month:            monthStart.Format("2006-01"),
		MonthStart:       monthStart,
		MonthEnd:         monthEnd,
		TotalAssigned:    totalAssigned,
		TotalCompleted:   totalCompleted,
		CurrentStreak:    currentStreak,
		LongestStreak:    longestStreak,
		TopCategory:      topCategory,
		TopCategoryCount: topCategoryCount,
		CapsulesCreated:  capsulesCreated,
		CapsulesUnlocked: capsulesUnlocked,
		PhotosAdded:      photosAdded,
		RecentPhotos:     recentPreviews,
		GeneratedAt:      now,
	})

	zap.L().Info("Monthly recap generated", zap.Uint64("userId", userId), zap.String("month", monthParam), zap.Int64("totalCompleted", totalCompleted), zap.Int("currentStreak", currentStreak))
}
