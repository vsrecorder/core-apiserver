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
//   - 並び順は考慮しない（全体をソートし、入力順序の違いによる分裂を防ぐ）
//
// フリーテキスト（デッキ名等）は受け取らず、スプライトの集合のみで指紋を決定する。
// スプライト未付与（空スライス）の場合は集計不能として空文字を返し、呼び出し側で除外する。
//
// 戻り値:
//   - key: 集計キー（正規化済みスプライトIDをカンマ区切りで連結した文字列）
//   - normalized: 表示用スプライト列（重複排除・ソート済み）
func NormalizeFingerprint(spriteIds []string) (key string, normalized []string) {
	if len(spriteIds) == 0 {
		return "", nil
	}

	seen := make(map[string]struct{}, len(spriteIds))
	unique := make([]string, 0, len(spriteIds))
	for _, id := range spriteIds {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	sort.Strings(unique)

	return strings.Join(unique, ","), unique
}
