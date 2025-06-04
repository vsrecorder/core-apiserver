package dto

import "time"

type EnvironmentResponse struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}
