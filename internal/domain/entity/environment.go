package entity

import "time"

type Environment struct {
	ID       string
	Title    string
	FromDate time.Time
	ToDate   time.Time
}

func NewEnvironment(
	id string,
	title string,
	fromDate time.Time,
	toDate time.Time,
) *Environment {
	return &Environment{
		ID:       id,
		Title:    title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
