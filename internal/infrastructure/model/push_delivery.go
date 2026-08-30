package model

import (
	"time"
)

type PushDelivery struct {
	ID             string `gorm:"primaryKey"`
	CreatedAt      time.Time
	UserId         string
	SubscriptionId string
	NotificationId string
	Campaign       string
	Status         string
	StatusCode     int
	DeliveredAt    *time.Time
	ClickedAt      *time.Time
}
