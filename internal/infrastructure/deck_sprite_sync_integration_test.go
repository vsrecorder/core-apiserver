package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

// デッキのスプライトと最新バージョンのスプライトが、どの書き込みのあとも揃っていることを
// 実DB・本物のトランザクションで通しで検証する(usecase/deck_sprite_sync.go)。
//
// usecase 側の単体テストはリポジトリをモックにしているため、保存の順序(外部キー)や、
// トランザクションの外から読む値と中で書く値の食い違いは確かめられない。ここでは
// 「作成 → 編集 → デッキ編集 → 削除」を続けて行い、各段階でDBの中身を見る。
func TestIntegrationDeckSpriteSync(t *testing.T) {
	db := setupIntegrationDB(t,
		"deck_code_post_imports", "deck_code_post_likes", "deck_code_posts",
		"deck_code_pokemon_sprites", "deck_pokemon_sprites", "pokemon_sprites",
		"deck_code_tags", "deck_tags", "user_favorite_decks", "deck_codes", "decks", "users",
	)
	ctx := context.Background()

	deckRepository := NewDeck(db)
	deckCodeRepository := NewDeckCode(db)

	// デッキコードは空で作る(外部サイトへのアップロードと称号評価を通らない)ので、
	// deckAsset と badgeEvaluation は呼ばれない。
	deckCodeUsecase := usecase.NewDeckCode(
		deckCodeRepository, deckRepository, nil, NewTag(db), nil,
		NewTransactionManager(db), NewDeckCodePost(db),
	)
	deckUsecase := usecase.NewDeck(
		deckRepository, nil, NewUserFavoriteDeck(db), NewTag(db),
		NewTransactionManager(db), nil, NewDeckCodePost(db), deckCodeRepository,
	)

	now := time.Now().Local().Truncate(time.Microsecond)
	uid := "sync-uid-0000000000000000001"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TS1"
	firstId := "01HD7Y3K8D6FDHMHTZ2GT41TS2"

	require.NoError(t, db.Create(model.NewUser(uid, now, "持ち主", "")).Error)
	require.NoError(t, db.Create(model.NewDeck(deckId, now.Add(-time.Hour), sql.NullTime{}, uid, "ドラパルト", true)).Error)
	// 最初のバージョンは未設定(この機能より前に作られたバージョンを想定)
	require.NoError(t, db.Create(model.NewDeckCode(firstId, now.Add(-time.Hour), uid, deckId, "", true, "")).Error)
	for id, name := range map[string]string{"0887": "ドラパルト", "0477": "ヨノワール", "0006": "リザードン", "0025": "ピカチュウ"} {
		require.NoError(t, db.Create(&model.PokemonSprite{ID: id, Name: name}).Error)
	}
	require.NoError(t, db.Create(model.NewDeckPokemonSprite(deckId, 1, "0887")).Error)
	require.NoError(t, db.Create(model.NewDeckPokemonSprite(deckId, 2, "0477")).Error)

	layout := func(sprites []*entity.PokemonSprite) string {
		ret := ""
		for _, sprite := range sprites {
			ret += fmt.Sprintf("%d:%s ", sprite.Position, sprite.ID)
		}
		return ret
	}
	deckLayout := func() string {
		deck, err := deckRepository.FindById(ctx, deckId)
		require.NoError(t, err)
		return layout(deck.PokemonSprites)
	}
	versionLayout := func(id string) string {
		deckcode, err := deckCodeRepository.FindById(ctx, id)
		require.NoError(t, err)
		return layout(deckcode.PokemonSprites)
	}
	params := func(ids ...string) []*usecase.PokemonSpriteParam {
		ret := make([]*usecase.PokemonSpriteParam, 0, len(ids))
		for index, id := range ids {
			ret = append(ret, usecase.NewPokemonSpriteParamWithPosition(id, uint(index+1)))
		}
		return ret
	}

	const dusknoir = "1:0887 2:0477 "
	const charizard = "1:0887 2:0006 "
	const pikachu = "1:0025 2:0006 "

	var secondId, thirdId string

	t.Run("相棒を替えた新バージョンを作るとデッキも替わり未設定だった過去バージョンは元の相棒で固定される", func(t *testing.T) {
		created, err := deckCodeUsecase.Create(ctx, usecase.NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, params("0887", "0006")))
		require.NoError(t, err)
		secondId = created.ID

		require.Equal(t, charizard, versionLayout(secondId))
		require.Equal(t, charizard, deckLayout())
		require.Equal(t, dusknoir, versionLayout(firstId))
	})

	t.Run("スプライトを省略して作った新バージョンは直前のバージョンを引き継ぎデッキは変わらない", func(t *testing.T) {
		time.Sleep(2 * time.Millisecond) // 作成日時の順(最新の判定)を確実にする
		created, err := deckCodeUsecase.Create(ctx, usecase.NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, nil))
		require.NoError(t, err)
		thirdId = created.ID

		require.Equal(t, charizard, versionLayout(thirdId))
		require.Equal(t, charizard, deckLayout())
	})

	t.Run("過去のバージョンのスプライトを変えてもデッキは変わらない", func(t *testing.T) {
		_, err := deckCodeUsecase.Update(ctx, secondId, usecase.NewDeckCodeUpdateParam(false, "", nil, params("0025")))
		require.NoError(t, err)

		require.Equal(t, "1:0025 ", versionLayout(secondId))
		require.Equal(t, charizard, deckLayout())
	})

	t.Run("デッキのスプライトを変えると最新バージョンだけが同じになる", func(t *testing.T) {
		deck, err := deckUsecase.Update(ctx, deckId, usecase.NewDeckUpdateParam("ドラパルト", true, params("0025", "0006"), nil))
		require.NoError(t, err)
		require.Equal(t, pikachu, layout(deck.LatestDeckCode.PokemonSprites), "レスポンスの最新バージョンにも反映される")

		require.Equal(t, pikachu, deckLayout())
		require.Equal(t, pikachu, versionLayout(thirdId))
		require.Equal(t, "1:0025 ", versionLayout(secondId))
		require.Equal(t, dusknoir, versionLayout(firstId))
	})

	t.Run("最新バージョンのスプライトを変えるとデッキも同じになる", func(t *testing.T) {
		_, err := deckCodeUsecase.Update(ctx, thirdId, usecase.NewDeckCodeUpdateParam(false, "", nil, params("0887", "0006")))
		require.NoError(t, err)

		require.Equal(t, charizard, versionLayout(thirdId))
		require.Equal(t, charizard, deckLayout())
	})

	t.Run("存在しないスプライトIDで最新バージョンを更新すると何も変わらない", func(t *testing.T) {
		_, err := deckCodeUsecase.Update(ctx, thirdId, usecase.NewDeckCodeUpdateParam(false, "変更後", nil, params("9999")))
		require.Error(t, err)

		// バージョン本体(メモ)・スプライト・デッキのどれもロールバックされる
		deckcode, err := deckCodeRepository.FindById(ctx, thirdId)
		require.NoError(t, err)
		require.Empty(t, deckcode.Memo)
		require.Equal(t, charizard, versionLayout(thirdId))
		require.Equal(t, charizard, deckLayout())
	})

	t.Run("最新バージョンを消すと繰り上がったバージョンのスプライトがデッキになる", func(t *testing.T) {
		require.NoError(t, deckCodeUsecase.Delete(ctx, thirdId))

		// 繰り上がるのは secondId(ピカチュウ1体)
		require.Equal(t, "1:0025 ", deckLayout())
		require.Equal(t, dusknoir, versionLayout(firstId))
	})
}
