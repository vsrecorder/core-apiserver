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
	// CountRecordsByUserId は、公式イベント・Tonamelイベント・記入形式(unofficial_event)
	// のいずれであるかを問わず、ユーザーが作成した全ての記録の件数を返す。
	CountRecordsByUserId(
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

	// CountRecordsGroupByUserId は CountRecordsByUserId のユーザー横断版。
	// 指定期間内に記録を1件以上持つユーザーごとの件数を user_id をキーに返す
	// (該当記録が0件のユーザーはキーに含まれない)。
	CountRecordsGroupByUserId(
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

	// ExistsCityLeagueResultByPlayerId は、公式サイトの結果(cityleague_results)に
	// 指定プレイヤーIDのレコードが、指定期間内に1件以上存在するかを返す。
	// records/official_events ではなく、公式サイトからスクレイピングした
	// cityleague_results を直接参照する点が他メソッドと異なる。
	// 加えて、その cityleague_results.official_event_id と同じ official_event_id を持つ
	// records(userId自身が作成した記録)が存在することも内部的な条件とする
	// (公式サイト側の結果だけでバトレコ側の記録が無い状態での到達を防ぐための条件であり、
	// ユーザーへ提示する説明文や表示には含めない)。
	ExistsCityLeagueResultByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		fromDate time.Time,
		toDate time.Time,
	) (bool, error)

	// ExistsCityLeagueResultGroupByUserId は ExistsCityLeagueResultByPlayerId のユーザー横断版。
	// users_players(プレイヤーズクラブ連携。deleted_at IS NULL のもののみ)を介して
	// cityleague_results.player_id をバトレコの user_id に変換した上で、指定期間内に
	// 1件以上該当レコードがあるユーザーを user_id をキーに返す(該当なしのユーザーは
	// キーに含まれない。値は常に1)。ExistsCityLeagueResultByPlayerId と同様、同じ
	// official_event_id を持つ本人の records が存在するユーザーのみを対象とする。
	ExistsCityLeagueResultGroupByUserId(
		ctx context.Context,
		fromDate time.Time,
		toDate time.Time,
	) (map[string]int, error)

	// ExistsCityLeagueFinalTournamentResultByPlayerId は ExistsCityLeagueResultByPlayerId と
	// 同様だが、cityleague_results.rank が maxRank 以下のレコード(決勝トーナメント進出と
	// みなす)に限定して存在有無を返す。しきい値の意味(usecase.DesignationCityLeagueFinal
	// TournamentMaxRank)は usecase 層が持ち、ここでは受け取った値でそのまま絞り込む。
	// ExistsCityLeagueResultByPlayerId と同様、同じ official_event_id を持つ userId 自身の
	// records が存在することも内部的な条件とする。
	ExistsCityLeagueFinalTournamentResultByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		maxRank int,
		fromDate time.Time,
		toDate time.Time,
	) (bool, error)

	// ExistsCityLeagueFinalTournamentResultGroupByUserId は
	// ExistsCityLeagueFinalTournamentResultByPlayerId のユーザー横断版。
	ExistsCityLeagueFinalTournamentResultGroupByUserId(
		ctx context.Context,
		maxRank int,
		fromDate time.Time,
		toDate time.Time,
	) (map[string]int, error)

	// ExistsCityLeagueResultWithoutMatchingRecordByPlayerId は、cityleague_results に
	// 指定プレイヤーIDのレコードが指定期間内に存在するにもかかわらず、その
	// official_event_id と一致する userId 自身の records が無い状態(=
	// ExistsCityLeagueResultByPlayerId が false になる原因が記録未登録であること)を
	// 検出する。称号詳細モーダルで「対象の大会の記録が見つからない」という案内を
	// 出し分けるためのヒント用途であり、達成条件そのものの判定には使わない。
	ExistsCityLeagueResultWithoutMatchingRecordByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		fromDate time.Time,
		toDate time.Time,
	) (bool, error)

	// ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId は
	// ExistsCityLeagueResultWithoutMatchingRecordByPlayerId と同様だが、熟練判定用に
	// rank が maxRank 以下のレコードに限定する。
	ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		maxRank int,
		fromDate time.Time,
		toDate time.Time,
	) (bool, error)
}
