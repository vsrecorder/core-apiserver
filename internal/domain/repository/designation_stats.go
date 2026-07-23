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
	// のいずれであるかを問わず、対戦結果(matches)が1件以上紐づいている記録の件数を返す
	// (対戦結果が未追加の記録はカウントしない)。
	CountRecordsByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	// CountRecordsAsOfByUserId は CountRecordsByUserId と同様の条件に加え、
	// records.deck_registered_at < asOf、および対戦結果(matches)についても
	// matches.created_at < asOf も要求する。CountRecordsByUserId は「現在の状態」
	// (現在デッキが登録されているか、現在対戦結果が1件以上紐づいているか)しか見ておらず、
	// それぞれいつ登録・追加されたかを記録していないため、そのまま過去の特定時点(asOf)の
	// 判定に使うと誤判定してしまう:
	//   - デッキ未登録のまま作成した記録に後からデッキを登録(使用したデッキとして編集)
	//     したケース → records.deck_registered_at(deck_id/deck_code_idが未設定から
	//     設定ありに変わった瞬間だけ進む。usecase.Record.Create/Updateが設定する)を
	//     下限として使う。records.updated_atは記録全体のあらゆる編集(メモ修正等の
	//     デッキ登録と無関係な変更)でも進んでしまうため使わない。
	//   - 対戦結果を追加する前に記録だけ先に作成したケース → 対戦結果はrecordsとは別
	//     テーブルへの追加行で、追加してもrecordsの行(updated_at含む)は更新されない
	//     ため、matches.created_atを直接下限として使う。
	//   - deck_registered_at がカラム追加時のマイグレーションで created_at 埋めされた
	//     既存記録 → 「記録作成時点で既にデッキが登録済み」という近似値になっており、
	//     デッキがまだ作成されてすらいない時点を達成済みと誤判定してしまう。紐づく
	//     デッキ(deck_id)・デッキコード(deck_code_id)の created_at < asOf も併せて
	//     要求し、デッキ作成前の時点を除外する。
	// これらにより、asOf時点でまだデッキ登録/対戦結果追加がされていなかった記録を正しく
	// 除外する。usecase.DesignationEvaluation.TierAsOf(backfill-notificationsの「実際に
	// 達成した日」再構築専用)からのみ呼ばれる。
	CountRecordsAsOfByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		asOf time.Time,
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
	// 指定期間内に対戦結果(matches)付きの記録を1件以上持つユーザーごとの件数を
	// user_id をキーに返す(該当記録が0件のユーザーはキーに含まれない)。
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

	// ExistsCityLeagueResultAsOfByPlayerId は ExistsCityLeagueResultByPlayerId と同様の条件に
	// 加え、一致するuserId自身のrecordについて records.created_at < asOf も要求する。
	// ExistsCityLeagueResultByPlayerId は「現在userIdの記録が存在するか」しか見ておらず、
	// その記録がいつ作成されたかを見ていないため、そのまま過去の特定時点(asOf)の判定に
	// 使うと、実際にその記録が作成されるより前のasOfでも達成済みとして扱ってしまう
	// (公式サイトの結果は先に存在していて、バトレコ側の記録を後から作成したケース)。
	// usecase.DesignationEvaluation.TierAsOf(backfill-notificationsの「実際に達成した日」
	// 再構築専用)からのみ呼ばれる。
	ExistsCityLeagueResultAsOfByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		fromDate time.Time,
		asOf time.Time,
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

	// ExistsCityLeagueFinalTournamentResultAsOfByPlayerId は
	// ExistsCityLeagueFinalTournamentResultByPlayerId と同様だが、
	// ExistsCityLeagueResultAsOfByPlayerId と同じ理由で records.created_at < asOf も要求する。
	// usecase.DesignationEvaluation.TierAsOf からのみ呼ばれる。
	ExistsCityLeagueFinalTournamentResultAsOfByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		maxRank int,
		fromDate time.Time,
		asOf time.Time,
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

	// ExistsCityLeagueRecordWithoutPlacementByPlayerId は、userId 自身のシティリーグ記録
	// (records、指定期間内)のうち、その official_event_id に対応する playerId の入賞結果
	// (cityleague_results)が存在しないものが1件以上あるかを返す。名人
	// (official_city_league_grandmaster)の「常に入賞以上」条件の判定に使う:
	// このメソッドが false(= 入賞を逃したシティリーグ記録が1件も無い)であることが、
	// 名人の到達条件の一部となる(もう一方は達人=優勝の達成)。
	// 「入賞」の定義はベテラン(ExistsCityLeagueResultByPlayerId)と同じく cityleague_results
	// にそのプレイヤーIDの結果が存在することで、rank のしきい値は持たない。
	// なお公式サイトに結果が公開されていない(または未取り込みの)大会の記録は、入賞結果が
	// 見つからないため「入賞を逃した」扱いになる点に注意(cityleague_results を正とする既存の
	// 判定方針に従う)。
	ExistsCityLeagueRecordWithoutPlacementByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		fromDate time.Time,
		toDate time.Time,
	) (bool, error)

	// ExistsCityLeagueRecordWithoutPlacementAsOfByPlayerId は
	// ExistsCityLeagueRecordWithoutPlacementByPlayerId と同様だが、records.created_at < asOf
	// も要求し、asOf 時点でまだ作成されていなかった記録を除外する
	// (ExistsCityLeagueResultAsOfByPlayerId と同じ理由。TierAsOf/backfill-notifications 専用)。
	ExistsCityLeagueRecordWithoutPlacementAsOfByPlayerId(
		ctx context.Context,
		userId string,
		playerId string,
		fromDate time.Time,
		asOf time.Time,
	) (bool, error)

	// ExistsCityLeagueRecordWithoutPlacementGroupByUserId は
	// ExistsCityLeagueRecordWithoutPlacementByPlayerId のユーザー横断版。
	// users_players(プレイヤーズクラブ連携。deleted_at IS NULL のもののみ)を介して
	// 各ユーザーの連携プレイヤーIDを解決し、指定期間内に「入賞を逃したシティリーグ記録」を
	// 1件以上持つユーザーを user_id をキーに返す(該当なしのユーザーはキーに含まれない。値は常に1)。
	// GetRankStats で名人の「常に入賞以上」条件(このマップに含まれない=常に入賞)を判定するために使う。
	ExistsCityLeagueRecordWithoutPlacementGroupByUserId(
		ctx context.Context,
		fromDate time.Time,
		toDate time.Time,
	) (map[string]int, error)
}
