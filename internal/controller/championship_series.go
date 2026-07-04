package controller

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ChampionshipSeriesPath = "/championship_series"
)

type ChampionshipSeries struct {
	router  *gin.Engine
	usecase usecase.ChampionshipSeriesInterface
}

func NewChampionshipSeries(
	router *gin.Engine,
	usecase usecase.ChampionshipSeriesInterface,
) *ChampionshipSeries {
	return &ChampionshipSeries{router, usecase}
}

func (c *ChampionshipSeries) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + ChampionshipSeriesPath)
	r.GET(
		"",
		validation.ChampionshipSeriesGetByDateMiddleware(),
		c.GetByDate,
		c.Get,
	)
	r.GET(
		"/:id",
		c.GetById,
	)
}

func (c *ChampionshipSeries) Get(ctx *gin.Context) {
	championshipSeriesList, err := c.usecase.Find(context.Background())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewChampionshipSeriesGetResponse(championshipSeriesList)

	ctx.JSON(http.StatusOK, res)
}

func (c *ChampionshipSeries) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	championshipSeries, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewChampionshipSeriesGetByIdResponse(championshipSeries)

	ctx.JSON(http.StatusOK, res)
}

func (c *ChampionshipSeries) GetByDate(ctx *gin.Context) {
	date := helper.GetDate(ctx)

	if date.Equal((time.Time{})) {
		return
	}

	championshipSeries, err := c.usecase.FindByDate(context.Background(), date)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewChampionshipSeriesGetByDateResponse(championshipSeries)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}
