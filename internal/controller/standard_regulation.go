package controller

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	StandardRegulationsPath = "/standard_regulations"
)

type StandardRegulation struct {
	router  *gin.Engine
	usecase usecase.StandardRegulationInterface
}

func NewStandardRegulation(
	router *gin.Engine,
	usecase usecase.StandardRegulationInterface,
) *StandardRegulation {
	return &StandardRegulation{router, usecase}
}

func (c *StandardRegulation) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + StandardRegulationsPath)
	r.GET(
		"",
		validation.StandardRegulationGetByDateMiddleware(),
		c.GetByDate,
		c.Get,
	)
	r.GET(
		"/:id",
		c.GetById,
	)
}

func (c *StandardRegulation) Get(ctx *gin.Context) {
	standardRegulations, err := c.usecase.Find(context.Background())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewStandardRegulationGetResponse(standardRegulations)

	ctx.JSON(http.StatusOK, res)
}

func (c *StandardRegulation) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	standardRegulation, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewStandardRegulationGetByIdResponse(standardRegulation)

	ctx.JSON(http.StatusOK, res)
}

func (c *StandardRegulation) GetByDate(ctx *gin.Context) {
	date := helper.GetDate(ctx)

	if date.Equal((time.Time{})) {
		return
	}

	standardRegulations, err := c.usecase.FindByDate(context.Background(), date)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewStandardRegulationGetByDateResponse(standardRegulations)

	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}
