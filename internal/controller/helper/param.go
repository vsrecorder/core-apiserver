package helper

import (
	"github.com/gin-gonic/gin"
)

func GetId(ctx *gin.Context) (id string) {
	return ctx.Param("id")
}
