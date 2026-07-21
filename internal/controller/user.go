package controller

import (
	"context"
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
	UsersPath = "/users"
)

type User struct {
	logger     *slog.Logger
	router     *gin.Engine
	repository repository.UserInterface
	usecase    usecase.UserInterface
}

func NewUser(
	logger *slog.Logger,
	router *gin.Engine,
	repository repository.UserInterface,
	usecase usecase.UserInterface,
) *User {
	return &User{logger, router, repository, usecase}
}

func (c *User) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id",
		c.GetById,
	)
	r.POST(
		"",
		authentication.RequiredAuthenticationMiddleware(),
		validation.UserCreateMiddleware(),
		c.Create,
	)
	r.PUT(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.UserUpdateAuthorizationMiddleware(c.repository),
		validation.UserUpdateMiddleware(),
		c.Update,
	)
	r.DELETE(
		"/:id",
		authentication.RequiredAuthenticationMiddleware(),
		authorization.UserDeleteAuthorizationMiddleware(c.repository),
		c.Delete,
	)
}

func (c *User) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	user, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if err == apperror.ErrRecordNotFound {
			apierror.ErrNotFound.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserGetByIdResponse(user)

	ctx.JSON(http.StatusOK, res)
}

func (c *User) Create(ctx *gin.Context) {
	req := helper.GetUserCreateRequest(ctx)
	id := helper.GetUID(ctx)

	param := usecase.NewUserCreateParam(
		id,
		req.Name,
		req.ImageURL,
	)

	user, err := c.usecase.Create(context.Background(), param)
	if err != nil {
		if errors.Is(err, apperror.ErrAlreadyExists) {
			apierror.ErrConflict.JSON(ctx)
			return
		}

		// 退会済みのユーザーによる再登録。未登録(404)と区別できるよう410で返す。
		if errors.Is(err, apperror.ErrWithdrawn) {
			apierror.ErrGone.JSON(ctx)
			return
		}

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserCreateResponse(user)

	ctx.JSON(http.StatusCreated, res)
}

func (c *User) Update(ctx *gin.Context) {
	req := helper.GetUserUpdateRequest(ctx)
	id := helper.GetId(ctx)

	param := usecase.NewUserUpdateParam(
		req.Name,
		req.ImageURL,
	)

	user, err := c.usecase.Update(context.Background(), id, param)
	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	res := presenter.NewUserCreateResponse(user)

	ctx.JSON(http.StatusOK, res)
}

func (c *User) Delete(ctx *gin.Context) {
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
