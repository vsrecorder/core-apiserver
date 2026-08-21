package controller

import (
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
	TagsPath = "/tags"
)

type Tag struct {
	router        *gin.Engine
	tagRepository repository.TagInterface
	usecase       usecase.TagInterface
}

func NewTag(
	router *gin.Engine,
	tagRepository repository.TagInterface,
	usecase usecase.TagInterface,
) *Tag {
	return &Tag{router, tagRepository, usecase}
}

func (c *Tag) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + TagsPath)
	r.GET(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		c.GetByUID,
	)
	r.GET(
		"/presets",
		authentication.RequiredAuthenticationMiddleware(),
		c.GetPresets,
	)
	r.POST(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.TagCreateMiddleware(),
		c.Create,
	)
	r.PUT(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.TagUpdateAuthorizationMiddleware(c.tagRepository),
		validation.TagUpdateMiddleware(),
		c.Update,
	)
	r.DELETE(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.TagDeleteAuthorizationMiddleware(c.tagRepository),
		c.Delete,
	)
}

func (c *Tag) GetByUID(ctx *gin.Context) {
	uid := helper.GetUID(ctx)

	tags, err := c.usecase.FindByUserId(ctx.Request.Context(), uid)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewTagGetResponse(tags)

	ctx.JSON(http.StatusOK, res)
}

// GetPresets は全ユーザー共通のプリセットタグ(ACE SPEC など)を返す。
// 認証は必須だがユーザーに依らず同じ結果を返す。
func (c *Tag) GetPresets(ctx *gin.Context) {
	tags, err := c.usecase.FindPresets(ctx.Request.Context())
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewTagGetResponse(tags)

	ctx.JSON(http.StatusOK, res)
}

func (c *Tag) Create(ctx *gin.Context) {
	req := helper.GetTagCreateRequest(ctx)
	uid := helper.GetUID(ctx)

	param := usecase.NewTagCreateParam(uid, req.Name, req.Color)

	tag, err := c.usecase.Create(ctx.Request.Context(), param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewTagCreateResponse(tag)

	ctx.JSON(http.StatusCreated, res)
}

func (c *Tag) Update(ctx *gin.Context) {
	req := helper.GetTagUpdateRequest(ctx)
	id := helper.GetId(ctx)

	param := usecase.NewTagUpdateParam(req.Name, req.Color)

	tag, err := c.usecase.Update(ctx.Request.Context(), id, param)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrNotFound.JSON(ctx, err)
			return
		}
		if errors.Is(err, apperror.ErrAlreadyExists) {
			apierror.ErrConflict.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	res := presenter.NewTagUpdateResponse(tag)

	ctx.JSON(http.StatusOK, res)
}

func (c *Tag) Delete(ctx *gin.Context) {
	id := helper.GetId(ctx)

	if err := c.usecase.Delete(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			apierror.ErrBadRequestNotFound.JSON(ctx, err)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx, err)
		return
	}

	ctx.JSON(http.StatusNoContent, gin.H{})
}
