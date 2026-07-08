package model

import (
	"time"
)

type UserEnvironmentBadge struct {
	UserId         string `gorm:"primaryKey"`
	EnvironmentId  string `gorm:"primaryKey"`
	RecordId       string
	NotificationId string
	AchievedAt     time.Time
	CreatedAt      time.Time
}
