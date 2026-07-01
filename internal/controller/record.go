package controller

import (
	"context"
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
	RecordsPath = "/records"
)

type Record struct {
	router     *gin.Engine
	repository repository.RecordInterface
	usecase    usecase.RecordInterface
}

func NewRecord(
	router *gin.Engine,
	repository repository.RecordInterface,
	usecase usecase.RecordInterface,
) *Record {
	return &Record{router, repository, usecase}
}

func (c *Record) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + RecordsPath)
	r.GET(
		"",
		authentication.OptionalAuthenticationMiddleware(),
		validation.RecordGetMiddleware(),
		c.Get,
		c.GetByUserId,
	)
	r.GET(
		"/:id",
		authentication.OptionalAuthenticationMiddleware(),
		authorization.RecordGetByIdAuthorizationMiddleware(c.repository),
		c.GetById,
	)
	r.POST(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.RecordCreateMiddleware(),
		c.Create,
	)
	r.PUT(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.RecordUpdateAuthorizationMiddleware(c.repository),
		validation.RecordUpdateMiddleware(),
		c.Update,
	)
	r.DELETE(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.RecordDeleteAuthorizationMiddleware(c.repository),
		c.Delete,
	)
}

func (c *Record) Get(ctx *gin.Context) {
	if uid := helper.GetUID(ctx); uid == "" {
		limit := helper.GetLimit(ctx)
		offset := helper.GetOffset(ctx)
		cursorEventDate := helper.GetCursorEventDate(ctx)
		cursorCreatedAt := helper.GetCursorCreatedAt(ctx)
		eventType := helper.GetEventType(ctx)

		if !cursorCreatedAt.IsZero() {
			records, err := c.usecase.FindOnCursor(context.Background(), limit, cursorEventDate, cursorCreatedAt, eventType)
			if err != nil {
				apierror.ErrInternalServerError.JSON(ctx)
				return
			}

			res := presenter.NewRecordGetResponse(limit, offset, cursorEventDate, cursorCreatedAt, records)

			ctx.JSON(http.StatusOK, res)
		} else {
			records, err := c.usecase.Find(context.Background(), limit, offset, eventType)
			if err != nil {
				apierror.ErrInternalServerError.JSON(ctx)
				return
			}

			res := presenter.NewRecordGetResponse(limit, offset, cursorEventDate, cursorCreatedAt, records)

			ctx.JSON(http.StatusOK, res)
		}
	}
}

func (c *Record) GetByUserId(ctx *gin.Context) {
	if uid := helper.GetUID(ctx); uid != "" {
		limit := helper.GetLimit(ctx)
		offset := helper.GetOffset(ctx)
		cursorEventDate := helper.GetCursorEventDate(ctx)
		cursorCreatedAt := helper.GetCursorCreatedAt(ctx)
		eventType := helper.GetEventType(ctx)
		deckId := helper.GetDeckId(ctx)

		if !cursorCreatedAt.IsZero() {
			if deckId != "" {
				records, err := c.usecase.FindByDeckIdOnCursor(context.Background(), deckId, limit, cursorEventDate, cursorCreatedAt, eventType)

				if err != nil {
					apierror.ErrInternalServerError.JSON(ctx)
					return
				}

				res := presenter.NewRecordGetByUserIdResponse(limit, offset, cursorEventDate, cursorCreatedAt, records)

				ctx.JSON(http.StatusOK, res)
			} else {
				records, err := c.usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursorEventDate, cursorCreatedAt, eventType)

				if err != nil {
					apierror.ErrInternalServerError.JSON(ctx)
					return
				}

				res := presenter.NewRecordGetByUserIdResponse(limit, offset, cursorEventDate, cursorCreatedAt, records)

				ctx.JSON(http.StatusOK, res)
			}
		} else {
			if deckId != "" {
				records, err := c.usecase.FindByDeckId(context.Background(), deckId, limit, offset, eventType)

				if err != nil {
					apierror.ErrInternalServerError.JSON(ctx)
					return
				}

				res := presenter.NewRecordGetByUserIdResponse(limit, offset, cursorEventDate, cursorCreatedAt, records)

				ctx.JSON(http.StatusOK, res)
				return
			} else {
				records, err := c.usecase.FindByUserId(context.Background(), uid, limit, offset, eventType)

				if err != nil {
					apierror.ErrInternalServerError.JSON(ctx)
					return
				}

				res := presenter.NewRecordGetByUserIdResponse(limit, offset, cursorEventDate, cursorCreatedAt, records)

				ctx.JSON(http.StatusOK, res)
			}
		}

	}
}

func (c *Record) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	record, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewRecordGetByIdResponse(record)

	ctx.JSON(http.StatusOK, res)
}

func (c *Record) Create(ctx *gin.Context) {
	req := helper.GetRecordCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	param := usecase.NewRecordParam(
		req.OfficialEventId,
		req.TonamelEventId,
		req.FriendId,
		req.UnofficialEventId,
		uid,
		req.DeckId,
		req.DeckCodeId,
		req.EventDate,
		req.PrivateFlg,
		req.TCGMeisterURL,
		req.Memo,
	)

	record, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewRecordCreateResponse(record)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Record) Update(ctx *gin.Context) {
	req := helper.GetRecordUpdateRequest(ctx)
	id := helper.GetId(ctx)
	uid := helper.GetUID(ctx)

	param := usecase.NewRecordParam(
		req.OfficialEventId,
		req.TonamelEventId,
		req.FriendId,
		req.UnofficialEventId,
		uid,
		req.DeckId,
		req.DeckCodeId,
		req.EventDate,
		req.PrivateFlg,
		req.TCGMeisterURL,
		req.Memo,
	)

	record, err := c.usecase.Update(context.Background(), id, param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewRecordUpdateResponse(record)

	ctx.JSON(http.StatusOK, res)
}

func (c *Record) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(context.Background(), id); err != nil {
		if err == apperror.ErrRecordNotFound {
			apierror.ErrBadRequestNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
