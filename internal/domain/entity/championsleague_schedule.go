package entity

import "time"

type ChampionsleagueSchedule struct {
	ID       string
	Title    string
	FromDate time.Time
	ToDate   time.Time
}

func NewChampionsleagueSchedule(
	id string,
	title string,
	fromDate time.Time,
	toDate time.Time,
) *ChampionsleagueSchedule {
	return &ChampionsleagueSchedule{
		ID:       id,
		Title:    title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
