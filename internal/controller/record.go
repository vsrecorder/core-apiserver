package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	RECORDS_PATH = "/records"
)

type Record struct {
	router  *gin.Engine
	usecase usecase.RecordInterface
}

func NewRecord(
	router *gin.Engine,
	usecase usecase.RecordInterface,
) *Record {
	return &Record{router, usecase}
}

func (c *Record) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + RECORDS_PATH)
	r.GET("", validation.RecordGetMiddleware(), c.Get)
	r.GET("/:id", c.GetById)
	r.POST("", validation.RecordCreateMiddleware(), c.Create)
	r.PUT("/:id", validation.RecordUpdateMiddleware(), c.Update)
	r.DELETE("/:id", c.Delete)
}

func (c *Record) Get(ctx *gin.Context) {
	limit := helper.GetLimit(ctx)
	offset := helper.GetOffset(ctx)

	records, err := c.usecase.Find(context.Background(), limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewRecordGetResponse(limit, offset, records)

	ctx.JSON(http.StatusOK, res)
}

func (c *Record) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	record, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "record not found"})
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

	res := presenter.NewRecordCreateResponse(record)

	ctx.JSON(http.StatusOK, res)
}

func (c *Record) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(context.Background(), id); err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "record not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{
		"message": "accepted",
	})
}
