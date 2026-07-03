package entity

import (
	"time"
)

type BadgeDefinition struct {
	ID            string
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

func NewBadgeDefinition(
	id string,
	code string,
	category string,
	name string,
	description string,
	iconKey string,
	criteriaType string,
	criteriaValue int,
	availableFrom time.Time,
	availableTo time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *BadgeDefinition {
	return &BadgeDefinition{
		ID:            id,
		Code:          code,
		Category:      category,
		Name:          name,
		Description:   description,
		IconKey:       iconKey,
		CriteriaType:  criteriaType,
		CriteriaValue: criteriaValue,
		AvailableFrom: availableFrom,
		AvailableTo:   availableTo,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
