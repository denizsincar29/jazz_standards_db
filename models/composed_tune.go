package models

import "time"

// ComposedTune is something a user composed themselves.  It has a
// shareable URL token so other users can view it and optionally add it
// to their own lists as a PersonalPiece.
type ComposedTune struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	ShareID      string    `gorm:"uniqueIndex;not null" json:"share_id"` // URL-safe random token
	Title        string    `gorm:"not null" json:"title"`
	Composer     string    `gorm:"not null" json:"composer"` // usually the user's name
	Style        JazzStyle `gorm:"type:varchar(50)" json:"style,omitempty"`
	Key          string    `gorm:"type:varchar(10);default:''" json:"key,omitempty"`
	Description  string    `gorm:"type:text" json:"description,omitempty"`
	IrealProLink string    `gorm:"type:text;default:''" json:"ireal_pro_link,omitempty"`
	IsPublic     bool      `gorm:"default:true" json:"is_public"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (ComposedTune) TableName() string { return "composed_tunes" }
