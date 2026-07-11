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
	DeckUsageStatsPath = "/deck_usage"
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
}

func (c *DeckUsageStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)
	regulationId := helper.GetRegulationId(ctx)
	allTime := helper.GetAllTime(ctx)

	stat, err := c.usecase.GetDeckUsageStat(context.Background(), uid, yearMonth, environmentId, season, regulationId, allTime)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewDeckUsageStatResponse(stat, yearMonth, environmentId, season, regulationId)

	ctx.JSON(http.StatusOK, res)
}
