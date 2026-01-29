package controller

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	CityleagueResultsPath = "/cityleague_results"
)

type CityleagueResult struct {
	router  *gin.Engine
	usecase usecase.CityleagueResultInterface
}

func NewCityleagueResult(
	router *gin.Engine,
	usecase usecase.CityleagueResultInterface,
) *CityleagueResult {
	return &CityleagueResult{router, usecase}
}

func (c *CityleagueResult) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + CityleagueResultsPath)
	r.GET(
		"",
		validation.CityleagueResultGetByDateMiddleware(),
		c.GetByDate,
		validation.CityleagueResultGetByTermMiddleware(),
		c.GetByTerm,
		validation.CityleagueResultGetByOfficialEventIdMiddleware(),
		c.GetByOfficialEventId,
		c.Get,
	)
}

func (c *CityleagueResult) GetByDate(ctx *gin.Context) {
	leagueType := helper.GetLeagueType(ctx)
	date := helper.GetDate(ctx)

	if date.Equal((time.Time{})) {
		return
	}

	cityleagueResults, err := c.usecase.FindByDate(context.Background(), leagueType, date)
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

	res := presenter.NewCityleagueResultGetByDateResponse(leagueType, date, len(cityleagueResults), cityleagueResults)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}

func (c *CityleagueResult) GetByTerm(ctx *gin.Context) {
	leagueType := helper.GetLeagueType(ctx)
	fromDate := helper.GetFromDate(ctx)
	toDate := helper.GetToDate(ctx)

	if (fromDate.Equal(time.Time{})) && (toDate.Equal(time.Time{})) {
		return
	}

	cityleagueResults, err := c.usecase.FindByTerm(context.Background(), leagueType, fromDate, toDate)
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

	res := presenter.NewCityleagueResultGetByTermResponse(leagueType, fromDate, toDate, len(cityleagueResults), cityleagueResults)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}

func (c *CityleagueResult) GetByOfficialEventId(ctx *gin.Context) {
	id := helper.GetOfficialEventId(ctx)

	if id == 0 {
		return
	}

	cityleagueResults, err := c.usecase.FindByOfficialEventId(context.Background(), id)
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

	res := presenter.NewCityleagueResultGetByOfficialEventIdResponse(cityleagueResults)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}

func (c *CityleagueResult) Get(ctx *gin.Context) {
	leagueType := helper.GetLeagueType(ctx)
	date := time.Now()
	fromDate := date.AddDate(0, 0, -7)
	fromDate = time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)

	cityleagueResults, err := c.usecase.FindByTerm(context.Background(), leagueType, fromDate, toDate)
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

	res := presenter.NewCityleagueResultGetByTermResponse(leagueType, fromDate, toDate, len(cityleagueResults), cityleagueResults)

	ctx.JSON(http.StatusOK, res)
}
