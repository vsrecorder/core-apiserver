package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
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
	router             *gin.Engine
	deckcodeRepository repository.DeckCodeInterface
	recordRepository   repository.RecordInterface
	usecase            usecase.DeckCodeInterface
}

func NewDeckCode(
	router *gin.Engine,
	deckcodeRepository repository.DeckCodeInterface,
	recordRepository repository.RecordInterface,
	usecase usecase.DeckCodeInterface,
) *DeckCode {
	return &DeckCode{router, deckcodeRepository, recordRepository, usecase}
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
				authentication.OptionalAuthenticationMiddleware(),
				c.GetById,
			)
			r.POST(
				"",
				authentication.RequiredAuthenticationMiddleware(),
				validation.DeckCodeCreateMiddleware(),
				c.Create,
			)
			r.PUT(
				"/:id",
				authentication.RequiredAuthenticationMiddleware(),
				authorization.DeckCodeUpdateAuthorizationMiddleware(c.deckcodeRepository),
				validation.DeckCodeUpdateMiddleware(),
				c.Update,
			)
			r.DELETE(
				"/:id",
				authentication.RequiredAuthenticationMiddleware(),
				authorization.DeckCodeDeleteAuthorizationMiddleware(c.deckcodeRepository, c.recordRepository),
				c.Delete,
			)
		}

		{
			r := c.router.Group(relativePath + DecksPath)
			r.GET(
				"/:id"+DeckCodesPath,
				authentication.OptionalAuthenticationMiddleware(),
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
