package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	UserDailyActivityPath = "/activity"
)

type UserDailyActivity struct {
	router  *gin.Engine
	usecase usecase.UserDailyActivityInterface
}

func NewUserDailyActivity(
	router *gin.Engine,
	usecase usecase.UserDailyActivityInterface,
) *UserDailyActivity {
	return &UserDailyActivity{router, usecase}
}

func (c *UserDailyActivity) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	// 書き込みなので uid はパスパラメータではなく認証済みトークンから取る。
	// パスパラメータを信用すると他人の行を書けてしまう。
	r.POST(
		UserDailyActivityPath,
		authentication.RequiredAuthenticationMiddleware(),
		c.Record,
	)
}

func (c *UserDailyActivity) Record(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	// 計測は非クリティカルな処理のため、bodyが空でも壊れていても visit として受け付ける。
	// validation middleware を挟まず controller で読んでいるのは、
	// 「不正なら弾く」ではなく「既定値へ丸める」のがこのエンドポイントの仕様だから。
	var req dto.UserDailyActivityRequest
	_ = ctx.ShouldBindJSON(&req)

	categories := req.Categories
	if len(categories) == 0 {
		categories = []string{entity.UserDailyActivityCategoryVisit}
	}

	if err := c.usecase.Record(ctx.Request.Context(), uid, categories); err != nil {
		// 既知のカテゴリが1つも無いときだけ400。未知が混ざっているだけなら
		// 既知のぶんを記録して204を返す(webapp先行デプロイ時に計測を落とさないため)。
		if errors.Is(err, apperror.ErrNoKnownActivityCategory) {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	// 返す情報はない
	ctx.Status(http.StatusNoContent)
}
