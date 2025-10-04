package dto

type EnvironmentResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
}
