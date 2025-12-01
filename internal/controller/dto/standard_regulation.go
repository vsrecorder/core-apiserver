package dto

import "time"

type StandardRegulationResponse struct {
	ID       string    `json:"id"`
	Marks    string    `json:"marks"`
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}
