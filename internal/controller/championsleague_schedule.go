package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ChampionsleagueSchedulesPath = "/championsleague_schedules"
)

type ChampionsleagueSchedule struct {
	router  *gin.Engine
	usecase usecase.ChampionsleagueScheduleInterface
}

func NewChampionsleagueSchedule(
	router *gin.Engine,
	usecase usecase.ChampionsleagueScheduleInterface,
) *ChampionsleagueSchedule {
	return &ChampionsleagueSchedule{router, usecase}
}

func (c *ChampionsleagueSchedule) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + ChampionsleagueSchedulesPath)
	r.GET(
		"",
		c.Get,
	)
	r.GET(
		"/:id",
		c.GetById,
	)
}

func (c *ChampionsleagueSchedule) Get(ctx *gin.Context) {
	championsleagueSchedules, err := c.usecase.Find(ctx.Request.Context())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewChampionsleagueScheduleGetResponse(championsleagueSchedules)

	ctx.JSON(http.StatusOK, res)
}

func (c *ChampionsleagueSchedule) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	championsleagueSchedule, err := c.usecase.FindById(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewChampionsleagueScheduleGetByIdResponse(championsleagueSchedule)

	ctx.JSON(http.StatusOK, res)
}
