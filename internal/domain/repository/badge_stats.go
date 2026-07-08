package repository

import (
	"context"
	"time"
)

// BadgeStatsInterface はバッジ判定に必要な集計値を返す。
// records/matches/decks テーブルを直接集計する点は UserStatInterface と同様の設計。
//
// Count系メソッドは fromDate/toDate で期間を絞り込める。両方ゼロ値の場合は無期限(全期間)
// を意味する(オンボーディング系バッジの判定用)。シーズン等の期間を渡すとその期間内のみを
// 集計する(マイルストーン系バッジ・称号の判定用)。
type BadgeStatsInterface interface {
	CountRecordsByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	CountMatchesByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	CountDecksByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	// CountDeckCodesByUserId は、デッキコードが登録済みのデッキの件数を返す。
	// マイルストーン系バッジ(deck_code_count、「デッキコード数」)の判定用で、
	// CountDecksByUserId(デッキ登録数そのもの。オンボーディング系「初デッキ」判定用)とは
	// 別物であり、デッキコード未登録のデッキはカウントに含めない。
	CountDeckCodesByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) (int, error)

	// FindRecordDatesByUserId は指定期間内の記録の日付(event_dateが無ければcreated_at)を
	// 重複を許して返す。週次ストリークバッジの期間内集計のため、週への丸め込みや連続数の
	// 計算は呼び出し側(usecase層)で行う。
	FindRecordDatesByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) ([]time.Time, error)

	// FindDeckDatesByUserId は指定期間内のデッキ登録日時(created_at)を昇順で返す。
	// マイルストーン系バッジ(deck_count)の「シーズン内で何番目のデッキ登録が閾値に
	// 到達したか」を求めるために使う。
	FindDeckDatesByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) ([]time.Time, error)

	// FindMatchDatesByUserId は指定期間内の対戦の作成日時(created_at)を昇順で返す。
	// 期間の絞り込みは紐づく record の event_date で行う(CountMatchesByUserId と同じ基準)。
	// マイルストーン系バッジ(match_count)の「シーズン内で何番目の対戦が閾値に到達したか」
	// を求めるために使う。
	FindMatchDatesByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) ([]time.Time, error)

	// FindDeckCodeDatesByUserId は指定期間内のデッキコード登録日時(deck_codes.created_at)を
	// 昇順で返す。マイルストーン系バッジ(deck_code_count)の「シーズン内で何番目の
	// デッキコード登録が閾値に到達したか」を求めるために使う。
	FindDeckCodeDatesByUserId(
		ctx context.Context,
		userId string,
		fromDate time.Time,
		toDate time.Time,
	) ([]time.Time, error)
}
