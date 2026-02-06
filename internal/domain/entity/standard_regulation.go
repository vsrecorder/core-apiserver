package entity

import "time"

type StandardRegulation struct {
	ID       string
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
