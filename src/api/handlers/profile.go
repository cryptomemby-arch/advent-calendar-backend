package handlers

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Badge struct {
	ID          string `gorm:"primaryKey" json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Criteria    string `json:"criteria"`
}

type ProfileBadgeDto struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Icon          string `json:"icon"`
	Criteria      string `json:"criteria"`
	Earned        bool   `json:"earned"`
	NewlyUnlocked bool   `json:"newlyUnlocked"`
}

type ProfileResponseDto struct {
	ID              uint            `json:"id"`
	Name            string          `json:"name"`
	Email           string          `json:"email"`
	Avatar          string          `json:"avatar"`
	Streak          int             `json:"streak"`
	TotalPoints     int64           `json:"totalPoints"`
	EarnedBadgeIDs  []string        `json:"earnedBadgeIds"`
	ThemePreference ThemePreference `json:"themePreference"`
	NewlyUnlocked   []string        `json:"newlyUnlocked"`
}

type ProfileBadgesResponseDto struct {
	Badges        []ProfileBadgeDto `json:"badges"`
	EarnedBadges  []ProfileBadgeDto `json:"earnedBadges"`
	NewlyUnlocked []string          `json:"newlyUnlocked"`
}

type ProfileUpdateRequestDto struct {
	Name   *string `json:"name"`
	Avatar *string `json:"avatar"`
}

type ThemePreferenceUpdateRequestDto struct {
	ThemePreference *ThemePreference `json:"themePreference" binding:"required"`
}

type BadgeService struct {
	allBadges []Badge
}

func NewBadgeService() *BadgeService {
	return &BadgeService{
		allBadges: []Badge{
			{ID: "streak_1", Title: "First Streak", Description: "Complete 1 day", Icon: "🔥", Criteria: "streak >= 1"},
			{ID: "streak_3", Title: "3-Day Streak", Description: "Complete 3 day streak", Icon: "⚡", Criteria: "streak >= 3"},
			{ID: "points_100", Title: "100 Points", Description: "Earn 100 points", Icon: "💎", Criteria: "totalPoints >= 100"},
		},
	}
}

func (s *BadgeService) GetAllBadges() []Badge {
	return s.allBadges
}

func (s *BadgeService) GetEarnedBadgeIDs(user *User) []string {
	if user.EarnedBadgeIDs == nil {
		return []string{}
	}
	return user.EarnedBadgeIDs
}

func (s *BadgeService) EvaluateAndAssignBadges(db *gorm.DB, user *User) ([]Badge, error) {
	if user.EarnedBadgeIDs == nil {
		user.EarnedBadgeIDs = []string{}
	}
	earned := make(map[string]bool, len(user.EarnedBadgeIDs))
	for _, id := range user.EarnedBadgeIDs {
		earned[id] = true
	}

	newlyUnlocked := make([]Badge, 0)

	for _, badge := range s.allBadges {
		shouldEarn := false
		switch badge.ID {
		case "streak_1":
			shouldEarn = user.Streak >= 1
		case "streak_3":
			shouldEarn = user.Streak >= 3
		case "points_100":
			shouldEarn = user.TotalPoints >= 100
		}
		if shouldEarn && !earned[badge.ID] {
			user.EarnedBadgeIDs = append(user.EarnedBadgeIDs, badge.ID)
			newlyUnlocked = append(newlyUnlocked, badge)
			earned[badge.ID] = true
		}
	}

	if len(newlyUnlocked) > 0 {
		if err := db.Save(user).Error; err != nil {
			return nil, err
		}
	}

	return newlyUnlocked, nil
}

type ProfileService struct {
	db           *gorm.DB
	badgeService *BadgeService
}

func NewProfileService(db *gorm.DB) *ProfileService {
	return &ProfileService{
		db:           db,
		badgeService: NewBadgeService(),
	}
}

func (s *ProfileService) getCurrentUser(c *gin.Context) (*User, error) {
	userIDVal, ok := c.Get("userID")
	if !ok {
		return nil, gin.Error{Err: gorm.ErrRecordNotFound, Type: gin.ErrorTypePublic}
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		return nil, gin.Error{Err: gorm.ErrRecordNotFound, Type: gin.ErrorTypePublic}
	}
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *ProfileService) toProfileResponse(user *User, newlyUnlocked []Badge) ProfileResponseDto {
	theme := user.ThemePreference
	if theme == "" {
		theme = ThemePreferenceSystem
	}

	ids := make([]string, len(newlyUnlocked))
	for i, badge := range newlyUnlocked {
		ids[i] = badge.ID
	}
	sort.Strings(ids)

	return ProfileResponseDto{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Avatar:          user.Avatar,
		Streak:          user.Streak,
		TotalPoints:     user.TotalPoints,
		EarnedBadgeIDs:  s.badgeService.GetEarnedBadgeIDs(user),
		ThemePreference: theme,
		NewlyUnlocked:   ids,
	}
}

func (s *ProfileService) toProfileBadgeDto(badge Badge, earned, newlyUnlocked bool) ProfileBadgeDto {
	return ProfileBadgeDto{
		ID:            badge.ID,
		Title:         badge.Title,
		Description:   badge.Description,
		Icon:          badge.Icon,
		Criteria:      badge.Criteria,
		Earned:        earned,
		NewlyUnlocked: newlyUnlocked,
	}
}

func (s *ProfileService) GetProfile(c *gin.Context) {
	user, err := s.getCurrentUser(c)
	if err != nil {
		zap.L().Error("Failed to get current user for profile", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newlyUnlocked, err := s.badgeService.EvaluateAndAssignBadges(s.db, user)
	if err != nil {
		zap.L().Error("Badge evaluation failed for user", zap.Uint("userId", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Badge evaluation failed"})
		return
	}

	zap.L().Info("Profile retrieved", zap.Uint("userId", user.ID), zap.Int("newBadges", len(newlyUnlocked)))
	c.JSON(http.StatusOK, s.toProfileResponse(user, newlyUnlocked))
}

func (s *ProfileService) UpdateProfile(c *gin.Context) {
	var req ProfileUpdateRequestDto
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind profile update JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Name == nil && req.Avatar == nil {
		zap.L().Warn("Profile update request with no fields")
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field (name/avatar) must be provided"})
		return
	}

	user, err := s.getCurrentUser(c)
	if err != nil {
		zap.L().Error("Failed to get current user for profile update", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if req.Name != nil {
		trim := *req.Name
		if trim == "" {
			zap.L().Warn("Attempt to set blank name", zap.Uint("userId", user.ID))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name cannot be blank"})
			return
		}
		user.Name = trim
	}

	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}

	if err := s.db.Save(user).Error; err != nil {
		zap.L().Error("Failed to save profile", zap.Uint("userId", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile"})
		return
	}

	newlyUnlocked, err := s.badgeService.EvaluateAndAssignBadges(s.db, user)
	if err != nil {
		zap.L().Error("Badge evaluation failed after profile update", zap.Uint("userId", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Badge evaluation failed"})
		return
	}

	zap.L().Info("Profile updated", zap.Uint("userId", user.ID), zap.Bool("nameUpdated", req.Name != nil), zap.Bool("avatarUpdated", req.Avatar != nil), zap.Int("newBadges", len(newlyUnlocked)))
	c.JSON(http.StatusOK, s.toProfileResponse(user, newlyUnlocked))
}

func (s *ProfileService) UpdateThemePreference(c *gin.Context) {
	var req ThemePreferenceUpdateRequestDto
	if err := c.ShouldBindJSON(&req); err != nil || req.ThemePreference == nil {
		zap.L().Error("Failed to bind theme preference update JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "themePreference is required"})
		return
	}

	user, err := s.getCurrentUser(c)
	if err != nil {
		zap.L().Error("Failed to get current user for theme update", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.ThemePreference = *req.ThemePreference
	if err := s.db.Save(user).Error; err != nil {
		zap.L().Error("Failed to save theme preference", zap.Uint("userId", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save theme preference"})
		return
	}

	newlyUnlocked, err := s.badgeService.EvaluateAndAssignBadges(s.db, user)
	if err != nil {
		zap.L().Error("Badge evaluation failed after theme update", zap.Uint("userId", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Badge evaluation failed"})
		return
	}

	zap.L().Info("Theme preference updated", zap.Uint("userId", user.ID), zap.String("theme", string(user.ThemePreference)), zap.Int("newBadges", len(newlyUnlocked)))
	c.JSON(http.StatusOK, s.toProfileResponse(user, newlyUnlocked))
}

func (s *ProfileService) GetProfileBadges(c *gin.Context) {
	user, err := s.getCurrentUser(c)
	if err != nil {
		zap.L().Error("Failed to get current user for badges", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newlyUnlocked, err := s.badgeService.EvaluateAndAssignBadges(s.db, user)
	if err != nil {
		zap.L().Error("Badge evaluation failed for user", zap.Uint("userId", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Badge evaluation failed"})
		return
	}

	allBadges := s.badgeService.GetAllBadges()
	earnedSet := map[string]bool{}
	for _, id := range s.badgeService.GetEarnedBadgeIDs(user) {
		earnedSet[id] = true
	}
	newlySet := map[string]bool{}
	for _, b := range newlyUnlocked {
		newlySet[b.ID] = true
	}

	badgeDtos := make([]ProfileBadgeDto, 0, len(allBadges))
	for _, badge := range allBadges {
		badgeDtos = append(badgeDtos, s.toProfileBadgeDto(badge, earnedSet[badge.ID], newlySet[badge.ID]))
	}

	earnedBadges := make([]ProfileBadgeDto, 0)
	for _, badge := range badgeDtos {
		if badge.Earned {
			earnedBadges = append(earnedBadges, badge)
		}
	}

	newlyIds := make([]string, 0, len(newlySet))
	for id := range newlySet {
		newlyIds = append(newlyIds, id)
	}
	sort.Strings(newlyIds)

	zap.L().Info("Profile badges retrieved", zap.Uint("userId", user.ID), zap.Int("totalBadges", len(badgeDtos)), zap.Int("earnedBadges", len(earnedBadges)), zap.Int("newlyUnlocked", len(newlyIds)))
	c.JSON(http.StatusOK, ProfileBadgesResponseDto{
		Badges:        badgeDtos,
		EarnedBadges:  earnedBadges,
		NewlyUnlocked: newlyIds,
	})
}
