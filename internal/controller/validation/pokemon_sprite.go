package validation

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

// MaxPokemonSpritesPerEntity はデッキ・対戦結果に付けられるスプライトの数。
// 表示枠は position 1 / 2 の2つ(deck_pokemon_sprites / match_pokemon_sprites の主キーは position)。
const MaxPokemonSpritesPerEntity = 2

/*
 * validatePokemonSprites はデッキ・対戦結果のスプライト指定を検証する。
 *
 * 保存側は position を主キーに持ち(重複で主キー違反)、pokemon_sprite_id は外部キー
 * (形式外の値でも列幅超過や参照先無しで DB エラー)になる。いずれもクライアント起因の
 * 入力なので、DB エラーの 500 ではなく入口で 400 にする。
 *
 * position は省略(0)できる。省略時は保存側が配列の並び順(1, 2)で採番するため、
 * 「全て省略」か「全て指定(重複なし)」のどちらかに限る。混在すると採番した値と指定した値が
 * 衝突しうる。
 */
func validatePokemonSprites(sprites []*dto.PokemonSpriteRequest) bool {
	if len(sprites) > MaxPokemonSpritesPerEntity {
		return false
	}

	specified := 0
	seen := make(map[uint]struct{}, len(sprites))
	for _, sprite := range sprites {
		if sprite == nil {
			return false
		}

		if !pokemonSpriteIdPattern.MatchString(sprite.ID) {
			return false
		}

		if sprite.Position > MaxPokemonSpritesPerEntity {
			return false
		}

		if sprite.Position == 0 {
			continue
		}

		if _, ok := seen[sprite.Position]; ok {
			return false
		}
		seen[sprite.Position] = struct{}{}
		specified++
	}

	// 指定と省略の混在は不可
	return specified == 0 || specified == len(sprites)
}
