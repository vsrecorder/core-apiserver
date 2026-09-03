package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ChampionsleagueResultsPath      = "/championsleague_results"
	ChampionsleagueResultEventsPath = "/events"
)

type ChampionsleagueResult struct {
	router  *gin.Engine
	usecase usecase.ChampionsleagueResultInterface
}

func NewChampionsleagueResult(
	router *gin.Engine,
	usecase usecase.ChampionsleagueResultInterface,
) *ChampionsleagueResult {
	return &ChampionsleagueResult{router, usecase}
}

func (c *ChampionsleagueResult) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + ChampionsleagueResultsPath)
	r.GET(
		"",
		validation.ChampionsleagueResultGetByChampionsleagueScheduleIdMiddleware(),
		c.GetByChampionsleagueScheduleId,
	)
	r.GET(
		ChampionsleagueResultEventsPath,
		c.GetEvents,
	)
}

// GetEvents は結果が登録されているイベントを、入賞者を含めずに全件返す。
// どの大会の結果が閲覧できるかを知りたい用途(大会一覧ハブ・sitemap)はこちらを使う。
func (c *ChampionsleagueResult) GetEvents(ctx *gin.Context) {
	championsleagueResultEvents, err := c.usecase.FindEvents(ctx.Request.Context())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewChampionsleagueResultGetEventsResponse(len(championsleagueResultEvents), championsleagueResultEvents)

	ctx.JSON(http.StatusOK, res)
}

func (c *ChampionsleagueResult) GetByChampionsleagueScheduleId(ctx *gin.Context) {
	championsleagueScheduleId := helper.GetChampionsleagueScheduleId(ctx)
	leagueType := helper.GetLeagueType(ctx)

	championsleagueResults, err := c.usecase.FindByChampionsleagueScheduleId(ctx.Request.Context(), leagueType, championsleagueScheduleId)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewChampionsleagueResultGetByChampionsleagueScheduleIdResponse(
		championsleagueScheduleId,
		leagueType,
		len(championsleagueResults),
		championsleagueResults,
	)

	ctx.JSON(http.StatusOK, res)
}
