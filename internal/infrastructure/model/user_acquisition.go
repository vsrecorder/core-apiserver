package model

import (
	"time"
)

// UserAcquisition は user_acquisitions テーブル。
// 判明しなかった項目は NULL で持つ。空文字で埋めると Grafana 側の
// COALESCE(campaign, '(direct/unknown)') が効かず、「タグ無し」と区別できなくなる。
type UserAcquisition struct {
	UserId            string `gorm:"primaryKey"`
	Source            *string
	Medium            *string
	Campaign          *string
	Content           *string
	Referrer          *string
	LandingPath       *string
	LandingAt         *time.Time
	SourceInferredFlg bool
	SurveyAnswer      *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
