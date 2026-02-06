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

func (c *Record) RegisterRoute(relativePath string, authDisable bool) {
	if authDisable {
		r := c.router.Group(relativePath + RecordsPath)
		r.GET(
			"",
			validation.RecordGetMiddleware(),
			c.Get,
			c.GetByUserId,
		)
		r.GET(
			"/:id",
			c.GetById,
		)
		r.POST(
			"",
			validation.RecordCreateMiddleware(),
			c.Create,
		)
		r.PUT(
			"/:id",
			validation.RecordUpdateMiddleware(),
			c.Update,
		)
		r.DELETE(
			"/:id",
			c.Delete,
		)
	} else {
		r := c.router.Group(relativePath + RecordsPath)
		r.GET(
			"",
			auth.OptionalAuthenticationMiddleware(),
			validation.RecordGetMiddleware(),
			c.Get,
			c.GetByUserId,
		)
		r.GET(
			"/:id",
			auth.OptionalAuthenticationMiddleware(),
			auth.RecordGetByIdAuthorizationMiddleware(c.repository),
			c.GetById,
		)
		r.POST(
			"",
			auth.RequiredAuthenticationMiddleware(),
			validation.RecordCreateMiddleware(),
			c.Create,
		)
		r.PUT(
			"/:id",
			auth.RequiredAuthenticationMiddleware(),
			auth.RecordUpdateAuthorizationMiddleware(c.repository),
			validation.RecordUpdateMiddleware(),
			c.Update,
		)
		r.DELETE(
			"/:id",
			auth.RequiredAuthenticationMiddleware(),
			auth.RecordDeleteAuthorizationMiddleware(c.repository),
			c.Delete,
		)
	}
}

func (c *Record) Get(ctx *gin.Context) {
	if uid := helper.GetUID(ctx); uid == "" {
		limit := helper.GetLimit(ctx)
		offset := helper.GetOffset(ctx)
		cursor := helper.GetCursor(ctx)
		eventType := helper.GetEventType(ctx)

		if !cursor.IsZero() {
			records, err := c.usecase.FindOnCursor(context.Background(), limit, cursor, eventType)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
				return
			}

			res := presenter.NewRecordGetResponse(limit, offset, cursor, records)

			ctx.JSON(http.StatusOK, res)
		} else {
			records, err := c.usecase.Find(context.Background(), limit, offset, eventType)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
				return
			}

			res := presenter.NewRecordGetResponse(limit, offset, cursor, records)

			ctx.JSON(http.StatusOK, res)
		}
	}
}

func (c *Record) GetByUserId(ctx *gin.Context) {
	if uid := helper.GetUID(ctx); uid != "" {
		limit := helper.GetLimit(ctx)
		offset := helper.GetOffset(ctx)
		cursor := helper.GetCursor(ctx)
		eventType := helper.GetEventType(ctx)

		if !cursor.IsZero() {
			records, err := c.usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursor, eventType)

			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
				return
			}

			res := presenter.NewRecordGetByUserIdResponse(limit, offset, cursor, records)

			ctx.JSON(http.StatusOK, res)
		} else {
			records, err := c.usecase.FindByUserId(context.Background(), uid, limit, offset, eventType)

			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
				ctx.Abort()
				return
			}

			res := presenter.NewRecordGetByUserIdResponse(limit, offset, cursor, records)

			ctx.JSON(http.StatusOK, res)
		}

	}
}

func (c *Record) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	record, err := c.usecase.FindById(context.Background(), id)
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
		uid,
		req.DeckId,
		req.DeckCodeId,
		req.PrivateFlg,
		req.TCGMeisterURL,
		req.Memo,
	)

	record, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
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
		uid,
		req.DeckId,
		req.DeckCodeId,
		req.PrivateFlg,
		req.TCGMeisterURL,
		req.Memo,
	)

	record, err := c.usecase.Update(context.Background(), id, param)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewRecordUpdateResponse(record)

	ctx.JSON(http.StatusOK, res)
}

func (c *Record) Delete(ctx *gin.Context) {
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
