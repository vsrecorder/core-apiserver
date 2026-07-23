package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// cityLeagueTypeId・trainersLeagueTypeId は official_events.type_id のうち
// シティリーグ・トレーナーズリーグを表す区分(webapp側の officialEventHelpers.ts と同じ判定条件)。
const (
	cityLeagueTypeId     = 2
	trainersLeagueTypeId = 3
)

type DesignationStats struct {
	db *gorm.DB
}

func NewDesignationStats(
	db *gorm.DB,
) repository.DesignationStatsInterface {
	return &DesignationStats{db}
}

// existsMatchForRecordCondition は、records に対戦結果(matches)が1件以上
// 紐づいていることを求める条件(駆け出し・見習いの達成条件)。
const existsMatchForRecordCondition = "EXISTS (" +
	"SELECT 1 FROM matches WHERE matches.record_id = records.id AND matches.deleted_at IS NULL" +
	")"

// hasDeckForRecordCondition は、records にデッキ(deck_id もしくは deck_code_id)が
// 指定されていることを求める条件(駆け出し・見習いの達成条件)。デッキ未指定のまま
// 対戦結果だけを追加したケースを「オンボーディング(はじめの一歩)未完了」として除外する。
// deck_id/deck_code_id は未指定時にNULLではなく空文字列で保存される(deck_usage_stat.goの
// records.deck_id != ” 判定と同じ理由)ため、IS NOT NULLではなく != ” で判定する。
const hasDeckForRecordCondition = "(records.deck_id != '' OR records.deck_code_id != '')"

func (i *DesignationStats) CountRecordsByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").
		Where("user_id = ? AND deleted_at IS NULL AND ignore_stats_flg = false", userId).
		Where(existsMatchForRecordCondition).
		Where(hasDeckForRecordCondition)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}

	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

// existsMatchForRecordConditionAsOf は existsMatchForRecordCondition と同様だが、
// 加えて matches.created_at < asOf も要求する。対戦結果(matches)はrecordsとは別の
// テーブルへの追加行であり、対戦結果を追加してもrecordsの行自体(updated_at含む)は
// 更新されない。そのため existsMatchForRecordCondition をそのまま過去の特定時点(asOf)の
// 判定に使うと、「現在は対戦結果が付いているか」しか見ておらず、実際に対戦結果が
// 追加されるより前のasOfでも達成済みと誤判定してしまう(記録だけ先に作成し、対戦結果を
// 後から追加したケース)。matches.created_atも見ることで、asOf時点でまだ対戦結果が
// 追加されていなかった記録を正しく除外する。
const existsMatchForRecordConditionAsOf = "EXISTS (" +
	"SELECT 1 FROM matches WHERE matches.record_id = records.id AND matches.deleted_at IS NULL " +
	"AND matches.created_at < ?" +
	")"

// hasDeckCreatedForRecordConditionAsOf は、records に指定されているデッキ(deck_id)もしくは
// デッキコード(deck_code_id)が asOf 時点で既に作成済みであることを求める条件。
// 「いつデッキが登録されたか」は records.deck_registered_at で判定するが、この値は
// カラム追加時のマイグレーションで既存記録を created_at で埋めた近似値であり、デッキを
// 後から登録した記録では「記録作成時点で既に登録済み」と誤って扱われる。その結果、
// デッキがまだ存在すらしていない過去時点を称号の達成済み時点と誤判定し、称号・ランクの
// 達成日が「初デッキ」バッジの達成日より前になる(=通知の並び順が達成条件と逆転する)。
// デッキ(コード)は記録に登録されるより前に必ず作成されているため、その created_at を
// asOf と比較してデッキ作成前の時点を除外する。
const hasDeckCreatedForRecordConditionAsOf = "(" +
	"EXISTS (SELECT 1 FROM decks WHERE decks.id = records.deck_id AND decks.created_at < ?)" +
	" OR " +
	"EXISTS (SELECT 1 FROM deck_codes WHERE deck_codes.id = records.deck_code_id AND deck_codes.created_at < ?)" +
	")"

func (i *DesignationStats) CountRecordsAsOfByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	asOf time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").
		Where("user_id = ? AND deleted_at IS NULL AND ignore_stats_flg = false", userId).
		Where(existsMatchForRecordConditionAsOf, asOf).
		Where(hasDeckForRecordCondition).
		Where("COALESCE(deck_registered_at, created_at) < ?", asOf).
		Where(hasDeckCreatedForRecordConditionAsOf, asOf, asOf).
		Where("event_date < ?", asOf)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}

	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

func (i *DesignationStats) CountCityLeagueRecordsByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND official_events.type_id = ?",
			userId, cityLeagueTypeId,
		)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

func (i *DesignationStats) CountLeagueRecordsByUserId(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (int, error) {
	var count int64

	query := i.db.Table("records").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND official_events.type_id IN (?, ?)",
			userId, cityLeagueTypeId, trainersLeagueTypeId,
		)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	if tx := query.Count(&count); tx.Error != nil {
		return 0, tx.Error
	}

	return int(count), nil
}

// userRecordCount は user_id ごとの件数集計クエリの結果を受けるための行構造体。
type userRecordCount struct {
	UserId string
	Count  int
}

func scanUserRecordCounts(query *gorm.DB) (map[string]int, error) {
	var results []userRecordCount

	if tx := query.Scan(&results); tx.Error != nil {
		return nil, tx.Error
	}

	counts := make(map[string]int, len(results))
	for _, r := range results {
		counts[r.UserId] = r.Count
	}

	return counts, nil
}

func (i *DesignationStats) CountRecordsGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("records").
		Select("user_id AS user_id, COUNT(*) AS count").
		Where("deleted_at IS NULL AND ignore_stats_flg = false").
		Where(existsMatchForRecordCondition).
		Where(hasDeckForRecordCondition)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}
	query = query.Group("user_id")

	return scanUserRecordCounts(query)
}

func (i *DesignationStats) CountCityLeagueRecordsGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("records").
		Select("records.user_id AS user_id, COUNT(*) AS count").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.deleted_at IS NULL AND records.ignore_stats_flg = false AND official_events.type_id = ?",
			cityLeagueTypeId,
		)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}
	query = query.Group("records.user_id")

	return scanUserRecordCounts(query)
}

// existsRecordWithSameOfficialEventIdCondition は、cityleague_results.official_event_id と
// 同じ official_event_id を持つ userId 自身の records が存在することを求める、内部限定の
// 追加条件(ユーザーへ提示する説明文には含めない)。
const existsRecordWithSameOfficialEventIdCondition = "EXISTS (" +
	"SELECT 1 FROM records WHERE records.official_event_id = cityleague_results.official_event_id " +
	"AND records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false" +
	")"

func (i *DesignationStats) ExistsCityLeagueResultByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	fromDate time.Time,
	toDate time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("cityleague_results").
		Where("player_id = ?", playerId).
		Where(existsRecordWithSameOfficialEventIdCondition, userId)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

// existsRecordWithSameOfficialEventIdConditionAsOf は existsRecordWithSameOfficialEventIdCondition
// と同様だが、加えて records.created_at < asOf も要求する(理由は
// repository.DesignationStatsInterface.ExistsCityLeagueResultAsOfByPlayerId参照)。
const existsRecordWithSameOfficialEventIdConditionAsOf = "EXISTS (" +
	"SELECT 1 FROM records WHERE records.official_event_id = cityleague_results.official_event_id " +
	"AND records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND records.created_at < ?" +
	")"

func (i *DesignationStats) ExistsCityLeagueResultAsOfByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	fromDate time.Time,
	asOf time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("cityleague_results").
		Where("player_id = ?", playerId).
		Where(existsRecordWithSameOfficialEventIdConditionAsOf, userId, asOf).
		Where("event_date < ?", asOf)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *DesignationStats) ExistsCityLeagueResultGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("cityleague_results").
		Select("DISTINCT users_players.user_id AS user_id, 1 AS count").
		Joins(
			"JOIN users_players ON users_players.player_id = cityleague_results.player_id AND users_players.deleted_at IS NULL",
		).
		Joins(
			"JOIN records ON records.official_event_id = cityleague_results.official_event_id " +
				"AND records.user_id = users_players.user_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false",
		)
	if !fromDate.IsZero() {
		query = query.Where("cityleague_results.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("cityleague_results.event_date < ?", toDate)
	}

	return scanUserRecordCounts(query)
}

func (i *DesignationStats) ExistsCityLeagueFinalTournamentResultByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	maxRank int,
	fromDate time.Time,
	toDate time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("cityleague_results").
		Where("player_id = ? AND rank <= ?", playerId, maxRank).
		Where(existsRecordWithSameOfficialEventIdCondition, userId)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *DesignationStats) ExistsCityLeagueFinalTournamentResultAsOfByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	maxRank int,
	fromDate time.Time,
	asOf time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("cityleague_results").
		Where("player_id = ? AND rank <= ?", playerId, maxRank).
		Where(existsRecordWithSameOfficialEventIdConditionAsOf, userId, asOf).
		Where("event_date < ?", asOf)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *DesignationStats) ExistsCityLeagueFinalTournamentResultGroupByUserId(
	ctx context.Context,
	maxRank int,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("cityleague_results").
		Select("DISTINCT users_players.user_id AS user_id, 1 AS count").
		Joins(
			"JOIN users_players ON users_players.player_id = cityleague_results.player_id AND users_players.deleted_at IS NULL",
		).
		Joins(
			"JOIN records ON records.official_event_id = cityleague_results.official_event_id "+
				"AND records.user_id = users_players.user_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false",
		).
		Where("cityleague_results.rank <= ?", maxRank)
	if !fromDate.IsZero() {
		query = query.Where("cityleague_results.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("cityleague_results.event_date < ?", toDate)
	}

	return scanUserRecordCounts(query)
}

func (i *DesignationStats) ExistsCityLeagueResultWithoutMatchingRecordByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	fromDate time.Time,
	toDate time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("cityleague_results").
		Where("player_id = ?", playerId).
		Where("NOT "+existsRecordWithSameOfficialEventIdCondition, userId)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *DesignationStats) ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	maxRank int,
	fromDate time.Time,
	toDate time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("cityleague_results").
		Where("player_id = ? AND rank <= ?", playerId, maxRank).
		Where("NOT "+existsRecordWithSameOfficialEventIdCondition, userId)
	if !fromDate.IsZero() {
		query = query.Where("event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("event_date < ?", toDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *DesignationStats) CountLeagueRecordsGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("records").
		Select("records.user_id AS user_id, COUNT(*) AS count").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.deleted_at IS NULL AND records.ignore_stats_flg = false AND official_events.type_id IN (?, ?)",
			cityLeagueTypeId, trainersLeagueTypeId,
		)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}
	query = query.Group("records.user_id")

	return scanUserRecordCounts(query)
}

// existsCityLeagueResultForRecordCondition は、records.official_event_id と同じ
// official_event_id を持つ、指定プレイヤーIDの入賞結果(cityleague_results)が存在することを
// 求める条件。名人(official_city_league_grandmaster)の「入賞を逃したシティリーグ記録が
// 無いか」を NOT 付きで判定するのに使う。入賞の定義はベテランと同じく cityleague_results への
// 掲載有無で、rank のしきい値は持たない。
const existsCityLeagueResultForRecordCondition = "EXISTS (" +
	"SELECT 1 FROM cityleague_results " +
	"WHERE cityleague_results.official_event_id = records.official_event_id AND cityleague_results.player_id = ?" +
	")"

func (i *DesignationStats) ExistsCityLeagueRecordWithoutPlacementByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	fromDate time.Time,
	toDate time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("records").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND official_events.type_id = ?",
			userId, cityLeagueTypeId,
		).
		Where("NOT "+existsCityLeagueResultForRecordCondition, playerId)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *DesignationStats) ExistsCityLeagueRecordWithoutPlacementAsOfByPlayerId(
	ctx context.Context,
	userId string,
	playerId string,
	fromDate time.Time,
	asOf time.Time,
) (bool, error) {
	var count int64

	query := i.db.Table("records").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND official_events.type_id = ?",
			userId, cityLeagueTypeId,
		).
		Where("records.created_at < ?", asOf).
		Where("records.event_date < ?", asOf).
		Where("NOT "+existsCityLeagueResultForRecordCondition, playerId)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}

	if tx := query.Limit(1).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *DesignationStats) ExistsCityLeagueRecordWithoutPlacementGroupByUserId(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	query := i.db.Table("records").
		Select("DISTINCT records.user_id AS user_id, 1 AS count").
		Joins("JOIN official_events ON official_events.id = records.official_event_id").
		Joins("JOIN users_players ON users_players.user_id = records.user_id AND users_players.deleted_at IS NULL").
		Where(
			"records.deleted_at IS NULL AND records.ignore_stats_flg = false AND official_events.type_id = ?",
			cityLeagueTypeId,
		).
		Where(
			"NOT EXISTS (SELECT 1 FROM cityleague_results " +
				"WHERE cityleague_results.official_event_id = records.official_event_id " +
				"AND cityleague_results.player_id = users_players.player_id)",
		)
	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	return scanUserRecordCounts(query)
}
