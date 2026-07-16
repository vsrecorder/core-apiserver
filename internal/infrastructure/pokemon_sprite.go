package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// matchIdsOf は matches と games を JOIN した結果から、重複を除いた match_id を取り出す。
// 1つの対戦結果が対局の数だけ行に現れるため、そのままではIDが重複する。
func matchIdsOf(results []*model.MatchJoinGame) []string {
	seen := make(map[string]struct{}, len(results))
	ids := make([]string, 0, len(results))

	for _, result := range results {
		if _, ok := seen[result.MatchID]; ok {
			continue
		}
		seen[result.MatchID] = struct{}{}
		ids = append(ids, result.MatchID)
	}

	return ids
}

// findMatchPokemonSpritesByMatchIds は複数の対戦結果のスプライトを1クエリでまとめて取得し、
// match_id ごとに束ねて返す。
//
// 対戦結果を一覧で返す処理でスプライトを1件ずつ引くと、件数に比例してクエリが増える(N+1)。
// 一覧を扱う箇所では必ずこちらを使う。
//
// position の昇順で並べる。テーブルの主キーが (match_id, position) のため未指定でも
// 結果的に並ぶことが多いが、IN でまとめて引くと順序は保証されないため明示する。
func findMatchPokemonSpritesByMatchIds(
	ctx context.Context,
	db *gorm.DB,
	matchIds []string,
) (map[string][]*entity.PokemonSprite, error) {
	if len(matchIds) == 0 {
		return map[string][]*entity.PokemonSprite{}, nil
	}

	var spriteModels []*model.MatchPokemonSprite
	if tx := db.WithContext(ctx).
		Where("match_id IN ?", matchIds).
		Order("position ASC").
		Find(&spriteModels); tx.Error != nil {
		return nil, tx.Error
	}

	spritesByMatchId := make(map[string][]*entity.PokemonSprite, len(matchIds))
	for _, m := range spriteModels {
		spritesByMatchId[m.MatchId] = append(
			spritesByMatchId[m.MatchId],
			entity.NewPokemonSprite(m.PokemonSpriteId),
		)
	}

	return spritesByMatchId, nil
}

// deckIdsOf は decks と deck_codes を JOIN した結果から、重複を除いた deck_id を取り出す。
func deckIdsOf(results []*model.DeckJoinDeckCode) []string {
	seen := make(map[string]struct{}, len(results))
	ids := make([]string, 0, len(results))

	for _, result := range results {
		if _, ok := seen[result.DeckID]; ok {
			continue
		}
		seen[result.DeckID] = struct{}{}
		ids = append(ids, result.DeckID)
	}

	return ids
}

// findDeckPokemonSpritesByDeckIds は複数のデッキのスプライトを1クエリでまとめて取得し、
// deck_id ごとに束ねて返す。意図は findMatchPokemonSpritesByMatchIds と同じ。
func findDeckPokemonSpritesByDeckIds(
	ctx context.Context,
	db *gorm.DB,
	deckIds []string,
) (map[string][]*entity.PokemonSprite, error) {
	if len(deckIds) == 0 {
		return map[string][]*entity.PokemonSprite{}, nil
	}

	var spriteModels []*model.DeckPokemonSprite
	if tx := db.WithContext(ctx).
		Where("deck_id IN ?", deckIds).
		Order("position ASC").
		Find(&spriteModels); tx.Error != nil {
		return nil, tx.Error
	}

	spritesByDeckId := make(map[string][]*entity.PokemonSprite, len(deckIds))
	for _, m := range spriteModels {
		spritesByDeckId[m.DeckId] = append(
			spritesByDeckId[m.DeckId],
			entity.NewPokemonSprite(m.PokemonSpriteId),
		)
	}

	return spritesByDeckId, nil
}
