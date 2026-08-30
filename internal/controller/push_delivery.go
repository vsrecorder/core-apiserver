package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	PushDeliveriesPath = "/push_deliveries"
)

// PushDelivery は push の到達・タップの計測を受ける。
// 所有者チェックは usecase → repository の更新条件(user_id)で行い、他人の配達ログは 404 になる。
type PushDelivery struct {
	router  *gin.Engine
	usecase usecase.PushDeliveryInterface
}

func NewPushDelivery(
	router *gin.Engine,
	usecase usecase.PushDeliveryInterface,
) *PushDelivery {
	return &PushDelivery{router, usecase}
}

func (c *PushDelivery) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath + PushDeliveriesPath)
	r.POST(
		"/:id/delivered",
		authentication.RequiredAuthenticationMiddleware(),
		c.MarkDelivered,
	)
	r.POST(
		"/:id/clicked",
		authentication.RequiredAuthenticationMiddleware(),
		c.MarkClicked,
	)
}

func (c *PushDelivery) MarkDelivered(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	id := helper.GetId(ctx)

	if err := c.usecase.MarkDelivered(ctx.Request.Context(), uid, id); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *PushDelivery) MarkClicked(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	id := helper.GetId(ctx)

	if err := c.usecase.MarkClicked(ctx.Request.Context(), uid, id); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
