package helper

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

func SetLimit(ctx *gin.Context, value int) {
	ctx.Set("limit", value)
}

func GetLimit(ctx *gin.Context) int {
	value, _ := ctx.Get("limit")
	limit, _ := value.(int)

	return limit

}

func SetOffset(ctx *gin.Context, value int) {
	ctx.Set("offset", value)
}

func GetOffset(ctx *gin.Context) int {
	value, _ := ctx.Get("offset")
	offset, _ := value.(int)

	return offset
}

func SetStartDate(ctx *gin.Context, value time.Time) {
	ctx.Set("start_date", value)
}

func GetStartDate(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("start_date")
	startDate, _ := value.(time.Time)

	return startDate
}

func SetEndDate(ctx *gin.Context, value time.Time) {
	ctx.Set("end_date", value)
}

func GetEndDate(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("end_date")
	endDate, _ := value.(time.Time)

	return endDate
}

func SetTypeId(ctx *gin.Context, value uint) {
	ctx.Set("type_id", value)
}

func GetTypeId(ctx *gin.Context) uint {
	value, _ := ctx.Get("type_id")
	typeId, _ := value.(uint)

	return typeId
}

func SetLeagueType(ctx *gin.Context, value uint) {
	ctx.Set("league_type", value)
}

func GetLeagueType(ctx *gin.Context) uint {
	value, _ := ctx.Get("league_type")
	leagueType, _ := value.(uint)

	return leagueType
}

func SetOfficialEventId(ctx *gin.Context, value uint) {
	ctx.Set("official_event_id", value)
}

func GetOfficialEventId(ctx *gin.Context) uint {
	value, _ := ctx.Get("official_event_id")
	officialEventId, _ := value.(uint)

	return officialEventId
}

func SetUID(ctx *gin.Context, value string) {
	ctx.Set("uid", value)
}

func GetUID(ctx *gin.Context) string {
	value, _ := ctx.Get("uid")
	uid, _ := value.(string)

	return uid
}

func SetRecordCreateRequest(ctx *gin.Context, value dto.RecordCreateRequest) {
	ctx.Set("record_create_request", value)
}

func GetRecordCreateRequest(ctx *gin.Context) dto.RecordCreateRequest {
	value, _ := ctx.Get("record_create_request")
	recordRequest, _ := value.(dto.RecordCreateRequest)

	return recordRequest
}

func SetRecordUpdateRequest(ctx *gin.Context, value dto.RecordUpdateRequest) {
	ctx.Set("record_update_request", value)
}

func GetRecordUpdateRequest(ctx *gin.Context) dto.RecordUpdateRequest {
	value, _ := ctx.Get("record_update_request")
	recordRequest, _ := value.(dto.RecordUpdateRequest)

	return recordRequest
}
