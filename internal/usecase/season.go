package usecase

import (
	"strconv"
	"time"
)

// CurrentSeasonLabel は「9月1日〜翌年8月31日」を1シーズンとしたとき、now が属する
// シーズンの識別子(終了年、例: 2026年8月まで続くシーズンなら"2026")を返す。
// UserStat の season 絞り込み(usecase/user_stat.go)と同じシーズン定義に揃えている。
// controller層が season 未指定時のレスポンスに実際の値を埋めるために公開している。
func CurrentSeasonLabel(now time.Time) string {
	year := now.Year()
	if now.Month() >= time.September {
		year++
	}

	return strconv.Itoa(year)
}

// seasonRange は season(シーズン識別子の文字列。空文字なら now が属する現在のシーズン)を
// 「9月1日〜翌年8月31日」の期間に変換する(toDate は翌日0時のexclusive上限)。
func seasonRange(season string, now time.Time) (fromDate time.Time, toDate time.Time, err error) {
	if season == "" {
		season = CurrentSeasonLabel(now)
	}

	year, err := strconv.Atoi(season)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	fromDate = time.Date(year-1, time.September, 1, 0, 0, 0, 0, now.Location())
	toDate = time.Date(year, time.August, 31, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)

	return fromDate, toDate, nil
}

// previousSeasonRange は season(空文字なら現在のシーズン)のひとつ前のシーズンの期間を返す。
// 「前シーズンに引き続き」といった、シーズンをまたいだ継続条件の判定に使う。
func previousSeasonRange(season string, now time.Time) (fromDate time.Time, toDate time.Time, err error) {
	if season == "" {
		season = CurrentSeasonLabel(now)
	}

	year, err := strconv.Atoi(season)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return seasonRange(strconv.Itoa(year-1), now)
}
