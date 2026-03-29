package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ChallengeCategory string
type EnergyLevel string
type Culture string

const (
	Mental   ChallengeCategory = "MENTAL"
	Physical ChallengeCategory = "PHYSICAL"

	Low    EnergyLevel = "LOW"
	Medium EnergyLevel = "MEDIUM"
	High   EnergyLevel = "HIGH"

	Global Culture = "GLOBAL"
	Local  Culture = "LOCAL"
)

type Challenge struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Title       string            `gorm:"not null" json:"title"`
	Description string            `gorm:"not null" json:"description"`
	Category    ChallengeCategory `gorm:"type:varchar(50);not null" json:"category"`
	EnergyLevel EnergyLevel       `gorm:"type:varchar(50);not null" json:"energy_level"`
	Active      bool              `gorm:"default:false;not null" json:"active"`
	Culture     Culture           `gorm:"type:varchar(50);default:'GLOBAL';not null" json:"culture"`
}

type ChallengeService struct {
	db *gorm.DB
}

func NewChallengeService(db *gorm.DB) *ChallengeService {
	return &ChallengeService{
		db: db,
	}
}

func (s *ChallengeService) CreateChallenge(challenge *Challenge) (*Challenge, error) {
	if err := s.db.Create(challenge).Error; err != nil {
		return nil, err
	}
	return challenge, nil
}

func (s *ChallengeService) GetAllChallenges() ([]Challenge, error) {
	var challenges []Challenge
	if err := s.db.Find(&challenges).Error; err != nil {
		return nil, err
	}
	return challenges, nil
}

func (s *ChallengeService) GetActiveChallengesByCategory(category ChallengeCategory) ([]Challenge, error) {
	var challenges []Challenge
	err := s.db.Where("category = ? AND active = ?", category, true).Find(&challenges).Error
	if err != nil {
		return nil, err
	}
	return challenges, nil
}

func (s *ChallengeService) GetTodayChallenge() (*Challenge, error) {
	var challenge Challenge
	if err := s.db.Where("active = ?", true).First(&challenge).Error; err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *ChallengeService) GetTodayChallengePreview() (*Challenge, error) {
	return s.GetTodayChallenge()
}

func (s *ChallengeService) CreateChallengeHandler(c *gin.Context) {
	var challenge Challenge
	if err := c.ShouldBindJSON(&challenge); err != nil {
		zap.L().Error("Failed to bind challenge JSON", zap.Error(err))
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	created, err := s.CreateChallenge(&challenge)
	if err != nil {
		zap.L().Error("Failed to create challenge", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("Challenge created", zap.Uint("challengeId", created.ID))
	c.JSON(201, created)
}

func (s *ChallengeService) GetAllChallengesHandler(c *gin.Context) {
	challenges, err := s.GetAllChallenges()
	if err != nil {
		zap.L().Error("Failed to get all challenges", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("Retrieved all challenges", zap.Int("count", len(challenges)))
	c.JSON(200, challenges)
}

func (s *ChallengeService) GetActiveChallengesByCategoryHandler(c *gin.Context) {
	category := ChallengeCategory(c.Param("category"))
	challenges, err := s.GetActiveChallengesByCategory(category)
	if err != nil {
		zap.L().Error("Failed to get active challenges by category", zap.String("category", string(category)), zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("Retrieved active challenges by category", zap.String("category", string(category)), zap.Int("count", len(challenges)))
	c.JSON(200, challenges)
}

func (s *ChallengeService) GetTodayChallengeHandler(c *gin.Context) {
	challenge, err := s.GetTodayChallenge()
	if err != nil {
		zap.L().Error("Failed to get today's challenge", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("Retrieved today's challenge", zap.Uint("challengeId", challenge.ID))
	c.JSON(200, challenge)
}

func (s *ChallengeService) GetTodayChallengePreviewHandler(c *gin.Context) {
	challenge, err := s.GetTodayChallengePreview()
	if err != nil {
		zap.L().Error("Failed to get today's challenge preview", zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("Retrieved today's challenge preview", zap.Uint("challengeId", challenge.ID))
	c.JSON(200, challenge)
}
