package entity

import "time"

type User struct {
	ID          string
	CreatedAt   time.Time
	DisplayName string
	PhotoURL    string
}

func NewUser(
	id string,
	created_at time.Time,
	displayName string,
	photoURL string,
) *User {
	return &User{
		ID:          id,
		CreatedAt:   created_at,
		DisplayName: displayName,
		PhotoURL:    photoURL,
	}
}
