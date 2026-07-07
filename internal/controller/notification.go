package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	NotificationsPath = "/notifications"
)

type Notification struct {
	router  *gin.Engine
	usecase usecase.NotificationInterface
}

func NewNotification(
	router *gin.Engine,
	usecase usecase.NotificationInterface,
) *Notification {
	return &Notification{router, usecase}
}

func (c *Notification) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + NotificationsPath)

	r.GET(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.NotificationGetMiddleware(),
		c.GetByUID,
	)
	r.GET(
		"/unread_count",
		authentication.RequiredAuthenticationMiddleware(),
		c.CountUnread,
	)
	r.PATCH(
		"/:id/read",
		authentication.RequiredAuthenticationMiddleware(),
		c.MarkAsRead,
	)
	r.POST(
		"/read_all",
		authentication.RequiredAuthenticationMiddleware(),
		c.MarkAllAsRead,
	)
}

func (c *Notification) GetByUID(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	limit := helper.GetLimit(ctx)

	notifications, err := c.usecase.ListByUserId(context.Background(), uid, limit)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewNotificationsResponse(notifications)

	ctx.JSON(http.StatusOK, res)
}

func (c *Notification) CountUnread(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	count, err := c.usecase.CountUnreadByUserId(context.Background(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUnreadCountResponse(count)

	ctx.JSON(http.StatusOK, res)
}

func (c *Notification) MarkAsRead(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	id := helper.GetId(ctx)

	if err := c.usecase.MarkAsRead(context.Background(), uid, id); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *Notification) MarkAllAsRead(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	if err := c.usecase.MarkAllAsRead(context.Background(), uid); err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	ctx.Status(http.StatusNoContent)
}
