package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	MyGymsPath = "/my_gyms"
)

type UserGym struct {
	router  *gin.Engine
	usecase usecase.UserGymInterface
}

func NewUserGym(
	router *gin.Engine,
	usecase usecase.UserGymInterface,
) *UserGym {
	return &UserGym{router, usecase}
}

// RegisterRoute はMyジムのルートを /users 配下に登録する。
//
// 読み取りも含めて uid はパスパラメータではなく認証済みトークンから取る
// (push_subscriptions と同じ)。Myジムは本人しか見ないため、
// /users/:id/... の形にして他人のIDを指定できる余地を作らない。
func (c *UserGym) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath + MyGymsPath)
	r.GET(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		c.Get,
	)
	r.POST(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.UserGymCreateMiddleware(),
		c.Create,
	)
	r.DELETE(
		"/:shop_id",
		authentication.RequiredAuthenticationMiddleware(),
		validation.UserGymDeleteMiddleware(),
		c.Delete,
	)
	r.GET(
		OfficialEventsPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.UserGymOfficialEventGetMiddleware(),
		c.GetOfficialEvents,
	)
}

func (c *UserGym) Get(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	views, err := c.usecase.Find(ctx.Request.Context(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserGymGetResponse(views)

	ctx.JSON(http.StatusOK, res)
}

func (c *UserGym) Create(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	req := helper.GetUserGymCreateRequest(ctx)

	view, err := c.usecase.Create(ctx.Request.Context(), uid, req.ShopId)
	if err != nil {
		// 実在しない店舗ID
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		// 既に登録済みの店舗
		if errors.Is(err, apperror.ErrAlreadyExists) {
			apierror.ErrConflict.JSON(ctx, err)
			return
		}

		// 上限に達している。どれを外すかはユーザーに選ばせる
		if errors.Is(err, apperror.ErrTooManyUserGyms) {
			apierror.ErrTooManyUserGyms.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserGymCreateResponse(view)

	ctx.JSON(http.StatusCreated, res)
}

// Delete はMyジムを解除する。登録されていない店舗を指定されても 204 を返す
// (解除の結果として「登録されていない」状態は同じで、再試行を安全にするため)。
func (c *UserGym) Delete(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	shopId := helper.GetShopId(ctx)

	if err := c.usecase.Delete(ctx.Request.Context(), uid, shopId); err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (c *UserGym) GetOfficialEvents(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	startDate := helper.GetStartDate(ctx)
	endDate := helper.GetEndDate(ctx)

	views, officialEvents, err := c.usecase.FindOfficialEvents(ctx.Request.Context(), uid, startDate, endDate)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserGymOfficialEventGetResponse(startDate, endDate, views, officialEvents)

	ctx.JSON(http.StatusOK, res)
}
