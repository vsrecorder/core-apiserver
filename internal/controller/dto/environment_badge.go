package dto

import "time"

type EnvironmentBadgeResponse struct {
	EnvironmentId string     `json:"environment_id"`
	Title         string     `json:"title"`
	FromDate      time.Time  `json:"from_date"`
	ToDate        time.Time  `json:"to_date"`
	Achieved      bool       `json:"achieved"`
	AchievedAt    *time.Time `json:"achieved_at,omitempty"`
}

type UserEnvironmentBadgesResponse struct {
	UserId string                      `json:"user_id"`
	Badges []*EnvironmentBadgeResponse `json:"badges"`
}
