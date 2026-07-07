package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func newNotificationResponse(n *entity.Notification) *dto.NotificationResponse {
	return &dto.NotificationResponse{
		ID:        n.ID,
		Category:  n.Category,
		Title:     n.Title,
		Body:      n.Body,
		LinkUrl:   n.LinkUrl,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}

func NewNotificationsResponse(
	notifications []*entity.Notification,
) *dto.NotificationsResponse {
	res := make([]*dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		res = append(res, newNotificationResponse(n))
	}

	return &dto.NotificationsResponse{
		Notifications: res,
	}
}

func NewUnreadCountResponse(
	unreadCount int,
) *dto.UnreadCountResponse {
	return &dto.UnreadCountResponse{
		UnreadCount: unreadCount,
	}
}
