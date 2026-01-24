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

func GetQueryDate(ctx *gin.Context) string {
	return ctx.Query("date")
}

func GetQueryFromDate(ctx *gin.Context) string {
	return ctx.Query("from_date")
}

func GetQueryToDate(ctx *gin.Context) string {
	return ctx.Query("to_date")
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

func GetQueryEventType(ctx *gin.Context) string {
	return ctx.Query("event_type")
}

func GetQueryArchived(ctx *gin.Context) string {
	return ctx.Query("archived")
}
