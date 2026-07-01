package controller

import (
	"context"
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
	UnofficialEventsPath = "/unofficial_events"
)

type UnofficialEvent struct {
	router  *gin.Engine
	usecase usecase.UnofficialEventInterface
}

func NewUnofficialEvent(
	router *gin.Engine,
	usecase usecase.UnofficialEventInterface,
) *UnofficialEvent {
	return &UnofficialEvent{router, usecase}
}

func (c *UnofficialEvent) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UnofficialEventsPath)
	r.GET(
		"/:id",
		c.GetById,
	)
	r.POST(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.UnofficialEventCreateMiddleware(),
		c.Create,
	)
}

func (c *UnofficialEvent) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	unofficialEvent, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUnofficialEventGetByIdResponse(unofficialEvent)

	ctx.JSON(http.StatusOK, res)
}

func (c *UnofficialEvent) Create(ctx *gin.Context) {
	req := helper.GetUnofficialEventCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	param := usecase.NewUnofficialEventParam(
		uid,
		req.Title,
		req.Date,
	)

	unofficialEvent, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUnofficialEventCreateResponse(unofficialEvent)

	ctx.JSON(http.StatusCreated, res)
}
