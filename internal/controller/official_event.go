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
	OfficialEventsPath = "/official_events"
)

type OfficialEvent struct {
	router  *gin.Engine
	usecase usecase.OfficialEventInterface
}

func NewOfficialEvent(
	router *gin.Engine,
	usecase usecase.OfficialEventInterface,
) *OfficialEvent {
	return &OfficialEvent{router, usecase}
}

func (c *OfficialEvent) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + OfficialEventsPath)
	r.GET(
		"",
		validation.OfficialEventGetMiddleware(),
		c.Get,
	)
	r.GET(
		"/:id",
		validation.OfficialEventGetByIdMiddleware(),
		c.GetById,
	)
}

func (c *OfficialEvent) Get(ctx *gin.Context) {
	typeId := helper.GetTypeId(ctx)
	leagueType := helper.GetLeagueType(ctx)
	date := helper.GetDate(ctx)
	startDate := helper.GetStartDate(ctx)
	endDate := helper.GetEndDate(ctx)

	if !date.Equal((time.Time{})) {
		officialEvents, err := c.usecase.Find(context.Background(), typeId, leagueType, date, date)
		if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		count := len(officialEvents)
		res := presenter.NewOfficialEventGetResponse(typeId, leagueType, date, date, count, officialEvents)

		ctx.JSON(http.StatusOK, res)
	} else {
		officialEvents, err := c.usecase.Find(context.Background(), typeId, leagueType, startDate, endDate)
		if err != nil {
			apierror.ErrInternalServerError.JSON(ctx)
			return
		}

		count := len(officialEvents)
		res := presenter.NewOfficialEventGetResponse(typeId, leagueType, startDate, endDate, count, officialEvents)

		ctx.JSON(http.StatusOK, res)
	}
}

func (c *OfficialEvent) GetById(ctx *gin.Context) {
	id := helper.GetOfficialEventId(ctx)

	officialEvent, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewOfficialEventGetByIdResponse(officialEvent)

	ctx.JSON(http.StatusOK, res)
}
