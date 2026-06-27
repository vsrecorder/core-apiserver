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

func (c *DeckUsageStat) RegisterRoute(relativePath string, authDisable bool) {
	r := c.router.Group(relativePath + UsersPath)
	if authDisable {
		r.GET(
			"/:id"+DeckUsageStatsPath,
			validation.DeckUsageStatGetMiddleware(),
			c.GetByUserId,
		)
	} else {
		r.GET(
			"/:id"+DeckUsageStatsPath,
			authentication.RequiredAuthenticationMiddleware(),
			authorization.DeckUsageStatAuthorizationMiddleware(),
			validation.DeckUsageStatGetMiddleware(),
			c.GetByUserId,
		)
	}
}

func (c *DeckUsageStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)

	stat, err := c.usecase.GetDeckUsageStat(context.Background(), uid, yearMonth, environmentId, season)
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

	res := presenter.NewDeckUsageStatResponse(stat, yearMonth, environmentId, season)

	ctx.JSON(http.StatusOK, res)
}
