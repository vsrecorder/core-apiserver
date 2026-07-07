package entity

import (
	"time"
)

type Notification struct {
	ID        string
	CreatedAt time.Time
	UserId    string
	Category  string
	Title     string
	Body      string
	LinkUrl   string
	IsRead    bool
	ReadAt    time.Time
}

func NewNotification(
	id string,
	createdAt time.Time,
	userId string,
	category string,
	title string,
	body string,
	linkUrl string,
) *Notification {
	return &Notification{
		ID:        id,
		CreatedAt: createdAt,
		UserId:    userId,
		Category:  category,
		Title:     title,
		Body:      body,
		LinkUrl:   linkUrl,
		IsRead:    false,
	}
}
