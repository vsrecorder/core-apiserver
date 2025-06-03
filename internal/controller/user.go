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
	UsersPath = "/users"
)

type User struct {
	router     *gin.Engine
	repository repository.UserInterface
	usecase    usecase.UserInterface
}

func NewUser(
	router *gin.Engine,
	repository repository.UserInterface,
	usecase usecase.UserInterface,
) *User {
	return &User{router, repository, usecase}
}

func (c *User) RegisterRoute(relativePath string, authDisable bool) {
	if authDisable {
		r := c.router.Group(relativePath + UsersPath)
		r.GET(
			"/:id",
			c.GetById,
		)
		r.POST(
			"",
			validation.UserCreateMiddleware(),
			c.Create,
		)
		r.PUT(
			"/:id",
			validation.UserUpdateMiddleware(),
			c.Update,
		)
		r.DELETE(
			"/:id",
			c.Delete,
		)
	} else {
		r := c.router.Group(relativePath + UsersPath)
		r.GET(
			"/:id",
			c.GetById,
		)
		r.POST(
			"",
			auth.RequiredAuthenticationMiddleware(),
			validation.UserCreateMiddleware(),
			c.Create,
		)
		r.PUT(
			"/:id",
			auth.RequiredAuthenticationMiddleware(),
			auth.UserUpdateAuthorizationMiddleware(c.repository),
			validation.UserUpdateMiddleware(),
			c.Update,
		)
		r.DELETE(
			"/:id",
			auth.RequiredAuthenticationMiddleware(),
			auth.UserDeleteAuthorizationMiddleware(c.repository),
			c.Delete,
		)
	}
}

func (c *User) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	user, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "not found"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
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
		if err == errors.New("already exists") {
			ctx.JSON(http.StatusConflict, gin.H{"message": "already exists"})
			ctx.Abort()
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewUserCreateResponse(user)

	ctx.JSON(http.StatusOK, res)
}

func (c *User) Delete(ctx *gin.Context) {
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

	ctx.JSON(http.StatusAccepted, gin.H{
		"message": "accepted",
	})
}
