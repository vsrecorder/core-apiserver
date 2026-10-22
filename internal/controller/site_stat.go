package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	SiteStatsPath = "/site_stats"
)

type SiteStat struct {
	router  *gin.Engine
	usecase usecase.SiteStatInterface
}

func NewSiteStat(
	router *gin.Engine,
	usecase usecase.SiteStatInterface,
) *SiteStat {
	return &SiteStat{router, usecase}
}

func (c *SiteStat) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + SiteStatsPath)
	r.GET(
		"",
		c.Get,
	)
}

// Get はサービス全体のユーザ数・対戦数・デッキコード数を返す。トップページの統計カウンタに使う。
//
// 未ログインのトップページで出す数字で、個人を特定する情報を含まないため認証は要求しない。
// 以前は webapp が Grafana の公開ダッシュボード(dashboard.vsrecorder.mobi)から取っていた。
func (c *SiteStat) Get(ctx *gin.Context) {
	stat, err := c.usecase.Find(ctx.Request.Context())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewSiteStatResponse(stat)

	ctx.JSON(http.StatusOK, res)
}
