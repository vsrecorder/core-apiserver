package repository

import (
	"context"
)

// OfficialEventEnvironmentInterface は公式イベントの環境の例外(official_event_environments)を引く。
//
// 環境は通常イベントの開催日から引くが、大型大会(チャンピオンズリーグ・PJCS)は開催日
// 時点で発売済みの最新弾がカードプールに入らないことがあり、開催日から引いた環境と
// 実際に対戦する環境がズレる。そのズレを表すのがこのテーブル。
type OfficialEventEnvironmentInterface interface {
	// FindEnvironmentIdByOfficialEventId は公式イベントに例外登録があればその環境IDを返す。
	// 登録が無い場合は apperror.ErrRecordNotFound を返す(例外が無いのは通常の状態なので、
	// 呼び出し側はこれを見て開催日からの判定にフォールバックする)。
	FindEnvironmentIdByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
	) (string, error)

	// FindAll は例外登録されている公式イベントIDと環境IDの対応を全件返す。
	//
	// 全件を返すのは、このテーブルが「ズレるイベントだけ」を持つ極小のマスタ(年に数件)
	// だから。環境ごとの絞り込みや、対戦の集合に対する引き当ては、この1回の問い合わせで
	// まかなえる。
	FindAll(
		ctx context.Context,
	) (map[uint]string, error)
}
