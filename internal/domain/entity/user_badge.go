package entity

import (
	"time"
)

type UserBadge struct {
	ID                string
	CreatedAt         time.Time
	UserId            string
	BadgeDefinitionId string
	RecordId          string
	AchievedAt        time.Time
}

func NewUserBadge(
	id string,
	createdAt time.Time,
	userId string,
	badgeDefinitionId string,
	recordId string,
	achievedAt time.Time,
) *UserBadge {
	return &UserBadge{
		ID:                id,
		CreatedAt:         createdAt,
		UserId:            userId,
		BadgeDefinitionId: badgeDefinitionId,
		RecordId:          recordId,
		AchievedAt:        achievedAt,
	}
}
