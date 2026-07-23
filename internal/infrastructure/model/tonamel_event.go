package model

import "time"

// TonamelEvent は tonamel_events テーブルの1行。Tonamelの大会情報のキャッシュ。
// ID は records.tonamel_event_id と同じ Tonamel の大会ID。
type TonamelEvent struct {
	ID          string `gorm:"primaryKey"`
	Title       string
	Description string
	Image       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewTonamelEvent(
	id string,
	title string,
	description string,
	image string,
	createdAt time.Time,
	updatedAt time.Time,
) *TonamelEvent {
	return &TonamelEvent{
		ID:          id,
		Title:       title,
		Description: description,
		Image:       image,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
