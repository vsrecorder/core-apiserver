package controller

import (
	"context"
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
	OpponentDeckUsageStatsPath = "/opponent_deck_usage"
)

type OpponentDeckUsageStat struct {
	router  *gin.Engine
	usecase usecase.OpponentDeckUsageStatInterface
}

func NewOpponentDeckUsageStat(
	router *gin.Engine,
	usecase usecase.OpponentDeckUsageStatInterface,
) *OpponentDeckUsageStat {
	return &OpponentDeckUsageStat{router, usecase}
}

func (c *OpponentDeckUsageStat) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+OpponentDeckUsageStatsPath,
		authentication.RequiredAuthenticationMiddleware(),
		authorization.OpponentDeckUsageStatAuthorizationMiddleware(),
		validation.OpponentDeckUsageStatGetMiddleware(),
		c.GetByUserId,
	)
}

func (c *OpponentDeckUsageStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)

	stat, err := c.usecase.GetOpponentDeckUsageStat(context.Background(), uid, yearMonth, environmentId, season)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewOpponentDeckUsageStatResponse(stat, yearMonth, environmentId, season)

	ctx.JSON(http.StatusOK, res)
}
