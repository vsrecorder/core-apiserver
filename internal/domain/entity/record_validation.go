package entity

import "time"

// RecordEventSource は記録が紐づくイベント種別の指定状況。
// 4種のうちちょうど1つだけ指定されている必要がある。
type RecordEventSource struct {
	OfficialEventId   uint
	TonamelEventId    string
	FriendId          string
	UnofficialEventId string
}

// IsValidRecordEventSource は記録の紐づくイベントが、
// 公式イベント / Tonamel / フレンド対戦 / 自由形式 の4種のうち
// ちょうど1つだけ指定されているかを検証する。
// 2つ以上でも、1つも無くても不正(false)。
//
// controller(middleware)と usecase の両方から呼び、検証の二重実装による
// 仕様の乖離を防ぐための単一の真実。
func IsValidRecordEventSource(src RecordEventSource) bool {
	count := 0
	if src.OfficialEventId != 0 {
		count++
	}
	if src.TonamelEventId != "" {
		count++
	}
	if src.FriendId != "" {
		count++
	}
	if src.UnofficialEventId != "" {
		count++
	}

	return count == 1
}

// IsValidRecordEventDate は記録の対戦日が入力されているかを検証する。
// 未入力(ゼロ値)を許すと、週次ストリークやシーズン集計の基準日が created_at へ
// フォールバックし(usecase.RecordBasisTime)、ユーザーが指定していない日付で
// 集計されてしまうため必須にする。
func IsValidRecordEventDate(eventDate time.Time) bool {
	return !eventDate.IsZero()
}
