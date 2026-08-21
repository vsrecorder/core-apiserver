package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	TonamelEventsPath = "/tonamel_events"
)

type TonamelEvent struct {
	router  *gin.Engine
	usecase usecase.TonamelEventInterface
}

func NewTonamelEvent(
	router *gin.Engine,
	usecase usecase.TonamelEventInterface,
) *TonamelEvent {
	return &TonamelEvent{router, usecase}
}

func (c *TonamelEvent) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + TonamelEventsPath)
	r.GET(
		"/:id",
		c.GetById,
	)
}

func (c *TonamelEvent) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	tonamelEvent, err := c.usecase.FindById(ctx.Request.Context(), id)

	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewTonamelEventGetByIdResponse(tonamelEvent)

	ctx.JSON(http.StatusOK, res)
}
