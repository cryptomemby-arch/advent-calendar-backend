package models

import (
	"time"
)

type Photo struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"userId"`
	PublicID  string    `gorm:"not null" json:"publicId"`
	SecureURL string    `gorm:"not null" json:"secureUrl"`
	Caption   string    `gorm:"type:text" json:"caption"`
	Format    string    `json:"format"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Bytes     int64     `json:"bytes"`
	TakenAt   time.Time `json:"takenAt"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}
