package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
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
	startDate := helper.GetStartDate(ctx)
	endDate := helper.GetEndDate(ctx)

	officialEvents, err := c.usecase.Find(context.Background(), typeId, leagueType, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	count := len(officialEvents)

	res := presenter.NewOfficialEventGetResponse(typeId, leagueType, startDate, endDate, count, officialEvents)

	ctx.JSON(http.StatusOK, res)
}

func (c *OfficialEvent) GetById(ctx *gin.Context) {
	id := helper.GetOfficialEventId(ctx)

	officialEvent, err := c.usecase.FindById(context.Background(), id)
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

	res := presenter.NewOfficialEventGetByIdResponse(officialEvent)

	ctx.JSON(http.StatusOK, res)
}
