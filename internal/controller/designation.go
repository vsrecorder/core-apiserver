package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	DesignationsPath     = "/designations"
	DesignationPath      = "/designation"
	DesignationStatsPath = "/stats"
)

type Designation struct {
	router  *gin.Engine
	usecase usecase.DesignationInterface
}

func NewDesignation(
	router *gin.Engine,
	usecase usecase.DesignationInterface,
) *Designation {
	return &Designation{router, usecase}
}

func (c *Designation) RegisterRoute(relativePath string) {
	c.router.GET(
		relativePath+DesignationsPath,
		c.GetAllDefinitions,
	)

	c.router.GET(
		relativePath+DesignationsPath+DesignationStatsPath,
		c.GetRankStats,
	)

	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+DesignationPath,
		c.GetByUserId,
	)
}

func (c *Designation) GetAllDefinitions(ctx *gin.Context) {
	definitions, err := c.usecase.GetAllDefinitions(context.Background())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewDesignationsResponse(definitions)

	ctx.JSON(http.StatusOK, res)
}

func (c *Designation) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)

	season, err := helper.ParseQuerySeason(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}

	view, err := c.usecase.GetByUserId(context.Background(), uid, season)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserDesignationResponse(uid, season, view)

	ctx.JSON(http.StatusOK, res)
}

func (c *Designation) GetRankStats(ctx *gin.Context) {
	season, err := helper.ParseQuerySeason(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}

	view, err := c.usecase.GetRankStats(context.Background(), season)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewDesignationRankStatsResponse(season, view)

	ctx.JSON(http.StatusOK, res)
}
