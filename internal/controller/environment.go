package controller

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	EnvironmentsPath = "/environments"
)

type Environment struct {
	router  *gin.Engine
	usecase usecase.EnvironmentInterface
}

func NewEnvironment(
	router *gin.Engine,
	usecase usecase.EnvironmentInterface,
) *Environment {
	return &Environment{router, usecase}
}

func (c *Environment) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + EnvironmentsPath)
	r.GET(
		"",
		validation.EnvironmentGetByDateMiddleware(),
		c.GetByDate,
		validation.EnvironmentGetByTermMiddleware(),
		c.GetByTerm,
		c.Get,
	)
	r.GET(
		"/:id",
		c.GetById,
	)
}

func (c *Environment) Get(ctx *gin.Context) {
	environments, err := c.usecase.Find(context.Background())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewEnvironmentGetResponse(environments)

	ctx.JSON(http.StatusOK, res)
}

func (c *Environment) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	environment, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewEnvironmentGetByIdResponse(environment)

	ctx.JSON(http.StatusOK, res)
}

func (c *Environment) GetByDate(ctx *gin.Context) {
	date := helper.GetDate(ctx)

	if date.Equal((time.Time{})) {
		return
	}

	environment, err := c.usecase.FindByDate(context.Background(), date)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewEnvironmentGetByDateResponse(environment)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}

func (c *Environment) GetByTerm(ctx *gin.Context) {
	fromDate := helper.GetFromDate(ctx)
	toDate := helper.GetToDate(ctx)

	if (fromDate.Equal(time.Time{})) && (toDate.Equal(time.Time{})) {
		return
	}

	environments, err := c.usecase.FindByTerm(context.Background(), fromDate, toDate)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewEnvironmentGetByTermResponse(environments)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}
