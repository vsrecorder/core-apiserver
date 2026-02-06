package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
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

	tonamelEvent, err := c.usecase.FindById(context.Background(), id)

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

	res := presenter.NewTonamelEventGetByIdResponse(tonamelEvent)

	ctx.JSON(http.StatusOK, res)
}
