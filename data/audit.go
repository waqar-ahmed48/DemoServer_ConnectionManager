package data

import (
	"time"

	"github.com/google/uuid"
)

// AuditRecord represents schema for Audit Trial kept by application manager
//
// swagger:model
type AuditRecord struct {
	ID           uuid.UUID            `json:"id" gorm:"primaryKey"`
	CreatedAt    time.Time            `json:"createdat" gorm:"autoCreateTime;index;not null"`
	RequestID    uuid.UUID            `json:"request_id" gorm:"index;not null"`
	ConnectionID uuid.UUID            `json:"connection_id" gorm:"index"`
	Action       uuid.UUID            `json:"action" gorm:"not null;index"`
	UserID       uuid.UUID            `json:"userid" validate:"required" gorm:"index;not null"`
	Status       ActionStatusTypeEnum `json:"status" validate:"required" gorm:"index;not null"`
	Details      string               `json:"details" validate:"required"`
}

// AuditRecordWrapper represents schema for response of command executed by application manager including output
//
// swagger:model
type AuditRecordWrapper struct {
	ID           uuid.UUID            `json:"id" gorm:"primaryKey"`
	CreatedAt    time.Time            `json:"createdat" gorm:"autoCreateTime;index;not null"`
	RequestID    uuid.UUID            `json:"request_id" gorm:"index;not null"`
	ConnectionID uuid.UUID            `json:"connection_id" gorm:"index"`
	Action       uuid.UUID            `json:"action" gorm:"not null;index"`
	UserID       uuid.UUID            `json:"userid" validate:"required" gorm:"index;not null"`
	Status       ActionStatusTypeEnum `json:"status" validate:"required" gorm:"index;not null"`
	Details      string               `json:"details" validate:"required"`
}
