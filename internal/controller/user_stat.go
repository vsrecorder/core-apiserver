package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/controller/presenter"
	"github.com/vsrecorder/core-apiserver/internal/controller/validation"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
	"gorm.io/gorm"
)

const (
	UserStatsPath = "/stats"
)

type UserStat struct {
	router  *gin.Engine
	usecase usecase.UserStatInterface
}

func NewUserStat(
	router *gin.Engine,
	usecase usecase.UserStatInterface,
) *UserStat {
	return &UserStat{router, usecase}
}

func (c *UserStat) RegisterRoute(relativePath string) {
	r := c.router.Group(relativePath + UsersPath)
	r.GET(
		"/:id"+UserStatsPath,
		validation.UserStatGetMiddleware(),
		c.GetByUserId,
	)
}

func (c *UserStat) GetByUserId(ctx *gin.Context) {
	uid := helper.GetId(ctx)
	yearMonth := helper.GetYearMonth(ctx)
	environmentId := helper.GetEnvironmentId(ctx)
	season := helper.GetSeason(ctx)

	stats, err := c.usecase.GetUserStat(context.Background(), uid, yearMonth, environmentId, season)
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

	res := presenter.NewUserStatResponse(stats, yearMonth, environmentId, season)

	ctx.JSON(http.StatusOK, res)
}
