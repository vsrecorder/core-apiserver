package dto

import (
	"time"
)

type UnofficialEventRequest struct {
	Title string    `json:"title"`
	Date  time.Time `json:"date"`
}

type UnofficialEventCreateRequest struct {
	UnofficialEventRequest
}

type UnofficialEventResponse struct {
	ID     string    `json:"id"`
	UserId string    `json:"user_id"`
	Title  string    `json:"title"`
	Date   time.Time `json:"date"`
}

type UnofficialEventGetByIdResponse struct {
	UnofficialEventResponse
}

type UnofficialEventCreateResponse struct {
	UnofficialEventResponse
}
