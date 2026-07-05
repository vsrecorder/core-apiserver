package infrastructure

import (
	"sort"
	"strings"
)

// NormalizeFingerprint は position 順のスプライトID列を正規化し、集計キー・代表スプライト・
// 表示用スプライト列に変換する（DATA_STRATEGY.md B1「スプライト指紋の正規化」）。
//
// 正規化ルール:
//   - 先頭(Position1)＝代表ポケモン(大分類)としてアンカーに残す（2階層モデル 3-4節）
//   - 残りは重複排除してソートし、並び順・重複のゆらぎを吸収する
//   - フリーテキスト(OpponentsDeckInfo)はキーに含めない（補助ラベルは呼び出し側で扱う）
//
// これにより「同一デッキが並び順や重複、テキスト表記の違いで別グループに割れる」現象を解消する。
// スプライト未付与（空スライス）の場合は集計不能として空文字を返し、呼び出し側で除外する。
//
// 戻り値:
//   - key: 集計キー（primary と正規化済みの残りを "|" で連結した文字列）
//   - primary: 先頭スプライトID（大分類。空指紋なら ""）
//   - normalized: 表示用スプライト列（primary を先頭に、残りを重複排除・ソートしたもの）
func NormalizeFingerprint(spriteIds []string) (key string, primary string, normalized []string) {
	if len(spriteIds) == 0 {
		return "", "", nil
	}

	primary = spriteIds[0]

	// 先頭以外を重複排除する。primary と同一のものも冗長なので除く。
	seen := map[string]struct{}{primary: {}}
	rest := make([]string, 0, len(spriteIds)-1)
	for _, id := range spriteIds[1:] {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		rest = append(rest, id)
	}

	sort.Strings(rest)

	normalized = make([]string, 0, len(rest)+1)
	normalized = append(normalized, primary)
	normalized = append(normalized, rest...)

	key = primary + "|" + strings.Join(rest, ",")

	return key, primary, normalized
}
