package model

import (
	"time"
)

type Designation struct {
	ID            string `gorm:"primaryKey"`
	Tier          int
	Code          string
	Emoji         string
	Name          string
	Description   string
	CriteriaType  string
	CriteriaValue int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
