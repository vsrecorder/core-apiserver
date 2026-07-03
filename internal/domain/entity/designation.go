package entity

import (
	"time"
)

type Designation struct {
	ID            string
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

func NewDesignation(
	id string,
	tier int,
	code string,
	emoji string,
	name string,
	description string,
	criteriaType string,
	criteriaValue int,
	createdAt time.Time,
	updatedAt time.Time,
) *Designation {
	return &Designation{
		ID:            id,
		Tier:          tier,
		Code:          code,
		Emoji:         emoji,
		Name:          name,
		Description:   description,
		CriteriaType:  criteriaType,
		CriteriaValue: criteriaValue,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
