package dto

import (
	"time"
)

type NotificationResponse struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	LinkUrl   string    `json:"link_url"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationsResponse struct {
	Notifications []*NotificationResponse `json:"notifications"`
}

type UnreadCountResponse struct {
	UnreadCount int `json:"unread_count"`
}
