package model

import "time"

type StandardRegulation struct {
	ID       string `gorm:"primaryKey"`
	Marks    string
	FromDate time.Time
	ToDate   time.Time
}

func NewStandardRegulation(
	id string,
	marks string,
	fromDate time.Time,
	toDate time.Time,
) *StandardRegulation {
	return &StandardRegulation{
		ID:       id,
		Marks:    marks,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
