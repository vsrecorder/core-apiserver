package entity

import "time"

type User struct {
	ID        string
	CreatedAt time.Time
	Name      string
	ImageURL  string
}

func NewUser(
	id string,
	createdAt time.Time,
	Name string,
	ImageURL string,
) *User {
	return &User{
		ID:        id,
		CreatedAt: createdAt,
		Name:      Name,
		ImageURL:  ImageURL,
	}
}
