package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	EnvironmentBadgesPath = "/environment_badges"
)

type EnvironmentBadge struct {
	router  *gin.Engine
	usecase usecase.EnvironmentBadgeInterface
}

func NewEnvironmentBadge(
	router *gin.Engine,
	usecase usecase.EnvironmentBadgeInterface,
) *EnvironmentBadge {
	return &EnvironmentBadge{router, usecase}
}

func (c *EnvironmentBadge) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+EnvironmentBadgesPath,
		c.GetByUserId,
	)
}

func (c *EnvironmentBadge) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)

	views, err := c.usecase.GetByUserId(ctx.Request.Context(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUserEnvironmentBadgesResponse(uid, views)

	ctx.JSON(http.StatusOK, res)
}
