package entity

import (
	"time"
)

type UserEnvironmentBadge struct {
	UserId         string
	EnvironmentId  string
	RecordId       string
	NotificationId string
	AchievedAt     time.Time
	CreatedAt      time.Time
}

func NewUserEnvironmentBadge(
	userId string,
	environmentId string,
	recordId string,
	notificationId string,
	achievedAt time.Time,
	createdAt time.Time,
) *UserEnvironmentBadge {
	return &UserEnvironmentBadge{
		UserId:         userId,
		EnvironmentId:  environmentId,
		RecordId:       recordId,
		NotificationId: notificationId,
		AchievedAt:     achievedAt,
		CreatedAt:      createdAt,
	}
}
