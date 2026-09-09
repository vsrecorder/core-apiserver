package repository

import "time"

// StatPeriod はユーザー統計(戦績サマリー・デッキ使用率・相手デッキ使用率)の期間の
// 絞り込み条件。records.event_date に対して適用する。
//
// 期間だけでなく例外の公式イベントを持つのは、環境が「期間」では表しきれないため。
// 大型大会(チャンピオンズリーグ・PJCS)は開催日時点で発売済みの最新弾がカードプールに
// 入らないことがあり、開催日が次の環境の期間に入っていても前の環境として集計する必要が
// ある(official_event_environments。usecase.ResolveEnvironmentForOfficialEvent と同じ考え方)。
//
// 期間だけを指定する(環境で絞らない)場合は Include/Exclude が空になり、従来どおり
// event_date の範囲だけで絞られる。
type StatPeriod struct {
	// From / To は event_date の半開区間 [From, To)。両方ゼロ値なら期間で絞らない(全期間)。
	From time.Time
	To   time.Time

	// BaseFrom / BaseTo は環境以外の条件(week / year_month / season / standard_regulation)
	// だけで決まる期間。IncludeOfficialEventIds を拾う範囲に使う。
	//
	// 環境と他の条件は交差を取る仕様なので、期間外から拾い直す例外イベントも他の条件は
	// 満たしている必要がある(例: year_month=2026-10 と environment=m6 を同時に指定したとき、
	// 9月開催のチャンピオンズリーグを拾ってはいけない)。
	BaseFrom time.Time
	BaseTo   time.Time

	// IncludeOfficialEventIds は開催日が From〜To の外でも、この環境の記録として集計する
	// 公式イベントのID。
	IncludeOfficialEventIds []uint

	// ExcludeOfficialEventIds は開催日が From〜To の中でも、別の環境の記録として扱うため
	// 集計から外す公式イベントのID。
	ExcludeOfficialEventIds []uint
}
