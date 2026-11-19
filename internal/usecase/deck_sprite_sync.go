package usecase

import (
	"context"
	"errors"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// デッキのスプライトと、最新バージョンのスプライトを揃えるための処理。
//
// デッキのスプライトは「いまのデッキの顔」で、デッキ一覧や使用率などの集計に使う。バージョンにも
// スプライトを設定できるようになったので、放っておくと「最新バージョンで相棒を入れ替えたのに
// デッキ一覧は前の相棒のまま」という食い違いが起きる。そこで書き込みのたびに双方向で揃える:
//
//   - 最新バージョンのスプライトを決めた(作成・編集・最新の削除で繰り上がった) → デッキへ写す
//   - デッキのスプライトを変えた → 最新バージョンに設定があれば、それも同じにする
//
// バージョンが未設定(行なし)のときは表示側がデッキのスプライトを出すので、最新バージョンが
// 未設定のあいだは何もしなくても同じになる。揃える必要があるのは設定があるときだけ。

// normalizedSpritePosition は保存時と同じ規約で枠を決める(未指定の 0 は配列順)。
func normalizedSpritePosition(sprite *entity.PokemonSprite, index int) uint {
	if sprite.Position == 0 {
		return uint(index + 1)
	}

	return sprite.Position
}

// samePokemonSprites は2つのスプライト設定が同じ(同じ枠に同じスプライト)かを返す。
func samePokemonSprites(a, b []*entity.PokemonSprite) bool {
	if len(a) != len(b) {
		return false
	}

	byPosition := make(map[uint]string, len(a))
	for index, sprite := range a {
		byPosition[normalizedSpritePosition(sprite, index)] = sprite.ID
	}

	for index, sprite := range b {
		if id, ok := byPosition[normalizedSpritePosition(sprite, index)]; !ok || id != sprite.ID {
			return false
		}
	}

	return true
}

// copyPokemonSprites は別の持ち主へ渡すための複製を返す(エンティティを共有しない)。戻り値は常に非 nil。
func copyPokemonSprites(sprites []*entity.PokemonSprite) []*entity.PokemonSprite {
	copied := make([]*entity.PokemonSprite, 0, len(sprites))
	for index, sprite := range sprites {
		copied = append(copied, entity.NewPokemonSpriteWithPosition(sprite.ID, normalizedSpritePosition(sprite, index)))
	}

	return copied
}

// syncDeckSpritesWithLatestVersion は、最新バージョンのスプライトが sprites に決まったときに、
// デッキのスプライトを同じものへ置き換える。トランザクションの中から呼ぶこと。
//
// sprites が空(未設定)のときは何もしない。未設定の最新バージョンはデッキのスプライトで
// 表示されるので、そのままで同じになる。
//
// otherVersions は同じデッキのほかのバージョン(最新バージョン自身と、削除するものは含めない)。
// このうち未設定のものには、置き換える前のデッキのスプライトを書き込んでから置き換える。
// 未設定のバージョンはデッキのスプライトで表示されるため、何もしないと、最新バージョンで
// 相棒を入れ替えただけで過去のバージョン(とそれを使った記録)の表示まで新しい相棒に変わり、
// バージョンごとに見分けるという目的が崩れる。デッキにスプライトが無かったときは書き込めない
// (空は未設定と同じ)ので、そのバージョンは新しいデッキのスプライトで表示される。
func syncDeckSpritesWithLatestVersion(
	ctx context.Context,
	deckRepository repository.DeckInterface,
	deckCodeRepository repository.DeckCodeInterface,
	deckId string,
	otherVersions []*entity.DeckCode,
	sprites []*entity.PokemonSprite,
) error {
	if len(sprites) == 0 {
		return nil
	}

	deck, err := deckRepository.FindById(ctx, deckId)
	if errors.Is(err, apperror.ErrRecordNotFound) {
		// デッキが見つからない(削除済みなど)なら揃える相手がいない
		return nil
	} else if err != nil {
		return err
	}

	if samePokemonSprites(deck.PokemonSprites, sprites) {
		return nil
	}

	if len(deck.PokemonSprites) > 0 {
		for _, version := range otherVersions {
			if len(version.PokemonSprites) > 0 {
				continue
			}

			if err := deckCodeRepository.ReplacePokemonSprites(ctx, version.ID, copyPokemonSprites(deck.PokemonSprites)); err != nil {
				return err
			}
		}
	}

	return deckRepository.ReplacePokemonSprites(ctx, deckId, copyPokemonSprites(sprites))
}
