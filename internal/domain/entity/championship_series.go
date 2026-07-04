package entity

import "time"

type ChampionshipSeries struct {
	ID       string
	Title    string
	FromDate time.Time
	ToDate   time.Time
}

func NewChampionshipSeries(
	id string,
	title string,
	fromDate time.Time,
	toDate time.Time,
) *ChampionshipSeries {
	return &ChampionshipSeries{
		ID:       id,
		Title:    title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
