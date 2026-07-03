package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	StreakPath = "/streak"
)

type Streak struct {
	router  *gin.Engine
	usecase usecase.StreakInterface
}

func NewStreak(
	router *gin.Engine,
	usecase usecase.StreakInterface,
) *Streak {
	return &Streak{router, usecase}
}

func (c *Streak) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+StreakPath,
		c.GetByUserId,
	)
}

func (c *Streak) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)

	streak, err := c.usecase.GetByUserId(context.Background(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserStreakResponse(streak)

	ctx.JSON(http.StatusOK, res)
}
