package models

import "time"

// Notification is sent to followers when a user reaches know_it or master proficiency.
type Notification struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RecipientID    uint      `gorm:"index;not null" json:"recipient_id"`
	ActorID        uint      `gorm:"not null" json:"actor_id"`
	ActionType     string    `gorm:"type:varchar(50)" json:"action_type"` // "learned" | "mastered"
	JazzStandardID *uint     `json:"jazz_standard_id,omitempty"`
	Read           bool      `gorm:"default:false" json:"read"`
	CreatedAt      time.Time `json:"created_at"`

	Recipient    *User         `gorm:"foreignKey:RecipientID;constraint:OnDelete:CASCADE" json:"recipient,omitempty"`
	Actor        *User         `gorm:"foreignKey:ActorID;constraint:OnDelete:CASCADE" json:"actor,omitempty"`
	JazzStandard *JazzStandard `gorm:"foreignKey:JazzStandardID;constraint:OnDelete:SET NULL" json:"jazz_standard,omitempty"`
}
