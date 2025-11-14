package models

import (
	"time"
)

type UserStandard struct {
	UserID         uint      `gorm:"primaryKey;not null" json:"user_id"`
	JazzStandardID uint      `gorm:"primaryKey;not null" json:"jazz_standard_id"`
	CategoryID     *uint     `json:"category_id,omitempty"`
	Notes          string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`

	// Relationships
	User         *User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	JazzStandard *JazzStandard `gorm:"foreignKey:JazzStandardID;constraint:OnDelete:CASCADE" json:"jazz_standard,omitempty"`
	Category     *Category     `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`
}

func (UserStandard) TableName() string {
	return "user_jazz_standards"
}
