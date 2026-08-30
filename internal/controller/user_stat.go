package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	UserStatsPath = "/stats"
)

type UserStat struct {
	router         *gin.Engine
	usecase        usecase.UserStatInterface
	historyUsecase usecase.UserStatHistoryInterface
	recentUsecase  usecase.UserStatRecentInterface
}

func NewUserStat(
	router *gin.Engine,
	usecase usecase.UserStatInterface,
	historyUsecase usecase.UserStatHistoryInterface,
	recentUsecase usecase.UserStatRecentInterface,
) *UserStat {
	return &UserStat{router, usecase, historyUsecase, recentUsecase}
}

func (c *UserStat) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+UserStatsPath,
		validation.UserStatGetMiddleware(),
		c.GetByUserId,
	)
	r.GET(
		"/:id"+UserStatsPath+"/history",
		validation.UserStatHistoryGetMiddleware(),
		c.GetHistoryByUserId,
	)
	r.GET(
		"/:id"+UserStatsPath+"/recent",
		validation.UserStatRecentGetMiddleware(),
		c.GetRecentByUserId,
	)
}

func (c *UserStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	week := helper.GetWeek(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)
	standardRegulationId := helper.GetStandardRegulationId(ctx)
	regulationId := helper.GetRegulationId(ctx)

	stats, err := c.usecase.GetUserStat(ctx.Request.Context(), uid, week, yearMonth, environmentId, season, standardRegulationId, regulationId)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserStatResponse(stats, week, yearMonth, environmentId, season, standardRegulationId, regulationId)

	ctx.JSON(http.StatusOK, res)
}

func (c *UserStat) GetHistoryByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	period := helper.GetPeriod(ctx)
	season := helper.GetSeason(ctx)
	deckId := helper.GetDeckId(ctx)
	regulationId := helper.GetRegulationId(ctx)

	history, err := c.historyUsecase.GetUserStatHistory(ctx.Request.Context(), uid, period, season, deckId, regulationId)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserStatHistoryResponse(uid, period, season, deckId, regulationId, history)

	ctx.JSON(http.StatusOK, res)
}

func (c *UserStat) GetRecentByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	count := helper.GetLimit(ctx)
	deckId := helper.GetDeckId(ctx)
	regulationId := helper.GetRegulationId(ctx)

	stat, err := c.recentUsecase.GetRecentMatches(ctx.Request.Context(), uid, count, deckId, regulationId)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewRecentMatchStatResponse(stat, deckId, regulationId)

	ctx.JSON(http.StatusOK, res)
}
