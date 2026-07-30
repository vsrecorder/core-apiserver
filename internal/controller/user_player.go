package controller

import (
	"context"
	"errors"
	"log/slog"
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
	UserPlayersPath = "/usersplayers"
)

type UserPlayer struct {
	logger         *slog.Logger
	router         *gin.Engine
	usecase        usecase.UserPlayerInterface
	linkingEnabled bool
}

func NewUserPlayer(
	logger *slog.Logger,
	router *gin.Engine,
	usecase usecase.UserPlayerInterface,
	linkingEnabled bool,
) *UserPlayer {
	return &UserPlayer{logger, router, usecase, linkingEnabled}
}

// linkingEnabledMiddleware は運用者が環境変数でプレイヤーID連携機能を
// 一時的に停止できるようにするためのキルスイッチ。悪用が多発した場合に
// デプロイなしで機能全体を止められるようにする。
func (c *UserPlayer) linkingEnabledMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !c.linkingEnabled {
			apierror.ErrUserPlayerLinkingDisabled.JSON(ctx)
			return
		}
	}
}

func (c *UserPlayer) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UserPlayersPath)
	r.GET(
		"",
		c.linkingEnabledMiddleware(),
		authentication.RequiredAuthenticationMiddleware(),
		c.GetByUID,
	)
	r.POST(
		"",
		c.linkingEnabledMiddleware(),
		authentication.RequiredAuthenticationMiddleware(),
		validation.UserPlayerCreateMiddleware(),
		c.Create,
	)
}

func (c *UserPlayer) GetByUID(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	userPlayer, err := c.usecase.FindByUserId(context.Background(), uid)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserPlayerGetResponse(userPlayer)

	ctx.JSON(http.StatusOK, res)
}

// Create は player_id と user_id の紐付けを保存する。
// player_id の実在確認・所有権確認は行わず、利用者の自己申告として受け入れる。
func (c *UserPlayer) Create(ctx *gin.Context) {
	req := helper.GetUserPlayerCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	param := usecase.NewUserPlayerCreateParam(
		uid,
		req.PlayerId,
	)

	userPlayer, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		if errors.Is(err, apperror.ErrLocked) {
			apierror.ErrUserPlayerLocked.JSON(ctx)
			return
		}

		c.logger.Error("controller_user_player_create_failed",
			slog.String("uid", uid),
			slog.String("player_id", req.PlayerId),
			slog.String("error_message", err.Error()),
		)

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserPlayerCreateResponse(userPlayer)

	ctx.JSON(http.StatusCreated, res)
}
