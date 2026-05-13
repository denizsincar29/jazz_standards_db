package models

import "time"

// PassKey stores a WebAuthn credential for a user (biometric login via Face ID /
// Touch ID / Windows Hello / hardware key).
//
// Legacy fields (Token, TokenHint) are kept so that existing rows are not
// broken; new rows use the WebAuthn credential fields and leave Token empty.
type PassKey struct {
	ID        uint       `gorm:"primaryKey"        json:"id"`
	UserID    uint       `gorm:"not null;index"    json:"user_id"`
	Name      string     `gorm:"not null"          json:"name"` // human-readable label
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`

	// ── WebAuthn credential fields ────────────────────────────────────────
	// CredentialID is the opaque identifier issued by the authenticator.
	CredentialID []byte `gorm:"uniqueIndex;size:1024" json:"-"`
	// PublicKey is the COSE-encoded public key.
	PublicKey []byte `gorm:"type:bytea" json:"-"`
	// AAGUID identifies the authenticator model (16 bytes).
	AAGUID []byte `gorm:"type:bytea" json:"-"`
	// SignCount is the authenticator's monotonic counter (replay protection).
	SignCount uint32 `gorm:"default:0" json:"-"`
	// Transports (comma-separated): "internal", "usb", "nfc", "ble", etc.
	Transports string `gorm:"type:text;default:''" json:"-"`

	// ── Legacy API-key fields (kept for backward compat; empty on new rows) ─
	Token     string `gorm:"uniqueIndex;default:''" json:"-"`
	TokenHint string `gorm:"default:''"            json:"token_hint,omitempty"`

	User *User `gorm:"foreignKey:UserID" json:"-"`
}

func (PassKey) TableName() string { return "pass_keys" }
