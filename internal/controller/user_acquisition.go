package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	UserAcquisitionPath       = "/acquisition"
	UserAcquisitionSurveyPath = "/acquisition/survey"
)

type UserAcquisition struct {
	router  *gin.Engine
	usecase usecase.UserAcquisitionInterface
}

func NewUserAcquisition(
	router *gin.Engine,
	usecase usecase.UserAcquisitionInterface,
) *UserAcquisition {
	return &UserAcquisition{router, usecase}
}

func (c *UserAcquisition) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	// 書き込みなので uid はパスパラメータではなく認証済みトークンから取る(activity と同じ)。
	// 呼び出し元は webapp の authorize() で、ユーザー作成が 201 で返った直後に1回だけ叩く。
	r.POST(
		UserAcquisitionPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.UserAcquisitionCreateMiddleware(),
		c.Record,
	)
	// 登録時アンケート(S4)。回答も本人のトークンでしか書けない。
	r.POST(
		UserAcquisitionSurveyPath,
		authentication.RequiredAuthenticationMiddleware(),
		validation.UserAcquisitionSurveyMiddleware(),
		c.AnswerSurvey,
	)
}

func (c *UserAcquisition) Record(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	req := helper.GetUserAcquisitionCreateRequest(ctx)

	param := &usecase.UserAcquisitionRecordParam{
		Source:      req.Source,
		Medium:      req.Medium,
		Campaign:    req.Campaign,
		Content:     req.Content,
		Referrer:    req.Referrer,
		LandingPath: req.LandingPath,
		LandingAt:   req.LandingAt,
	}

	if err := c.usecase.Record(ctx.Request.Context(), uid, param); err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	// 返す情報はない
	ctx.Status(http.StatusNoContent)
}

func (c *UserAcquisition) AnswerSurvey(ctx *gin.Context) {
	uid := helper.GetUID(ctx)
	req := helper.GetUserAcquisitionSurveyRequest(ctx)

	if err := c.usecase.AnswerSurvey(ctx.Request.Context(), uid, req.Answer); err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	// 返す情報はない
	ctx.Status(http.StatusNoContent)
}
