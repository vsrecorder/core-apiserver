package model

import (
	"time"
)

type PushSubscription struct {
	ID            string `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RevokedAt     *time.Time
	UserId        string
	Endpoint      string
	P256dh        string
	Auth          string
	Platform      string
	FailureCount  int
	LastSuccessAt *time.Time
}
