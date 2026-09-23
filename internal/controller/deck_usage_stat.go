package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	DeckUsageStatsPath = "/deck_usage"
	// DeckCodeUsageStatsPath はデッキの成績をバージョン(デッキコード)ごとに分けて返す。
	DeckCodeUsageStatsPath = "/deck_code_usage"
)

type DeckUsageStat struct {
	router  *gin.Engine
	usecase usecase.DeckUsageStatInterface
}

func NewDeckUsageStat(
	router *gin.Engine,
	usecase usecase.DeckUsageStatInterface,
) *DeckUsageStat {
	return &DeckUsageStat{router, usecase}
}

func (c *DeckUsageStat) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+DeckUsageStatsPath,
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckUsageStatAuthorizationMiddleware(),
		validation.DeckUsageStatGetMiddleware(),
		c.GetByUserId,
	)
	// 自分の成績だけを返す(他人の :id は DeckUsageStatAuthorizationMiddleware が 403 にする)。
	// 集計でも records.user_id で絞るため、他人のデッキIDを渡されても何も数えない。
	r.GET(
		"/:id"+DeckCodeUsageStatsPath,
		authentication.RequiredAuthenticationMiddleware(),
		authorization.DeckUsageStatAuthorizationMiddleware(),
		validation.DeckCodeUsageStatGetMiddleware(),
		c.GetDeckCodeUsageByUserId,
	)
}

func (c *DeckUsageStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	week := helper.GetWeek(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)
	standardRegulationId := helper.GetStandardRegulationId(ctx)
	regulationId := helper.GetRegulationId(ctx)
	allTime := helper.GetAllTime(ctx)
	excludeDefaultMatches := helper.GetExcludeDefaultMatches(ctx)

	stat, err := c.usecase.GetDeckUsageStat(ctx.Request.Context(), uid, week, yearMonth, environmentId, season, standardRegulationId, regulationId, allTime, excludeDefaultMatches)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckUsageStatResponse(stat, week, yearMonth, environmentId, season, standardRegulationId, regulationId)

	ctx.JSON(http.StatusOK, res)
}

func (c *DeckUsageStat) GetDeckCodeUsageByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	deckId := helper.GetDeckId(ctx)
	excludeDefaultMatches := helper.GetExcludeDefaultMatches(ctx)

	stat, err := c.usecase.GetDeckCodeUsageStat(ctx.Request.Context(), uid, deckId, excludeDefaultMatches)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckCodeUsageStatResponse(stat)

	ctx.JSON(http.StatusOK, res)
}
