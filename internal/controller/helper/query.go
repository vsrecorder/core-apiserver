package helper

import (
	"github.com/gin-gonic/gin"
)

func GetQueryLimit(ctx *gin.Context) string {
	return ctx.Query("limit")
}

func GetQueryOffset(ctx *gin.Context) string {
	return ctx.Query("offset")
}

func GetQueryCursor(ctx *gin.Context) string {
	return ctx.Query("cursor")
}

func GetQueryStartDate(ctx *gin.Context) string {
	return ctx.Query("start_date")
}

func GetQueryEndDate(ctx *gin.Context) string {
	return ctx.Query("end_date")
}

func GetQueryTypeId(ctx *gin.Context) string {
	return ctx.Query("type_id")
}

func GetQueryLeagueType(ctx *gin.Context) string {
	return ctx.Query("league_type")
}
