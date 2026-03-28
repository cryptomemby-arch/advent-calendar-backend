package handlers

import (
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

type ChallengeService interface {
	CreateChallenge(challenge *Challenge) (*Challenge, error)
	GetAllChallenges() ([]Challenge, error)
	GetActiveChallengesByCategory(category ChallengeCategory) ([]Challenge, error)
}

type challengeService struct {
	db *gorm.DB
}

func NewChallengeService(db *gorm.DB) ChallengeService {
	return &challengeService{
		db: db,
	}
}

func (s *challengeService) CreateChallenge(challenge *Challenge) (*Challenge, error) {
	if err := s.db.Create(challenge).Error; err != nil {
		return nil, err
	}
	return challenge, nil
}

func (s *challengeService) GetAllChallenges() ([]Challenge, error) {
	var challenges []Challenge
	if err := s.db.Find(&challenges).Error; err != nil {
		return nil, err
	}
	return challenges, nil
}

func (s *challengeService) GetActiveChallengesByCategory(category ChallengeCategory) ([]Challenge, error) {
	var challenges []Challenge
	err := s.db.Where("category = ? AND active = ?", category, true).Find(&challenges).Error
	if err != nil {
		return nil, err
	}
	return challenges, nil
}
