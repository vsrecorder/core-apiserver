package dto

type OldestRecordResponse struct {
	UserId    string  `json:"user_id"`
	DeckId    string  `json:"deck_id,omitempty"`
	EventDate *string `json:"event_date"`
}
