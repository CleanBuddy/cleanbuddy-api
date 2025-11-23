package store

import "time"

type AuthSession struct {
	ID string `gorm:"primaryKey;size:250;unique"`

	UserID string `gorm:"not null"`
	User   User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time `gorm:"autoCreateTime;not null"`
}
