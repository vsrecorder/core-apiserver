package model

import (
	"time"
)

type UserBadge struct {
	ID                string `gorm:"primaryKey"`
	CreatedAt         time.Time
	UserId            string
	BadgeDefinitionId string
	RecordId          string
	AchievedAt        time.Time
}
