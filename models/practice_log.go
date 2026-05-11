package models

import "time"

// PracticeLog records a practice session for a standard in a user's list.
type PracticeLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	JazzStandardID uint      `gorm:"not null;index" json:"jazz_standard_id"`
	DurationMin    int       `gorm:"default:0" json:"duration_min"` // minutes practiced
	Notes          string    `gorm:"type:text" json:"notes,omitempty"`
	PracticedAt    time.Time `gorm:"not null" json:"practiced_at"`
	CreatedAt      time.Time `json:"created_at"`

	User         *User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	JazzStandard *JazzStandard `gorm:"foreignKey:JazzStandardID;constraint:OnDelete:CASCADE" json:"jazz_standard,omitempty"`
}

func (PracticeLog) TableName() string { return "practice_logs" }
