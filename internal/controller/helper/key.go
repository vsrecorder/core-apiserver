package helper

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/logging"
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

func SetCursor(ctx *gin.Context, value time.Time) {
	ctx.Set("cursor", value)
}

func GetCursor(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("cursor")
	cursor, _ := value.(time.Time)

	return cursor
}

func SetCursorEventDate(ctx *gin.Context, value time.Time) {
	ctx.Set("cursor_event_date", value)
}

func GetCursorEventDate(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("cursor_event_date")
	t, _ := value.(time.Time)
	return t
}

func SetCursorCreatedAt(ctx *gin.Context, value time.Time) {
	ctx.Set("cursor_created_at", value)
}

func GetCursorCreatedAt(ctx *gin.Context) time.Time {
	value, _ := ctx.Get("cursor_created_at")
	t, _ := value.(time.Time)
	return t
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

func SetEventType(ctx *gin.Context, value string) {
	ctx.Set("event_type", value)
}

func GetEventType(ctx *gin.Context) string {
	value, _ := ctx.Get("event_type")
	eventType, _ := value.(string)

	return eventType
}

func SetOfficialEventId(ctx *gin.Context, value uint) {
	ctx.Set("official_event_id", value)
}

func GetOfficialEventId(ctx *gin.Context) uint {
	value, _ := ctx.Get("official_event_id")
	officialEventId, _ := value.(uint)

	return officialEventId
}

func SetDeckId(ctx *gin.Context, value string) {
	ctx.Set("deck_id", value)
}

func GetDeckId(ctx *gin.Context) string {
	value, _ := ctx.Get("deck_id")
	deckId, _ := value.(string)

	return deckId
}

func SetUID(ctx *gin.Context, value string) {
	ctx.Set("uid", value)

	// uid を Request の context にも載せることで、controller が下層へ渡した ctx を
	// 経由して usecase / infrastructure のログにも uid が自動で付与される。
	// 未認証を許容するエンドポイントでは空文字が渡るため、その場合は載せない。
	if value != "" && ctx.Request != nil {
		ctx.Request = ctx.Request.WithContext(
			logging.ContextWithUID(ctx.Request.Context(), value),
		)
	}
}

func GetUID(ctx *gin.Context) string {
	value, _ := ctx.Get("uid")
	uid, _ := value.(string)

	return uid
}

// SetPlayerId はリクエストが対象としているプレイヤーズクラブのplayer_idを保持する。
// アクセスログへ出力するため、バリデーションミドルウェアが取り出した時点で設定する。
func SetPlayerId(ctx *gin.Context, value string) {
	ctx.Set("player_id", value)
}

func GetPlayerId(ctx *gin.Context) string {
	value, _ := ctx.Get("player_id")
	playerId, _ := value.(string)

	return playerId
}

func SetArchived(ctx *gin.Context, value bool) {
	ctx.Set("archived", value)
}

func GetArchived(ctx *gin.Context) bool {
	value, _ := ctx.Get("archived")
	archived, _ := value.(bool)

	return archived
}

func SetAllTime(ctx *gin.Context, value bool) {
	ctx.Set("all_time", value)
}

func GetAllTime(ctx *gin.Context) bool {
	value, _ := ctx.Get("all_time")
	allTime, _ := value.(bool)

	return allTime
}

func SetYearMonth(ctx *gin.Context, value string) {
	ctx.Set("year_month", value)
}

func GetYearMonth(ctx *gin.Context) string {
	value, _ := ctx.Get("year_month")
	yearMonth, _ := value.(string)

	return yearMonth
}

func SetEnvironmentId(ctx *gin.Context, value string) {
	ctx.Set("environment_id", value)
}

func GetEnvironmentId(ctx *gin.Context) string {
	value, _ := ctx.Get("environment_id")
	environmentId, _ := value.(string)

	return environmentId
}

func SetStandardRegulationId(ctx *gin.Context, value string) {
	ctx.Set("standard_regulation_id", value)
}

func GetStandardRegulationId(ctx *gin.Context) string {
	value, _ := ctx.Get("standard_regulation_id")
	standardRegulationId, _ := value.(string)

	return standardRegulationId
}

// レギュレーション区分(スタンダード/エクストラ/殿堂)での絞り込み。
// 0 は「絞り込まない(全レギュレーション)」を表す。
func SetRegulationId(ctx *gin.Context, value uint) {
	ctx.Set("regulation_id", value)
}

func GetRegulationId(ctx *gin.Context) uint {
	value, _ := ctx.Get("regulation_id")
	regulationId, _ := value.(uint)

	return regulationId
}

func SetSeason(ctx *gin.Context, value string) {
	ctx.Set("season", value)
}

func GetSeason(ctx *gin.Context) string {
	value, _ := ctx.Get("season")
	season, _ := value.(string)

	return season
}

func SetWeek(ctx *gin.Context, value string) {
	ctx.Set("week", value)
}

func GetWeek(ctx *gin.Context) string {
	value, _ := ctx.Get("week")
	week, _ := value.(string)

	return week
}

func SetPeriod(ctx *gin.Context, value string) {
	ctx.Set("period", value)
}

func GetPeriod(ctx *gin.Context) string {
	value, _ := ctx.Get("period")
	period, _ := value.(string)

	return period
}

func SetUnofficialEventCreateRequest(ctx *gin.Context, value dto.UnofficialEventCreateRequest) {
	ctx.Set("unofficial_event_create_request", value)
}

func GetUnofficialEventCreateRequest(ctx *gin.Context) dto.UnofficialEventCreateRequest {
	value, _ := ctx.Get("unofficial_event_create_request")
	unofficialEventRequest, _ := value.(dto.UnofficialEventCreateRequest)

	return unofficialEventRequest
}

func SetUnofficialEventUpdateRequest(ctx *gin.Context, value dto.UnofficialEventUpdateRequest) {
	ctx.Set("unofficial_event_update_request", value)
}

func GetUnofficialEventUpdateRequest(ctx *gin.Context) dto.UnofficialEventUpdateRequest {
	value, _ := ctx.Get("unofficial_event_update_request")
	unofficialEventRequest, _ := value.(dto.UnofficialEventUpdateRequest)

	return unofficialEventRequest
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

func SetDeckCodeCreateRequest(ctx *gin.Context, value dto.DeckCodeCreateRequest) {
	ctx.Set("deck_code_create_request", value)
}

func GetDeckCodeCreateRequest(ctx *gin.Context) dto.DeckCodeCreateRequest {
	value, _ := ctx.Get("deck_code_create_request")
	ret, _ := value.(dto.DeckCodeCreateRequest)

	return ret
}

func SetDeckCodeUpdateRequest(ctx *gin.Context, value dto.DeckCodeUpdateRequest) {
	ctx.Set("deck_code_update_request", value)
}

func GetDeckCodeUpdateRequest(ctx *gin.Context) dto.DeckCodeUpdateRequest {
	value, _ := ctx.Get("deck_code_update_request")
	ret, _ := value.(dto.DeckCodeUpdateRequest)

	return ret
}

func SetTagPresetCategory(ctx *gin.Context, value string) {
	ctx.Set("tag_preset_category", value)
}

func GetTagPresetCategory(ctx *gin.Context) string {
	value, _ := ctx.Get("tag_preset_category")
	ret, _ := value.(string)

	return ret
}

func SetTagCreateRequest(ctx *gin.Context, value dto.TagCreateRequest) {
	ctx.Set("tag_create_request", value)
}

func GetTagCreateRequest(ctx *gin.Context) dto.TagCreateRequest {
	value, _ := ctx.Get("tag_create_request")
	ret, _ := value.(dto.TagCreateRequest)

	return ret
}

func SetTagUpdateRequest(ctx *gin.Context, value dto.TagUpdateRequest) {
	ctx.Set("tag_update_request", value)
}

func GetTagUpdateRequest(ctx *gin.Context) dto.TagUpdateRequest {
	value, _ := ctx.Get("tag_update_request")
	ret, _ := value.(dto.TagUpdateRequest)

	return ret
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

func SetMatchReorderRequest(ctx *gin.Context, value dto.MatchReorderRequest) {
	ctx.Set("match_reorder_request", value)
}

func GetMatchReorderRequest(ctx *gin.Context) dto.MatchReorderRequest {
	value, _ := ctx.Get("match_reorder_request")
	matchReorderRequest, _ := value.(dto.MatchReorderRequest)

	return matchReorderRequest
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

func SetUserPlayerCreateRequest(ctx *gin.Context, value dto.UserPlayerCreateRequest) {
	ctx.Set("user_player_create_request", value)
}

func GetUserPlayerCreateRequest(ctx *gin.Context) dto.UserPlayerCreateRequest {
	value, _ := ctx.Get("user_player_create_request")
	ret, _ := value.(dto.UserPlayerCreateRequest)

	return ret
}

func SetPushSubscriptionCreateRequest(ctx *gin.Context, value dto.PushSubscriptionCreateRequest) {
	ctx.Set("push_subscription_create_request", value)
}

func GetPushSubscriptionCreateRequest(ctx *gin.Context) dto.PushSubscriptionCreateRequest {
	value, _ := ctx.Get("push_subscription_create_request")
	ret, _ := value.(dto.PushSubscriptionCreateRequest)

	return ret
}

func SetPushSubscriptionDeleteRequest(ctx *gin.Context, value dto.PushSubscriptionDeleteRequest) {
	ctx.Set("push_subscription_delete_request", value)
}

func GetPushSubscriptionDeleteRequest(ctx *gin.Context) dto.PushSubscriptionDeleteRequest {
	value, _ := ctx.Get("push_subscription_delete_request")
	ret, _ := value.(dto.PushSubscriptionDeleteRequest)

	return ret
}

func SetUserAcquisitionCreateRequest(ctx *gin.Context, value dto.UserAcquisitionCreateRequest) {
	ctx.Set("user_acquisition_create_request", value)
}

func GetUserAcquisitionCreateRequest(ctx *gin.Context) dto.UserAcquisitionCreateRequest {
	value, _ := ctx.Get("user_acquisition_create_request")
	ret, _ := value.(dto.UserAcquisitionCreateRequest)

	return ret
}

func SetUserAcquisitionSurveyRequest(ctx *gin.Context, value dto.UserAcquisitionSurveyRequest) {
	ctx.Set("user_acquisition_survey_request", value)
}

func GetUserAcquisitionSurveyRequest(ctx *gin.Context) dto.UserAcquisitionSurveyRequest {
	value, _ := ctx.Get("user_acquisition_survey_request")
	ret, _ := value.(dto.UserAcquisitionSurveyRequest)

	return ret
}

func SetKeyword(ctx *gin.Context, value string) {
	ctx.Set("keyword", value)
}

func GetKeyword(ctx *gin.Context) string {
	value, _ := ctx.Get("keyword")
	keyword, _ := value.(string)

	return keyword
}

func SetShopId(ctx *gin.Context, value uint) {
	ctx.Set("shop_id", value)
}

func GetShopId(ctx *gin.Context) uint {
	value, _ := ctx.Get("shop_id")
	shopId, _ := value.(uint)

	return shopId
}

func SetUserGymCreateRequest(ctx *gin.Context, value dto.UserGymCreateRequest) {
	ctx.Set("user_gym_create_request", value)
}

func GetUserGymCreateRequest(ctx *gin.Context) dto.UserGymCreateRequest {
	value, _ := ctx.Get("user_gym_create_request")
	ret, _ := value.(dto.UserGymCreateRequest)

	return ret
}

func SetChampionsleagueScheduleId(ctx *gin.Context, value string) {
	ctx.Set("championsleague_schedule_id", value)
}

func GetChampionsleagueScheduleId(ctx *gin.Context) string {
	value, _ := ctx.Get("championsleague_schedule_id")
	championsleagueScheduleId, _ := value.(string)

	return championsleagueScheduleId
}

// SetSort / GetSort は一覧の並び順(みんなの公開デッキの new / popular)。
func SetSort(ctx *gin.Context, value string) {
	ctx.Set("sort", value)
}

func GetSort(ctx *gin.Context) string {
	value, _ := ctx.Get("sort")
	sort, _ := value.(string)

	return sort
}

// SetAceSpecCardId / GetAceSpecCardId は ACE SPEC カードIDでの絞り込み。空は絞り込みなし。
func SetAceSpecCardId(ctx *gin.Context, value string) {
	ctx.Set("ace_spec_card_id", value)
}

func GetAceSpecCardId(ctx *gin.Context) string {
	value, _ := ctx.Get("ace_spec_card_id")
	id, _ := value.(string)

	return id
}

// SetPokemonSpriteIds / GetPokemonSpriteIds はスプライトでの絞り込み(最大2体)。空は絞り込みなし。
func SetPokemonSpriteIds(ctx *gin.Context, value []string) {
	ctx.Set("pokemon_sprite_ids", value)
}

func GetPokemonSpriteIds(ctx *gin.Context) []string {
	value, _ := ctx.Get("pokemon_sprite_ids")
	ids, _ := value.([]string)

	return ids
}

func SetDeckCodePostCreateRequest(ctx *gin.Context, value dto.DeckCodePostCreateRequest) {
	ctx.Set("deck_code_post_create_request", value)
}

func GetDeckCodePostCreateRequest(ctx *gin.Context) dto.DeckCodePostCreateRequest {
	value, _ := ctx.Get("deck_code_post_create_request")
	req, _ := value.(dto.DeckCodePostCreateRequest)

	return req
}
