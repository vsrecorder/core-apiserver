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

func SetDate(ctx *gin.Context, value time.Time) {
	ctx.Set("date", value)
}

func GetDate(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("date")
	date, _ := value.(time.Time)

	return date
}

func SetFromDate(ctx *gin.Context, value time.Time) {
	ctx.Set("from_date", value)
}

func GetFromDate(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("from_date")
	fromDate, _ := value.(time.Time)

	return fromDate
}

func SetToDate(ctx *gin.Context, value time.Time) {
	ctx.Set("to_date", value)
}

func GetToDate(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("to_date")
	toDate, _ := value.(time.Time)

	return toDate
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

func SetArchived(ctx *gin.Context, value bool) {
	ctx.Set("archived", value)
}

func GetArchived(ctx *gin.Context) bool {
	value, _ := ctx.Get("archived")
	archived, _ := value.(bool)

	return archived
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

func SetDeckCreateRequest(ctx *gin.Context, value dto.DeckCreateRequest) {
	ctx.Set("deck_create_request", value)
}

func GetDeckCreateRequest(ctx *gin.Context) dto.DeckCreateRequest {
	value, _ := ctx.Get("deck_create_request")
	deckRequest, _ := value.(dto.DeckCreateRequest)

	return deckRequest
}

func SetDeckUpdateRequest(ctx *gin.Context, value dto.DeckUpdateRequest) {
	ctx.Set("deck_update_request", value)
}

func GetDeckUpdateRequest(ctx *gin.Context) dto.DeckUpdateRequest {
	value, _ := ctx.Get("deck_update_request")
	deckRequest, _ := value.(dto.DeckUpdateRequest)

	return deckRequest
}

func SetMatchCreateRequest(ctx *gin.Context, value dto.MatchCreateRequest) {
	ctx.Set("match_create_request", value)
}

func GetMatchCreateRequest(ctx *gin.Context) dto.MatchCreateRequest {
	value, _ := ctx.Get("match_create_request")
	matchRequest, _ := value.(dto.MatchCreateRequest)

	return matchRequest
}

func SetMatchUpdateRequest(ctx *gin.Context, value dto.MatchUpdateRequest) {
	ctx.Set("match_update_request", value)
}

func GetMatchUpdateRequest(ctx *gin.Context) dto.MatchUpdateRequest {
	value, _ := ctx.Get("match_update_request")
	matchRequest, _ := value.(dto.MatchUpdateRequest)

	return matchRequest
}

func SetUserCreateRequest(ctx *gin.Context, value dto.UserCreateRequest) {
	ctx.Set("user_create_request", value)
}

func GetUserCreateRequest(ctx *gin.Context) dto.UserCreateRequest {
	value, _ := ctx.Get("user_create_request")
	userRequest, _ := value.(dto.UserCreateRequest)

	return userRequest
}

func SetUserUpdateRequest(ctx *gin.Context, value dto.UserUpdateRequest) {
	ctx.Set("user_update_request", value)
}

func GetUserUpdateRequest(ctx *gin.Context) dto.UserUpdateRequest {
	value, _ := ctx.Get("user_update_request")
	userRequest, _ := value.(dto.UserUpdateRequest)

	return userRequest
}
