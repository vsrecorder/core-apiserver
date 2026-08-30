package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	PushSubscriptionsPath = "/push_subscriptions"
)

type PushSubscription struct {
	router  *gin.Engine
	usecase usecase.PushSubscriptionInterface
}

func NewPushSubscription(
	router *gin.Engine,
	usecase usecase.PushSubscriptionInterface,
) *PushSubscription {
	return &PushSubscription{router, usecase}
}

func (c *PushSubscription) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	// 書き込みなので uid はパスパラメータではなく認証済みトークンから取る(activity と同じ)。
	r.POST(
		PushSubscriptionsPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.PushSubscriptionCreateMiddleware(),
		c.Subscribe,
	)
	r.DELETE(
		PushSubscriptionsPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.PushSubscriptionDeleteMiddleware(),
		c.Unsubscribe,
	)
}

func (c *PushSubscription) Subscribe(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	req := helper.GetPushSubscriptionCreateRequest(ctx)

	if err := c.usecase.Subscribe(ctx.Request.Context(), uid, req.Endpoint, req.Keys.P256dh, req.Keys.Auth, req.Platform); err != nil {
		if errors.Is(err, apperror.ErrTooManyPushSubscriptions) {
			apierror.ErrTooManyPushSubscriptions.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	// 返す情報はない(購読IDはクライアントに不要。端末側は endpoint で自分を識別する)
	ctx.Status(http.StatusNoContent)
}

func (c *PushSubscription) Unsubscribe(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	req := helper.GetPushSubscriptionDeleteRequest(ctx)

	if err := c.usecase.Unsubscribe(ctx.Request.Context(), uid, req.Endpoint); err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
