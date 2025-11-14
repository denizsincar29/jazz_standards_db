package models

import (
	"time"
)

// JazzStyle represents the style of jazz
type JazzStyle string

const (
	StyleDixieland  JazzStyle = "dixieland"
	StyleRagtime    JazzStyle = "ragtime"
	StyleBigBand    JazzStyle = "big_band"
	StyleBossaNova  JazzStyle = "bossa_nova"
	StyleSamba      JazzStyle = "samba"
	StyleLatin      JazzStyle = "latin"
	StyleLatinSwing JazzStyle = "latin_swing"
	StyleSwing      JazzStyle = "swing"
	StyleWaltz      JazzStyle = "waltz"
	StyleBebop      JazzStyle = "bebop"
	StyleModal      JazzStyle = "modal"
	StyleFree       JazzStyle = "free"
	StyleFusion     JazzStyle = "fusion"
)

// ValidStyles returns all valid jazz styles
func ValidStyles() []JazzStyle {
	return []JazzStyle{
		StyleDixieland, StyleRagtime, StyleBigBand, StyleBossaNova,
		StyleSamba, StyleLatin, StyleLatinSwing, StyleSwing,
		StyleWaltz, StyleBebop, StyleModal, StyleFree, StyleFusion,
	}
}

// IsValidStyle checks if a style is valid
func IsValidStyle(style string) bool {
	for _, s := range ValidStyles() {
		if string(s) == style {
			return true
		}
	}
	return false
}

type JazzStandard struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Title          string    `gorm:"uniqueIndex;not null" json:"title"`
	Composer       string    `gorm:"not null" json:"composer"`
	AdditionalNote string    `gorm:"type:text" json:"additional_note,omitempty"`
	Style          JazzStyle `gorm:"type:varchar(50);not null" json:"style"`
	CreatedBy      *uint     `gorm:"index" json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relationships
	Creator       *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	UserStandards []UserStandard `gorm:"foreignKey:JazzStandardID;constraint:OnDelete:CASCADE" json:"-"`
}

func (JazzStandard) TableName() string {
	return "jazz_standards"
}
