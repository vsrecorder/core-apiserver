package infrastructure

import (
	"sort"
	"strings"
)

// NormalizeFingerprint はスプライトID列を正規化し、集計キーと表示用スプライト列に変換する
// （DATA_STRATEGY.md B1「スプライト指紋の正規化」）。
//
// 正規化ルール:
//   - 重複を排除する
//   - 集計キーは並び順を考慮しない（全体をソートし、入力順序の違いによる分裂を防ぐ）
//   - 一方、表示用スプライト列はソートせず、元データの並び順をできるだけ保つ
//     （重複排除のみ行う。先頭一致で最初に見つかった順序をそのまま使う）
//
// フリーテキスト（デッキ名等）は受け取らず、スプライトの集合のみで指紋を決定する。
// スプライト未付与（空スライス）の場合は集計不能として空文字を返し、呼び出し側で除外する。
//
// 戻り値:
//   - key: 集計キー（重複排除・ソート済みのスプライトIDをカンマ区切りで連結した文字列）
//   - ordered: 表示用スプライト列（重複排除のみ。並び順は元データのまま）
func NormalizeFingerprint(spriteIds []string) (key string, ordered []string) {
	if len(spriteIds) == 0 {
		return "", nil
	}

	seen := make(map[string]struct{}, len(spriteIds))
	ordered = make([]string, 0, len(spriteIds))
	for _, id := range spriteIds {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}

	sortedForKey := make([]string, len(ordered))
	copy(sortedForKey, ordered)
	sort.Strings(sortedForKey)

	return strings.Join(sortedForKey, ","), ordered
}
