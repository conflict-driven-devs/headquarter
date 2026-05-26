package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BootstrapToken struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	TokenHash  string     `json:"-" gorm:"not null;uniqueIndex"`
	InstanceID uuid.UUID  `gorm:"type:uuid;index" json:"instance_id"`
	Used       bool       `json:"used" gorm:"not null;default:false"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (t *BootstrapToken) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID = uuid.New()
	return nil
}
