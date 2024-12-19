package gorm

import (
	"time"

	"github.com/google/uuid"
)

// gorm.Model definition
type Model struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdat" gorm:"autoCreateTime;index;not null"`
	UpdatedAt time.Time `json:"updatedat" gorm:"autoUpdateTime;index"`
}
