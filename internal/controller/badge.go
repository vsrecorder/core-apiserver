package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	BadgesPath = "/badges"
)

type Badge struct {
	router                 *gin.Engine
	usecase                usecase.BadgeInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
}

func NewBadge(
	router *gin.Engine,
	usecase usecase.BadgeInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
) *Badge {
	return &Badge{router, usecase, championshipSeriesRepo}
}

func (c *Badge) RegisterRoute(relativePath string) {
	c.router.GET(
		relativePath+BadgesPath,
		c.GetAllDefinitions,
	)

	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+BadgesPath,
		c.GetByUserId,
	)
}

func (c *Badge) GetAllDefinitions(ctx *gin.Context) {
	definitions, err := c.usecase.GetAllDefinitions(context.Background())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewBadgeDefinitionsResponse(definitions)

	ctx.JSON(http.StatusOK, res)
}

func (c *Badge) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)

	season, err := helper.ParseQuerySeason(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}

	if season == "" {
		season, err = usecase.CurrentSeasonLabel(context.Background(), c.championshipSeriesRepo, time.Now().Local())
		if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}
	}

	views, err := c.usecase.GetByUserId(context.Background(), uid, season)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserBadgesResponse(uid, season, views)

	ctx.JSON(http.StatusOK, res)
}
