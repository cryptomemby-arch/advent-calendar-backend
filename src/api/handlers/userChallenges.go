package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CompletionStatus string
type Mood string

const (
	Assigned  CompletionStatus = "ASSIGNED"
	Completed CompletionStatus = "COMPLETED"

	MoodLow     Mood = "LOW"
	MoodNeutral Mood = "NEUTRAL"
	MoodHigh    Mood = "HIGH"
)

type UserChallenge struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	UserID         uint             `gorm:"index;not null" json:"userId"`
	User           User             `gorm:"foreignKey:UserID" json:"user"`
	ChallengeID    uint             `gorm:"index;not null" json:"challengeId"`
	Challenge      Challenge        `gorm:"foreignKey:ChallengeID" json:"challenge"`
	Status         CompletionStatus `gorm:"type:varchar(20);not null" json:"status"`
	Mood           Mood             `gorm:"type:varchar(20)" json:"mood"`
	StartTime      *time.Time       `json:"startTime"`
	CompletionTime *time.Time       `json:"completionTime"`
	CreatedAt      time.Time        `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time        `gorm:"autoUpdateTime" json:"updatedAt"`
}

type UserProgressDto struct {
	UserID         uint   `json:"userId"`
	Name           string `json:"name"`
	TotalAssigned  int64  `json:"totalAssigned"`
	TotalCompleted int64  `json:"totalCompleted"`
}

type DailyPreviewKey struct {
	UserID uint
	Date   string
	Mood   Mood
}

type UserChallengeService struct {
	db           *gorm.DB
	badgeService *BadgeService
	previewCache sync.Map // key: DailyPreviewKey, value: Challenge
}

func NewUserChallengeService(db *gorm.DB, badgeService *BadgeService) *UserChallengeService {
	return &UserChallengeService{db: db, badgeService: badgeService}
}

func (s *UserChallengeService) validateUser(userID uint) error {
	var count int64
	if err := s.db.Table("users").Where("id = ?", userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *UserChallengeService) validateChallenge(challengeID uint) (*Challenge, error) {
	var challenge Challenge
	if err := s.db.First(&challenge, challengeID).Error; err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *UserChallengeService) JoinChallenge(c *gin.Context) {
	var req struct {
		UserID      uint `json:"userId" binding:"required"`
		ChallengeID uint `json:"challengeId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind join challenge JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := s.validateUser(req.UserID); err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("User not found for join challenge", zap.Uint("userId", req.UserID))
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		zap.L().Error("Database error validating user for join", zap.Uint("userId", req.UserID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if _, err := s.validateChallenge(req.ChallengeID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var existing UserChallenge
	err := s.db.Where("user_id = ? AND challenge_id = ?", req.UserID, req.ChallengeID).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusOK, existing)
		return
	}
	if err != gorm.ErrRecordNotFound {
		zap.L().Error("Database error checking existing user challenge", zap.Uint("userId", req.UserID), zap.Uint("challengeId", req.ChallengeID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	userChallenge := UserChallenge{
		UserID:      req.UserID,
		ChallengeID: req.ChallengeID,
		Status:      Assigned,
	}

	if err := s.db.Create(&userChallenge).Error; err != nil {
		zap.L().Error("Failed to create user challenge", zap.Uint("userId", req.UserID), zap.Uint("challengeId", req.ChallengeID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join challenge"})
		return
	}

	zap.L().Info("User joined challenge", zap.Uint("userId", req.UserID), zap.Uint("challengeId", req.ChallengeID), zap.Uint("userChallengeId", userChallenge.ID))
	c.JSON(http.StatusCreated, userChallenge)
}

func (s *UserChallengeService) GetUserChallenges(c *gin.Context) {
	userID, err := s.parseUintParam(c, "userId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userId"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var userChallenges []UserChallenge
	if err := s.db.Preload("Challenge").Where("user_id = ?", userID).Find(&userChallenges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user challenges"})
		return
	}

	c.JSON(http.StatusOK, userChallenges)
}

func (s *UserChallengeService) GetChallengeParticipants(c *gin.Context) {
	challengeID, err := s.parseUintParam(c, "challengeId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid challengeId"})
		return
	}

	if _, err := s.validateChallenge(challengeID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var participants []UserChallenge
	if err := s.db.Preload("User").Where("challenge_id = ?", challengeID).Find(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load participants"})
		return
	}

	c.JSON(http.StatusOK, participants)
}

func (s *UserChallengeService) GetUserChallengeByID(c *gin.Context) {
	id, err := s.parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var userChallenge UserChallenge
	if err := s.db.Preload("User").Preload("Challenge").First(&userChallenge, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "UserChallenge not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, userChallenge)
}

func (s *UserChallengeService) MarkAsCompleted(c *gin.Context) {
	id, err := s.parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var uc UserChallenge
	if err := s.db.Preload("User").Preload("Challenge").First(&uc, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "UserChallenge not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if uc.Status != Assigned {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only ASSIGNED challenges can be marked as completed"})
		return
	}

	now := time.Now().UTC()
	uc.Status = Completed
	uc.CompletionTime = &now

	if err := s.db.Save(&uc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	if s.badgeService != nil {
		user := uc.User
		_, _ = s.badgeService.EvaluateAndAssignBadges(s.db, &user)
	}

	c.JSON(http.StatusOK, uc)
}

func (s *UserChallengeService) UpdateStatus(c *gin.Context) {
	id, err := s.parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var req struct {
		Status CompletionStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var uc UserChallenge
	if err := s.db.Preload("User").Preload("Challenge").First(&uc, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "UserChallenge not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	uc.Status = req.Status
	if req.Status == Completed {
		now := time.Now().UTC()
		uc.CompletionTime = &now
	}

	if err := s.db.Save(&uc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	if req.Status == Completed && s.badgeService != nil {
		user := uc.User
		_, _ = s.badgeService.EvaluateAndAssignBadges(s.db, &user)
	}

	c.JSON(http.StatusOK, uc)
}

func (s *UserChallengeService) GetUserChallengesByStatus(c *gin.Context) {
	userID, err := s.parseUintParam(c, "userId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userId"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	statusParam := c.Query("status")
	status := CompletionStatus(strings.ToUpper(statusParam))
	if status != Assigned && status != Completed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	var userChallenges []UserChallenge
	if err := s.db.Preload("Challenge").Where("user_id = ? AND status = ?", userID, status).Find(&userChallenges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user challenges"})
		return
	}

	c.JSON(http.StatusOK, userChallenges)
}

func (s *UserChallengeService) GetUserProgress(c *gin.Context) {
	userID, err := s.parseUintParam(c, "userId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userId"})
		return
	}

	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var totalAssigned int64
	if err := s.db.Model(&UserChallenge{}).Where("user_id = ?", userID).Count(&totalAssigned).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var totalCompleted int64
	if err := s.db.Model(&UserChallenge{}).Where("user_id = ? AND status = ?", userID, Completed).Count(&totalCompleted).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, UserProgressDto{
		UserID:         userID,
		Name:           user.Name,
		TotalAssigned:  totalAssigned,
		TotalCompleted: totalCompleted,
	})
}

func (s *UserChallengeService) getYesterdayCategory(userID uint, startOfToday time.Time) ChallengeCategory {
	var uc UserChallenge
	if err := s.db.Preload("Challenge").Where("user_id = ? AND start_time < ?", userID, startOfToday).Order("start_time DESC").First(&uc).Error; err != nil {
		return ""
	}
	return uc.Challenge.Category
}

func (s *UserChallengeService) filterOutCategory(challenges []Challenge, excluded ChallengeCategory) []Challenge {
	if excluded == "" {
		return challenges
	}
	out := make([]Challenge, 0, len(challenges))
	for _, c := range challenges {
		if c.Category != excluded {
			out = append(out, c)
		}
	}
	return out
}

func (s *UserChallengeService) selectDailyChallenge(user *User, mood Mood) (*Challenge, error) {
	energyLevel := "MEDIUM"
	switch mood {
	case MoodLow:
		energyLevel = "LOW"
	case MoodHigh:
		energyLevel = "HIGH"
	case MoodNeutral:
		energyLevel = "MEDIUM"
	}

	var candidateChallenges []Challenge
	if err := s.db.Where("energy_level = ? AND active = ?", energyLevel, true).Find(&candidateChallenges).Error; err != nil {
		return nil, err
	}

	if len(candidateChallenges) == 0 {
		if err := s.db.Where("active = ?", true).Find(&candidateChallenges).Error; err != nil {
			return nil, err
		}
	}

	if len(candidateChallenges) == 0 {
		return nil, nil
	}

	startOfToday := time.Now().UTC().Truncate(24 * time.Hour)
	yesterdayCategory := s.getYesterdayCategory(user.ID, startOfToday)
	available := s.filterOutCategory(candidateChallenges, yesterdayCategory)
	if len(available) == 0 {
		available = candidateChallenges
	}

	rand.Seed(time.Now().UnixNano())
	picked := available[rand.Intn(len(available))]
	return &picked, nil
}

func (s *UserChallengeService) getDailyChallenge(c *gin.Context, mood Mood) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	startOfToday := time.Now().UTC().Truncate(24 * time.Hour)
	var existing UserChallenge
	if err := s.db.Preload("Challenge").Where("user_id = ? AND start_time >= ? AND status = ?", userID, startOfToday, Assigned).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, existing)
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	user := User{}
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	challenge, err := s.selectDailyChallenge(&user, mood)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed selecting challenge"})
		return
	}
	if challenge == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No challenges available"})
		return
	}

	userChallenge := UserChallenge{
		UserID:      userID,
		ChallengeID: challenge.ID,
		Status:      Assigned,
		Mood:        mood,
		StartTime:   nil,
	}
	if err := s.db.Create(&userChallenge).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create daily challenge"})
		return
	}

	c.JSON(http.StatusOK, userChallenge)
}

func (s *UserChallengeService) GetOrAssignDailyChallenge(c *gin.Context) {
	moodParam := strings.ToUpper(strings.TrimSpace(c.Query("mood")))
	mood := MoodNeutral
	if moodParam != "" {
		mood = Mood(moodParam)
	}
	if mood != MoodLow && mood != MoodNeutral && mood != MoodHigh {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mood"})
		return
	}

	s.getDailyChallenge(c, mood)
}

func (s *UserChallengeService) PreviewDailyChallenge(c *gin.Context) {
	moodParam := strings.ToUpper(strings.TrimSpace(c.Query("mood")))
	mood := MoodNeutral
	if moodParam != "" {
		mood = Mood(moodParam)
	}
	if mood != MoodLow && mood != MoodNeutral && mood != MoodHigh {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mood"})
		return
	}

	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	startOfToday := time.Now().UTC().Truncate(24 * time.Hour)
	var existing UserChallenge
	if err := s.db.Preload("Challenge").Where("user_id = ? AND start_time >= ? AND status = ?", userID, startOfToday, Assigned).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, existing)
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	user := User{}
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	challenge, err := s.selectDailyChallenge(&user, mood)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed selecting challenge"})
		return
	}
	if challenge == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No challenges available"})
		return
	}

	key := DailyPreviewKey{UserID: userID, Date: startOfToday.Format("2006-01-02"), Mood: mood}
	s.previewCache.Store(key, *challenge)

	c.JSON(http.StatusOK, challenge)
}

func (s *UserChallengeService) ConfirmDailyChallenge(c *gin.Context) {
	var req struct {
		ChallengeID uint `json:"challengeId" binding:"required"`
		Mood        Mood `json:"mood" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Mood != MoodLow && req.Mood != MoodNeutral && req.Mood != MoodHigh {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mood"})
		return
	}

	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	startOfToday := time.Now().UTC().Truncate(24 * time.Hour)
	var existing UserChallenge
	if err := s.db.Preload("Challenge").Where("user_id = ? AND start_time >= ? AND status = ?", userID, startOfToday, Assigned).First(&existing).Error; err == nil {
		existing.Mood = req.Mood
		if err := s.db.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing challenge"})
			return
		}
		c.JSON(http.StatusOK, existing)
		return
	} else if err != gorm.ErrRecordNotFound {
		zap.L().Error("Database error in confirm daily challenge", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	key := DailyPreviewKey{UserID: userID, Date: startOfToday.Format("2006-01-02"), Mood: req.Mood}
	value, exists := s.previewCache.Load(key)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No preview challenge found. Please preview first."})
		return
	}

	expected, ok := value.(Challenge)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Preview state invalid"})
		return
	}

	if expected.ID != req.ChallengeID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Preview challenge mismatch"})
		return
	}

	userChallenge := UserChallenge{
		UserID:      userID,
		ChallengeID: req.ChallengeID,
		Status:      Assigned,
		Mood:        req.Mood,
		StartTime:   ptrTime(time.Now().UTC()),
	}
	if err := s.db.Create(&userChallenge).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm challenge"})
		return
	}

	s.previewCache.Delete(key)
	c.JSON(http.StatusOK, userChallenge)
}

func (s *UserChallengeService) ClearPendingChallenges(c *gin.Context) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	result := s.db.Where("user_id = ? AND status = ?", userID, Assigned).Delete(&UserChallenge{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear pending challenges"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

func (s *UserChallengeService) StartChallenge(c *gin.Context) {
	var req struct {
		ChallengeID uint `json:"challengeId" binding:"required"`
		Mood        Mood `json:"mood" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Mood != MoodLow && req.Mood != MoodNeutral && req.Mood != MoodHigh {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mood"})
		return
	}

	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user id"})
		return
	}

	if err := s.validateUser(userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if _, err := s.validateChallenge(req.ChallengeID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Challenge not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	startOfToday := time.Now().UTC().Truncate(24 * time.Hour)
	var existing UserChallenge
	if err := s.db.Where("user_id = ? AND challenge_id = ? AND start_time >= ? AND status = ?", userID, req.ChallengeID, startOfToday, Assigned).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, existing)
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	now := time.Now().UTC()
	userChallenge := UserChallenge{
		UserID:      userID,
		ChallengeID: req.ChallengeID,
		Status:      Assigned,
		Mood:        req.Mood,
		StartTime:   &now,
	}
	if err := s.db.Create(&userChallenge).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start challenge"})
		return
	}

	c.JSON(http.StatusCreated, userChallenge)
}

func (s *UserChallengeService) parseUintParam(c *gin.Context, name string) (uint, error) {
	v := c.Param(name)
	if v == "" {
		return 0, strconv.ErrSyntax
	}
	id64, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id64), nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
