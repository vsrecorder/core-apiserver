package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authorization"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	OldestRecordPath = "/oldest_record_event_date"
)

type OldestRecord struct {
	router  *gin.Engine
	usecase usecase.OldestRecordInterface
}

func NewOldestRecord(
	router *gin.Engine,
	usecase usecase.OldestRecordInterface,
) *OldestRecord {
	return &OldestRecord{router, usecase}
}

func (c *OldestRecord) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+OldestRecordPath,
		authentication.RequiredAuthenticationMiddleware(),
		authorization.OldestRecordAuthorizationMiddleware(),
		validation.OldestRecordGetMiddleware(),
		c.GetByUserId,
	)
}

func (c *OldestRecord) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	deckId := helper.GetDeckId(ctx)

	record, err := c.usecase.GetOldestRecord(context.Background(), uid, deckId)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewOldestRecordResponse(record, uid, deckId)

	ctx.JSON(http.StatusOK, res)
}
