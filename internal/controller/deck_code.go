package controller

import (
	"errors"
	"log/slog"
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
	DeckCodesPath = "/deckcodes"
)

type DeckCode struct {
	logger             *slog.Logger
	router             *gin.Engine
	deckcodeRepository repository.DeckCodeInterface
	// deckRepository は参照系の認可(親デッキが非公開なら他人に見せない)に使う。
	deckRepository   repository.DeckInterface
	recordRepository repository.RecordInterface
	usecase          usecase.DeckCodeInterface
}

func NewDeckCode(
	logger *slog.Logger,
	router *gin.Engine,
	deckcodeRepository repository.DeckCodeInterface,
	deckRepository repository.DeckInterface,
	recordRepository repository.RecordInterface,
	usecase usecase.DeckCodeInterface,
) *DeckCode {
	return &DeckCode{logger, router, deckcodeRepository, deckRepository, recordRepository, usecase}
}

func (c *DeckCode) RegisterRoute(relativePath string) {
	{
		r := c.router.Group(relativePath + DeckCodesPath)
		// 参照はデッキ本体(GET /decks/:id)と同じ公開範囲に揃える。非公開デッキのコードは他人に 403。
		r.GET(
			"/:id",
			authentication.OptionalAuthenticationMiddleware(),
			authorization.DeckCodeGetByIdAuthorizationMiddleware(c.deckcodeRepository, c.deckRepository),
			c.GetById,
		)
		r.POST(
			"",
			authentication.RequiredAuthenticationMiddleware(),
			validation.DeckCodeCreateMiddleware(c.logger),
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
		// デッキ別の一覧もデッキ本体と同じ公開範囲(非公開デッキは他人に 403)。
		r.GET(
			"/:id"+DeckCodesPath,
			authentication.OptionalAuthenticationMiddleware(),
			authorization.DeckGetByIdAuthorizationMiddleware(c.deckRepository),
			c.GetByDeckId,
		)
	}
}

func (c *DeckCode) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	deckcode, err := c.usecase.FindById(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
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

	deckcodes, err := c.usecase.FindByDeckId(ctx.Request.Context(), deckId)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
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
		req.TagIds,
	)

	deckcode, err := c.usecase.Create(ctx.Request.Context(), param)
	if err != nil {
		// deck_id が存在しない、または他人のデッキ(usecase は区別せず「存在しない」として返す)。
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
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
		req.TagIds,
	)

	deckcode, err := c.usecase.Update(ctx.Request.Context(), id, param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewDeckCodeUpdateResponse(deckcode)

	ctx.JSON(http.StatusOK, res)
}

func (c *DeckCode) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(ctx.Request.Context(), id); err != nil {
		if err == apperror.ErrRecordNotFound {
			apierror.ErrBadRequestNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
