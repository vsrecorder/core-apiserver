package repository

import (
	"context"
	"time"
)

// DesignationStatsInterface は称号判定に必要な集計値を返す。
// records/official_events テーブルを直接集計する点は BadgeStatsInterface と同様の設計。
// 称号は現在のシーズン内の集計のみで判定するため、fromDate/toDate には常にシーズンの
// 期間を渡す(BadgeStatsInterface と異なり、両方ゼロ値=無期限で呼び出す使い方はしない)。
type DesignationStatsInterface interface {
	// CountGymBattleRecordsByUserId は、公式イベントのうちジムバトル
	// (official_events.type_id = 4 かつ title に「ジムバトル」を含む。webapp側の
	// officialEventHelpers.ts の判定ロジックに合わせている)に紐づく記録の件数を返す。
	CountGymBattleRecordsByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	// CountLeagueRecordsByUserId は、公式イベントのうちトレーナーズリーグまたは
	// シティリーグに紐づく記録の件数を返す。
	CountLeagueRecordsByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	// CountCityLeagueRecordsByUserId は、公式イベントのうちシティリーグのみに
	// 紐づく記録の件数を返す(トレーナーズリーグは含まない)。
	CountCityLeagueRecordsByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	// CountGymBattleRecordsGroupByUserId は CountGymBattleRecordsByUserId のユーザー横断版。
	// 指定期間内にジムバトル記録を1件以上持つユーザーごとの件数を user_id をキーに返す
	// (該当記録が0件のユーザーはキーに含まれない)。
	CountGymBattleRecordsGroupByUserId(
		ctx context.Context,
		fromDate time.Time,
		toDate time.Time,
	) (map[string]int, error)

	// CountLeagueRecordsGroupByUserId は CountLeagueRecordsByUserId のユーザー横断版。
	CountLeagueRecordsGroupByUserId(
		ctx context.Context,
		fromDate time.Time,
		toDate time.Time,
	) (map[string]int, error)

	// CountCityLeagueRecordsGroupByUserId は CountCityLeagueRecordsByUserId のユーザー横断版。
	CountCityLeagueRecordsGroupByUserId(
		ctx context.Context,
		fromDate time.Time,
		toDate time.Time,
	) (map[string]int, error)
}
