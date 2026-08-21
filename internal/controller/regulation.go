package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	RegulationsPath = "/regulations"
)

type Regulation struct {
	router  *gin.Engine
	usecase usecase.RegulationInterface
}

func NewRegulation(
	router *gin.Engine,
	usecase usecase.RegulationInterface,
) *Regulation {
	return &Regulation{router, usecase}
}

func (c *Regulation) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + RegulationsPath)
	r.GET(
		"",
		c.Get,
	)
}

func (c *Regulation) Get(ctx *gin.Context) {
	regulations, err := c.usecase.Find(ctx.Request.Context())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewRegulationGetResponse(regulations)

	ctx.JSON(http.StatusOK, res)
}
