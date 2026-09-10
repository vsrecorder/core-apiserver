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
}

func NewUserStat(
	router *gin.Engine,
	usecase usecase.UserStatInterface,
	historyUsecase usecase.UserStatHistoryInterface,
) *UserStat {
	return &UserStat{router, usecase, historyUsecase}
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
}

func (c *UserStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	week := helper.GetWeek(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)
	standardRegulationId := helper.GetStandardRegulationId(ctx)
	regulationId := helper.GetRegulationId(ctx)
	excludeDefaultMatches := helper.GetExcludeDefaultMatches(ctx)

	stats, err := c.usecase.GetUserStat(ctx.Request.Context(), uid, week, yearMonth, environmentId, season, standardRegulationId, regulationId, excludeDefaultMatches)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserStatResponse(stats, week, yearMonth, environmentId, season, standardRegulationId, regulationId, excludeDefaultMatches)

	ctx.JSON(http.StatusOK, res)
}

func (c *UserStat) GetHistoryByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	period := helper.GetPeriod(ctx)
	season := helper.GetSeason(ctx)
	deckId := helper.GetDeckId(ctx)
	regulationId := helper.GetRegulationId(ctx)
	excludeDefaultMatches := helper.GetExcludeDefaultMatches(ctx)

	history, err := c.historyUsecase.GetUserStatHistory(ctx.Request.Context(), uid, period, season, deckId, regulationId, excludeDefaultMatches)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserStatHistoryResponse(uid, period, season, deckId, regulationId, excludeDefaultMatches, history)

	ctx.JSON(http.StatusOK, res)
}
