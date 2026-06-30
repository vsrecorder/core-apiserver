package controller

import (
	"context"
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
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)

	stats, err := c.usecase.GetUserStat(context.Background(), uid, yearMonth, environmentId, season)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserStatResponse(stats, yearMonth, environmentId, season)

	ctx.JSON(http.StatusOK, res)
}

func (c *UserStat) GetHistoryByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	period := helper.GetPeriod(ctx)
	season := helper.GetSeason(ctx)

	history, err := c.historyUsecase.GetUserStatHistory(context.Background(), uid, period, season)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserStatHistoryResponse(uid, period, season, history)

	ctx.JSON(http.StatusOK, res)
}
