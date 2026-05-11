package models

import "time"

// PersonalPiece is a tune that is "really not a standard" – local compositions,
// rare pieces that maybe 2-3 people in the world play.  It lives entirely in
// the user's space and is NOT linked to the global jazz_standards table.
// Privacy is controlled per-piece.
type PersonalPiece struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	Title          string    `gorm:"not null" json:"title"`
	Composer       string    `gorm:"default:''" json:"composer,omitempty"`
	Style          JazzStyle `gorm:"type:varchar(50)" json:"style,omitempty"`
	Key            string    `gorm:"type:varchar(10);default:''" json:"key,omitempty"`
	Notes          string    `gorm:"type:text" json:"notes,omitempty"`
	IrealProLink   string    `gorm:"type:text;default:''" json:"ireal_pro_link,omitempty"`
	IsPublic       bool      `gorm:"default:false" json:"is_public"` // visible to other users?
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

func (PersonalPiece) TableName() string { return "personal_pieces" }
