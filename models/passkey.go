package models

import "time"

// PassKey is a named API key (bearer token) that a user can create for
// programmatic access (e.g. scripts, mobile apps).  Unlike the session
// token in the users table, a user can have multiple pass keys.
type PassKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Name      string    `gorm:"not null" json:"name"`          // human-readable label
	Token     string    `gorm:"uniqueIndex;not null" json:"-"` // returned only on creation
	TokenHint string    `gorm:"not null" json:"token_hint"`    // last 4 chars for display
	CreatedAt time.Time `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
}

func (PassKey) TableName() string { return "pass_keys" }
