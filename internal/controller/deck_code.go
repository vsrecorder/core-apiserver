package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	DeckCodesPath = "/deckcodes"
)

type DeckCode struct {
	router     *gin.Engine
	repository repository.DeckCodeInterface
	usecase    usecase.DeckCodeInterface
}

func NewDeckCode(
	router *gin.Engine,
	repository repository.DeckCodeInterface,
	usecase usecase.DeckCodeInterface,
) *DeckCode {
	return &DeckCode{router, repository, usecase}
}

func (c *DeckCode) RegisterRoute(relativePath string, authDisable bool) {
	if authDisable {
		{
			r := c.router.Group(relativePath + DeckCodesPath)
			r.GET(
				"/:id",
				c.GetById,
			)
			r.POST(
				"",
				validation.DeckCodeCreateMiddleware(),
				c.Create,
			)
			r.PUT(
				"/:id",
				validation.DeckCodeUpdateMiddleware(),
				c.Update,
			)
			r.DELETE(
				"/:id",
				c.Delete,
			)
		}

		{
			r := c.router.Group(relativePath + DecksPath)
			r.GET(
				"/:id"+DeckCodesPath,
				c.GetByDeckId,
			)
		}
	} else {
		{
			r := c.router.Group(relativePath + DeckCodesPath)
			r.GET(
				"/:id",
				auth.OptionalAuthenticationMiddleware(),
				c.GetById,
			)
			r.POST(
				"",
				auth.RequiredAuthenticationMiddleware(),
				validation.DeckCodeCreateMiddleware(),
				c.Create,
			)
			r.PUT(
				"/:id",
				auth.RequiredAuthenticationMiddleware(),
				auth.DeckCodeUpdateAuthorizationMiddleware(c.repository),
				validation.DeckCodeUpdateMiddleware(),
				c.Update,
			)
			r.DELETE(
				"/:id",
				auth.RequiredAuthenticationMiddleware(),
				auth.DeckCodeDeleteAuthorizationMiddleware(c.repository),
				c.Delete,
			)
		}

		{
			r := c.router.Group(relativePath + DecksPath)
			r.GET(
				"/:id"+DeckCodesPath,
				auth.OptionalAuthenticationMiddleware(),
				c.GetByDeckId,
			)
		}
	}
}

func (c *DeckCode) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	deckcode, err := c.usecase.FindById(context.Background(), id)
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

	if deckcode.PrivateCodeFlg && uid != deckcode.UserId {
		deckcode.Code = ""
	}

	res := presenter.NewDeckCodeGetByIdResponse(deckcode)

	ctx.JSON(http.StatusOK, res)
}

func (c *DeckCode) GetByDeckId(ctx *gin.Context) {
	deckId := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	deckcodes, err := c.usecase.FindByDeckId(context.Background(), deckId)
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

	for _, deckcode := range deckcodes {
		if deckcode.PrivateCodeFlg && uid != deckcode.UserId {
			deckcode.Code = ""
		}
	}

	res := presenter.NewDeckCodeGetByDeckIdResponse(deckcodes)

	ctx.JSON(http.StatusOK, res)
}

func (c *DeckCode) Create(ctx *gin.Context) {
	req := helper.GetDeckCodeCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	param := usecase.NewDeckCodeCreateParam(
		uid,
		req.DeckId,
		req.Code,
		req.PrivateCodeFlg,
		req.Memo,
	)

	deckcode, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewDeckCodeCreateResponse(deckcode)

	ctx.JSON(http.StatusCreated, res)
}

func (c *DeckCode) Update(ctx *gin.Context) {
	req := helper.GetDeckCodeUpdateRequest(ctx)
	id := helper.GetId(ctx)

	param := usecase.NewDeckCodeUpdateParam(
		req.PrivateCodeFlg,
		req.Memo,
	)

	deckcode, err := c.usecase.Update(context.Background(), id, param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewDeckCodeUpdateResponse(deckcode)

	ctx.JSON(http.StatusOK, res)
}

func (c *DeckCode) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(context.Background(), id); err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
