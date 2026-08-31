package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ShopsPath = "/shops"
)

type Shop struct {
	router  *gin.Engine
	usecase usecase.ShopInterface
}

func NewShop(
	router *gin.Engine,
	usecase usecase.ShopInterface,
) *Shop {
	return &Shop{router, usecase}
}

func (c *Shop) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + ShopsPath)
	r.GET(
		"",
		validation.ShopGetMiddleware(),
		c.Get,
	)
}

// Get は店舗を検索する。Myジムに登録する店舗を選ぶために使う。
//
// 店舗マスタは公式サイトが公開している情報のため認証は要求しない
// (official_events / cityleague_schedules と同じ扱い)。
func (c *Shop) Get(ctx *gin.Context) {
	keyword := helper.GetKeyword(ctx)
	limit := helper.GetLimit(ctx)

	shops, err := c.usecase.Find(ctx.Request.Context(), keyword, limit)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewShopGetResponse(keyword, shops)

	ctx.JSON(http.StatusOK, res)
}
