package models

import (
	"time"
)

type Category struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index:idx_user_category" json:"user_id"`
	Name      string    `gorm:"not null;index:idx_user_category" json:"name"`
	Color     string    `gorm:"default:'#000000'" json:"color"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User          *User          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	UserStandards []UserStandard `gorm:"foreignKey:CategoryID" json:"-"`
}

func (Category) TableName() string {
	return "user_categories"
}
