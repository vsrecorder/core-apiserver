package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
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

func (c *OpponentDeckUsageStat) RegisterRoute(relativePath string, authDisable bool) {
	r := c.router.Group(relativePath + UsersPath)
	if authDisable {
		r.GET(
			"/:id"+OpponentDeckUsageStatsPath,
			validation.OpponentDeckUsageStatGetMiddleware(),
			c.GetByUserId,
		)
	} else {
		r.GET(
			"/:id"+OpponentDeckUsageStatsPath,
			authentication.RequiredAuthenticationMiddleware(),
			authorization.OpponentDeckUsageStatAuthorizationMiddleware(),
			validation.OpponentDeckUsageStatGetMiddleware(),
			c.GetByUserId,
		)
	}
}

func (c *OpponentDeckUsageStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)

	stat, err := c.usecase.GetOpponentDeckUsageStat(context.Background(), uid, yearMonth, environmentId, season)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewOpponentDeckUsageStatResponse(stat, yearMonth, environmentId, season)

	ctx.JSON(http.StatusOK, res)
}
