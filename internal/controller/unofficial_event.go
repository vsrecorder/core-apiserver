package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	UnofficialEventsPath = "/unofficial_events"
)

type UnofficialEvent struct {
	router     *gin.Engine
	repository repository.UnofficialEventInterface
	usecase    usecase.UnofficialEventInterface
}

func NewUnofficialEvent(
	router *gin.Engine,
	repository repository.UnofficialEventInterface,
	usecase usecase.UnofficialEventInterface,
) *UnofficialEvent {
	return &UnofficialEvent{router, repository, usecase}
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
	r.PUT(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.UnofficialEventUpdateAuthorizationMiddleware(c.repository),
		validation.UnofficialEventUpdateMiddleware(),
		c.Update,
	)
	r.DELETE(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.UnofficialEventDeleteAuthorizationMiddleware(c.repository),
		c.Delete,
	)
}

func (c *UnofficialEvent) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	unofficialEvent, err := c.usecase.FindById(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
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

	unofficialEvent, err := c.usecase.Create(ctx.Request.Context(), param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUnofficialEventCreateResponse(unofficialEvent)

	ctx.JSON(http.StatusCreated, res)
}

func (c *UnofficialEvent) Update(ctx *gin.Context) {
	req := helper.GetUnofficialEventUpdateRequest(ctx)
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	param := usecase.NewUnofficialEventParam(
		uid,
		req.Title,
		req.Date,
	)

	unofficialEvent, err := c.usecase.Update(ctx.Request.Context(), id, param)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewUnofficialEventUpdateResponse(unofficialEvent)

	ctx.JSON(http.StatusOK, res)
}

func (c *UnofficialEvent) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrBadRequestNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
