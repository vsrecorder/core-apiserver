package model

import "time"

type CityleagueSchedule struct {
	ID       string `gorm:"primaryKey"`
	Title    string
	FromDate time.Time
	ToDate   time.Time
}

func NewCityleagueSchedule(
	id string,
	title string,
	fromDate time.Time,
	toDate time.Time,
) *CityleagueSchedule {
	return &CityleagueSchedule{
		ID:       id,
		Title:    title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
