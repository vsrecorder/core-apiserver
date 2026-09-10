package infrastructure

import "gorm.io/gorm"

// applyExcludeDefaultMatches は不戦勝/不戦敗を集計から外す条件を query に適用する。
//
// 不戦勝/不戦敗(default_victory_flg / default_defeat_flg)は対戦そのものが行われて
// いないため、勝率を「実力」として読みたい利用者にとっては雑音になる。一方で公式の
// スイスドロー成績とは一致しなくなるので、常に外すのではなく利用者の選択に委ねている。
//
// 除外は対戦数・勝ち数・負け数のすべてから外す(勝率の分母だけを外すのではない)。
// 引き分け(draw_flg)が「対戦はしたが決着がつかなかった」ために総対戦数には数えるのと違い、
// 不戦勝/不戦敗は対戦自体が存在しないため、1戦として数える根拠がないからである。
// みんなのデッキ環境(weekly_deck_usage_stat)が常時この条件で外しているのと同じ考え方。
//
// matches を主テーブル、または matches という別名で結合しているクエリで使うこと。
func applyExcludeDefaultMatches(query *gorm.DB, excludeDefaultMatches bool) *gorm.DB {
	if !excludeDefaultMatches {
		return query
	}

	return query.Where("matches.default_victory_flg = false AND matches.default_defeat_flg = false")
}
