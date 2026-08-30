package repository

import (
	"context"
)

// PokemonSpriteNameInterface は pokemon_sprites マスタ(スプライトID → 正式名)の参照。
// 通知本文のように「スプライトの組み合わせを人が読める名前にする」用途で使う。
type PokemonSpriteNameInterface interface {
	// FindNamesByIds は指定IDの正式名を id → name の対応で返す。存在しないIDは含まれない。
	// ids が空なら空の対応を返す(全件取得にはならない)。
	FindNamesByIds(
		ctx context.Context,
		ids []string,
	) (map[string]string, error)
}
