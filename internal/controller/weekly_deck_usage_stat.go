package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	DeckMetaPath        = "/deck_meta"
	WeeklyDeckUsagePath = "/weekly_usage"
)

type WeeklyDeckUsageStat struct {
	router  *gin.Engine
	usecase usecase.WeeklyDeckUsageStatInterface
}

func NewWeeklyDeckUsageStat(
	router *gin.Engine,
	usecase usecase.WeeklyDeckUsageStatInterface,
) *WeeklyDeckUsageStat {
	return &WeeklyDeckUsageStat{router, usecase}
}

// RegisterRoute はプラットフォーム全体のデッキメタ集計を公開エンドポイントとして登録する。
// 非会員も閲覧できる環境レポートのため、認証ミドルウェアは付けない。
func (c *WeeklyDeckUsageStat) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + DeckMetaPath)
	r.GET(
		WeeklyDeckUsagePath,
		validation.WeeklyDeckUsageStatGetMiddleware(),
		c.GetWeeklyUsage,
	)
}

func (c *WeeklyDeckUsageStat) GetWeeklyUsage(ctx *gin.Context) {
	week := helper.GetWeek(ctx)

	stat, err := c.usecase.GetWeeklyDeckUsageStat(context.Background(), week)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewWeeklyDeckUsageStatResponse(stat, week)

	ctx.JSON(http.StatusOK, res)
}
