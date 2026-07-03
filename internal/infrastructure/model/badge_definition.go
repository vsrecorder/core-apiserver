package model

import (
	"time"
)

type BadgeDefinition struct {
	ID            string `gorm:"primaryKey"`
	Code          string
	Category      string
	Name          string
	Description   string
	IconKey       string
	CriteriaType  string
	CriteriaValue int
	AvailableFrom time.Time
	AvailableTo   time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
