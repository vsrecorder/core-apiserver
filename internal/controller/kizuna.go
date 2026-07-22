package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	KizunaPath = "/kizuna"
)

type Kizuna struct {
	router  *gin.Engine
	usecase usecase.KizunaInterface
}

func NewKizuna(
	router *gin.Engine,
	usecase usecase.KizunaInterface,
) *Kizuna {
	return &Kizuna{router, usecase}
}

// クエリパラメータを取らないため validation ミドルウェアは無い。
// きずなは全期間の積み重ねであり、期間で切る概念が無いため（usecase のコメント参照）。
func (c *Kizuna) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+KizunaPath,
		authentication.RequiredAuthenticationMiddleware(),
		authorization.KizunaAuthorizationMiddleware(),
		c.GetByUserId,
	)
}

func (c *Kizuna) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)

	kizuna, err := c.usecase.GetKizuna(context.Background(), uid)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewKizunaResponse(kizuna)

	ctx.JSON(http.StatusOK, res)
}
