package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	USERS_PATH = "/users"
)

type User struct {
	router  *gin.Engine
	usecase usecase.UserInterface
}

func NewUser(
	router *gin.Engine,
	usecase usecase.UserInterface,
) *User {
	return &User{router, usecase}
}

func (c *User) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + USERS_PATH)
	r.GET(
		"/:id",
		c.GetById,
	)
}

func (c *User) GetById(ctx *gin.Context) {
	id := helper.GetId(ctx)

	user, err := c.usecase.FindById(context.Background(), id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	res := presenter.NewUserGetByIdResponse(user)

	ctx.JSON(http.StatusOK, res)
}
