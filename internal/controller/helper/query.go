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

func GetQueryOfficialEventId(ctx *gin.Context) string {
	return ctx.Query("official_event_id")
}

func GetQueryDeckId(ctx *gin.Context) string {
	return ctx.Query("deck_id")
}

func GetQueryCount(ctx *gin.Context) string {
	return ctx.Query("count")
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

// GetQueryTagPresetCategory は GET /tags/presets の絞り込み(プリセットの群)。
func GetQueryTagPresetCategory(ctx *gin.Context) string {
	return ctx.Query("category")
}

func GetQueryArchived(ctx *gin.Context) string {
	return ctx.Query("archived")
}

func GetQueryAllTime(ctx *gin.Context) string {
	return ctx.Query("all_time")
}

func GetQueryYearMonth(ctx *gin.Context) string {
	return ctx.Query("year_month")
}

func GetQueryEnvironmentId(ctx *gin.Context) string {
	return ctx.Query("environment_id")
}

// GetQueryStandardRegulationId は期間の絞り込みに使うスタンダードレギュレーション
// (『H・I・J』などのマークの組み合わせと、その適用期間)のID。
// レギュレーション区分(スタンダード/エクストラ/殿堂)の regulation_id とは別物。
func GetQueryStandardRegulationId(ctx *gin.Context) string {
	return ctx.Query("standard_regulation_id")
}

func GetQueryRegulationId(ctx *gin.Context) string {
	return ctx.Query("regulation_id")
}

func GetQuerySeason(ctx *gin.Context) string {
	return ctx.Query("season")
}

func GetQueryPeriod(ctx *gin.Context) string {
	return ctx.Query("period")
}

func GetQueryWeek(ctx *gin.Context) string {
	return ctx.Query("week")
}
