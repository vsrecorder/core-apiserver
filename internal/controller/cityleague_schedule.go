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
	CityleagueSchedulesPath = "/cityleague_schedules"
)

type CityleagueSchedule struct {
	router  *gin.Engine
	usecase usecase.CityleagueScheduleInterface
}

func NewCityleagueSchedule(
	router *gin.Engine,
	usecase usecase.CityleagueScheduleInterface,
) *CityleagueSchedule {
	return &CityleagueSchedule{router, usecase}
}

func (c *CityleagueSchedule) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + CityleagueSchedulesPath)
	r.GET(
		"",
		validation.CityleagueScheduleGetByDateMiddleware(),
		c.GetByDate,
		c.Get,
	)
	r.GET(
		"/:id",
		c.GetById,
	)
}

func (c *CityleagueSchedule) Get(ctx *gin.Context) {
	cityleagueSchedules, err := c.usecase.Find(context.Background())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewCityleagueScheduleGetResponse(cityleagueSchedules)

	ctx.JSON(http.StatusOK, res)
}

func (c *CityleagueSchedule) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	cs, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewCityleagueScheduleGetByIdResponse(cs)

	ctx.JSON(http.StatusOK, res)
}

func (c *CityleagueSchedule) GetByDate(ctx *gin.Context) {
	date := helper.GetDate(ctx)

	if date.Equal((time.Time{})) {
		return
	}

	cityleagueSchedules, err := c.usecase.FindByDate(context.Background(), date)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewCityleagueScheduleGetByDateResponse(cityleagueSchedules)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}
