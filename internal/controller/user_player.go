package controller

import (
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
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	UserPlayersPath                 = "/usersplayers"
	UserPlayerCityleagueResultsPath = "/cityleague_results"
)

type UserPlayer struct {
	logger                 *slog.Logger
	router                 *gin.Engine
	usecase                usecase.UserPlayerInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
	linkingEnabled         bool
}

func NewUserPlayer(
	logger *slog.Logger,
	router *gin.Engine,
	usecase usecase.UserPlayerInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	linkingEnabled bool,
) *UserPlayer {
	return &UserPlayer{logger, router, usecase, championshipSeriesRepo, linkingEnabled}
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
	// 入賞結果は「連携済みの本人のもの」しか返さないため、他人のIDを指定する余地を
	// 作らないよう、パスにプレイヤーIDは取らずトークンのuidから引く。
	r.GET(
		UserPlayerCityleagueResultsPath,
		c.linkingEnabledMiddleware(),
		authentication.RequiredAuthenticationMiddleware(),
		c.GetCityleagueResultsByUID,
	)
}

// GetCityleagueResultsByUID は連携済みプレイヤーIDの、指定シーズンにおける入賞を返す。
// 連携が無ければ404。連携済みで入賞0件は正常系(count=0)として200を返す。
func (c *UserPlayer) GetCityleagueResultsByUID(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	season, err := helper.ParseQuerySeason(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx, err)
		return
	}

	if season == "" {
		season, err = usecase.CurrentSeasonLabel(ctx.Request.Context(), c.championshipSeriesRepo, timeNow().Local())
		if err != nil {
			apierror.ErrInternalServerError.JSON(ctx, err)
			return
		}
	}

	playerCityleagueResults, err := c.usecase.FindCityleagueResultsByUserId(ctx.Request.Context(), uid, season)
	if err != nil {
		// 紐付けが無い場合と、形式は正しいが championship_series に存在しない season を
		// 指定された場合。いずれもクライアント起因のためサーバエラー(500)にはしない。
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserPlayerCityleagueResultsGetResponse(season, playerCityleagueResults)

	ctx.JSON(http.StatusOK, res)
}

func (c *UserPlayer) GetByUID(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	userPlayer, err := c.usecase.FindByUserId(ctx.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
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

	userPlayer, err := c.usecase.Create(ctx.Request.Context(), param)
	if err != nil {
		if errors.Is(err, apperror.ErrLocked) {
			apierror.ErrUserPlayerLocked.JSON(ctx, err)
			return
		}

		// uid と request_id は ContextHandler が ctx から付与するため指定しない。
		c.logger.ErrorContext(ctx.Request.Context(), "controller_user_player_create_failed",
			slog.String("player_id", req.PlayerId),
			slog.String("error_message", err.Error()),
		)

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserPlayerCreateResponse(userPlayer)

	ctx.JSON(http.StatusCreated, res)
}
