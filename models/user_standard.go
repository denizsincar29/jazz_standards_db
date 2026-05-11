package models

import (
	"time"
)

// Proficiency levels for a user's known standard
type Proficiency string

const (
	ProficiencyBeginner  Proficiency = "beginner"
	ProficiencyLearning  Proficiency = "learning"
	ProficiencyKnowIt    Proficiency = "know_it"
	ProficiencyMaster    Proficiency = "master"
)

func IsValidProficiency(p string) bool {
	switch Proficiency(p) {
	case ProficiencyBeginner, ProficiencyLearning, ProficiencyKnowIt, ProficiencyMaster:
		return true
	}
	return false
}

type UserStandard struct {
	UserID         uint        `gorm:"primaryKey;not null" json:"user_id"`
	JazzStandardID uint        `gorm:"primaryKey;not null" json:"jazz_standard_id"`
	CategoryID     *uint       `json:"category_id,omitempty"`
	Notes          string      `gorm:"type:text" json:"notes,omitempty"`
	Proficiency    Proficiency `gorm:"type:varchar(20);default:'learning'" json:"proficiency"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`

	// Relationships
	User         *User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	JazzStandard *JazzStandard `gorm:"foreignKey:JazzStandardID;constraint:OnDelete:CASCADE" json:"jazz_standard,omitempty"`
	Category     *Category     `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`
}

func (UserStandard) TableName() string {
	return "user_jazz_standards"
}
