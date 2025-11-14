package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	Name         string    `gorm:"not null" json:"name"`
	PasswordHash string    `gorm:"not null" json:"-"`
	IsAdmin      bool      `gorm:"default:false" json:"is_admin"`
	Token        *string   `gorm:"unique" json:"token,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationships
	Categories     []Category      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"categories,omitempty"`
	UserStandards  []UserStandard  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	CreatedStandards []JazzStandard `gorm:"foreignKey:CreatedBy" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// BeforeCreate hook to ensure token is handled correctly
func (u *User) BeforeCreate(tx *gorm.DB) error {
	return nil
}
