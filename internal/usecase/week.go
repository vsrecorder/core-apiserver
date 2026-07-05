package usecase

import "time"

// weekDateLayout は week パラメータ（週の開始日=月曜）の形式。
const weekDateLayout = "2006-01-02"

// weekRange は week（週内の任意日を表す "YYYY-MM-DD"。空文字なら now が属する週）を、
// その週の月曜0時から翌週月曜0時（exclusive上限）までの期間に変換する。
// 週は月曜始まりとする。時刻計算は now のロケーション（Asia/Tokyo）で行う。
func weekRange(week string, now time.Time) (fromDate time.Time, toDate time.Time, err error) {
	base := now

	if week != "" {
		t, perr := time.ParseInLocation(weekDateLayout, week, now.Location())
		if perr != nil {
			return time.Time{}, time.Time{}, perr
		}
		base = t
	}

	// 月曜からの経過日数（月曜=0 ... 日曜=6）を求める。
	// time.Weekday は日曜=0 ... 土曜=6 なので (weekday+6)%7 で月曜始まりへ変換する。
	offset := (int(base.Weekday()) + 6) % 7

	monday := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).AddDate(0, 0, -offset)

	return monday, monday.AddDate(0, 0, 7), nil
}
