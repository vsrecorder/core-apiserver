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
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	CalendarPath = "/calendar"
)

type Calendar struct {
	router  *gin.Engine
	usecase usecase.CalendarInterface
}

func NewCalendar(
	router *gin.Engine,
	usecase usecase.CalendarInterface,
) *Calendar {
	return &Calendar{router, usecase}
}

func (c *Calendar) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+CalendarPath,
		authentication.RequiredAuthenticationMiddleware(),
		authorization.CalendarAuthorizationMiddleware(),
		c.GetByUserId,
	)
}

// GetByUserId は活動ログのカレンダーに必要な全データを1回で返す。
//
// 記録・対戦結果・デッキ・デッキコードと、それらが参照するイベント情報をまとめて返すため、
// 呼び出し側は記録1件ごとにAPIを呼ぶ必要がない。
func (c *Calendar) GetByUserId(ctx *gin.Context) {
	userId := helper.GetId(ctx)

	calendar, err := c.usecase.GetCalendar(context.Background(), userId)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewCalendarGetByUserIdResponse(calendar)

	ctx.JSON(http.StatusOK, res)
}
