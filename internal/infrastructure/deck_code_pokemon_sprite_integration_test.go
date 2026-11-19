package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// デッキコード(バージョン)のスプライトを実DBで検証する。
//
// バージョンのスプライトは deck_code_pokemon_sprites に持ち、行が無いバージョンは「未設定」として
// 親デッキのスプライトへフォールバックする。sqlmock では外部キー・主キーと、みんなの公開デッキの
// 絞り込み(CASE でバージョン/デッキを切り替える SQL)の正しさを確かめられないため、ここで見る。
func TestIntegrationDeckCodePokemonSprites(t *testing.T) {
	db := setupIntegrationDB(t,
		"deck_code_post_imports", "deck_code_post_likes", "deck_code_posts",
		"deck_code_pokemon_sprites", "deck_pokemon_sprites", "pokemon_sprites",
		"deck_code_tags", "deck_tags", "user_favorite_decks", "deck_codes", "decks", "users",
	)
	ctx := context.Background()

	deckCodeRepository := NewDeckCode(db)
	deckRepository := NewDeck(db)
	postRepository := NewDeckCodePost(db)

	now := time.Now().Local().Truncate(time.Microsecond)
	uid := "author-uid-000000000000000001"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TD1"
	oldDeckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC1" // スプライト未設定のまま残すバージョン
	newDeckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC2" // スプライトを設定する最新バージョン

	// デッキのスプライトは「ドラパルト / ヨノワール」。最新バージョンだけ相棒をリザードンへ替える。
	require.NoError(t, db.Create(model.NewUser(uid, now, "投稿者", "")).Error)
	require.NoError(t, db.Create(model.NewDeck(deckId, now, sql.NullTime{}, uid, "ドラパルト", true)).Error)
	require.NoError(t, db.Create(model.NewDeckCode(oldDeckCodeId, now.Add(-time.Hour), uid, deckId, "kVkFF5-pQ2sZa-VFVfkV", true, "")).Error)
	require.NoError(t, db.Create(model.NewDeckCode(newDeckCodeId, now, uid, deckId, "5dbFbk-uBwjqP-VVk5Vv", true, "")).Error)
	require.NoError(t, db.Create(&model.PokemonSprite{ID: "0887", Name: "ドラパルト"}).Error)
	require.NoError(t, db.Create(&model.PokemonSprite{ID: "0477", Name: "ヨノワール"}).Error)
	require.NoError(t, db.Create(&model.PokemonSprite{ID: "0006", Name: "リザードン"}).Error)
	require.NoError(t, db.Create(model.NewDeckPokemonSprite(deckId, 1, "0887")).Error)
	require.NoError(t, db.Create(model.NewDeckPokemonSprite(deckId, 2, "0477")).Error)

	t.Run("正常系_設定したスプライトをバージョンごとに取得でき未設定のバージョンは空になる", func(t *testing.T) {
		require.NoError(t, deckCodeRepository.ReplacePokemonSprites(ctx, newDeckCodeId, []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("0887", 1),
			entity.NewPokemonSpriteWithPosition("0006", 2),
		}))

		ret, err := deckCodeRepository.FindById(ctx, newDeckCodeId)
		require.NoError(t, err)
		require.Len(t, ret.PokemonSprites, 2)
		require.Equal(t, "0887", ret.PokemonSprites[0].ID)
		require.Equal(t, uint(1), ret.PokemonSprites[0].Position)
		require.Equal(t, "0006", ret.PokemonSprites[1].ID)
		require.Equal(t, uint(2), ret.PokemonSprites[1].Position)

		// 一覧は新しい順。未設定のバージョンにデッキのスプライトを混ぜない(フォールバックは表示側の責務)
		list, err := deckCodeRepository.FindByDeckId(ctx, deckId)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, newDeckCodeId, list[0].ID)
		require.Len(t, list[0].PokemonSprites, 2)
		require.Equal(t, oldDeckCodeId, list[1].ID)
		require.Empty(t, list[1].PokemonSprites)
	})

	t.Run("正常系_デッキの取得では最新バージョンのスプライトも載りデッキ自体のスプライトは変わらない", func(t *testing.T) {
		deck, err := deckRepository.FindById(ctx, deckId)
		require.NoError(t, err)

		require.Len(t, deck.PokemonSprites, 2)
		require.Equal(t, "0477", deck.PokemonSprites[1].ID, "デッキのスプライトはバージョンの設定に引きずられない")

		require.Equal(t, newDeckCodeId, deck.LatestDeckCode.ID)
		require.Len(t, deck.LatestDeckCode.PokemonSprites, 2)
		require.Equal(t, "0006", deck.LatestDeckCode.PokemonSprites[1].ID)
	})

	t.Run("正常系_置き換えは枠を保ち空を渡すと未設定に戻る", func(t *testing.T) {
		// 1枠目を空にして2枠目だけ残す。position は詰めずに 2 のまま保存される
		require.NoError(t, deckCodeRepository.ReplacePokemonSprites(ctx, oldDeckCodeId, []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("0006", 2),
		}))

		ret, err := deckCodeRepository.FindById(ctx, oldDeckCodeId)
		require.NoError(t, err)
		require.Len(t, ret.PokemonSprites, 1)
		require.Equal(t, uint(2), ret.PokemonSprites[0].Position)

		require.NoError(t, deckCodeRepository.ReplacePokemonSprites(ctx, oldDeckCodeId, []*entity.PokemonSprite{}))

		ret, err = deckCodeRepository.FindById(ctx, oldDeckCodeId)
		require.NoError(t, err)
		require.Empty(t, ret.PokemonSprites)
	})

	t.Run("異常系_マスタに無いスプライトIDはErrInvalidReferenceで元の設定は消えない", func(t *testing.T) {
		err := deckCodeRepository.ReplacePokemonSprites(ctx, newDeckCodeId, []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("9999", 1),
		})
		require.ErrorIs(t, err, apperror.ErrInvalidReference)

		// 「全削除 → 再INSERT」の途中で失敗しても、削除ごとロールバックされる
		ret, err := deckCodeRepository.FindById(ctx, newDeckCodeId)
		require.NoError(t, err)
		require.Len(t, ret.PokemonSprites, 2)
	})

	t.Run("正常系_みんなの公開デッキは投稿したバージョンのスプライトを優先し未設定ならデッキのスプライトを使う", func(t *testing.T) {
		oldPostId := "01HD7Y3K8D6FDHMHTZ2GT41TP1"
		newPostId := "01HD7Y3K8D6FDHMHTZ2GT41TP2"
		require.NoError(t, postRepository.Save(ctx, entity.NewDeckCodePost(
			oldPostId, now, now, uid, deckId, oldDeckCodeId, now, time.Time{}, time.Time{}, "", "", 0,
		)))
		require.NoError(t, postRepository.Save(ctx, entity.NewDeckCodePost(
			newPostId, now, now, uid, deckId, newDeckCodeId, now, time.Time{}, time.Time{}, "", "", 0,
		)))

		// 未設定のバージョンの投稿はデッキのスプライト(ドラパルト / ヨノワール)
		oldPost, err := postRepository.FindById(ctx, oldPostId, uid)
		require.NoError(t, err)
		require.Len(t, oldPost.PokemonSprites, 2)
		require.Equal(t, "0477", oldPost.PokemonSprites[1].ID)

		// 設定済みのバージョンの投稿はバージョンのスプライト(ドラパルト / リザードン)
		newPost, err := postRepository.FindById(ctx, newPostId, uid)
		require.NoError(t, err)
		require.Len(t, newPost.PokemonSprites, 2)
		require.Equal(t, "0006", newPost.PokemonSprites[1].ID)

		find := func(spriteIds ...string) []string {
			posts, err := postRepository.Find(ctx, &repository.DeckCodePostFilter{
				PokemonSpriteIds: spriteIds,
				PopularSince:     now.Add(-time.Hour),
			}, 10, 0)
			require.NoError(t, err)

			ids := make([]string, 0, len(posts))
			for _, post := range posts {
				ids = append(ids, post.ID)
			}
			return ids
		}

		// 絞り込みも表示と同じスプライトを見る。リザードンで出るのはバージョンに設定した投稿だけ
		require.Equal(t, []string{newPostId}, find("0006"))
		// ヨノワールはデッキのスプライトだが、バージョンで上書きした投稿は表示していないので出さない
		require.Equal(t, []string{oldPostId}, find("0477"))
		// ドラパルトはどちらも表示している
		require.ElementsMatch(t, []string{oldPostId, newPostId}, find("0887"))
		require.Equal(t, []string{newPostId}, find("0887", "0006"))
	})

	// 最新バージョンのスプライトをデッキへ写すときの書き込み(usecase の deck_sprite_sync.go が使う)。
	t.Run("正常系_デッキのスプライトだけを置き換えられ失敗時は元のまま残る", func(t *testing.T) {
		require.NoError(t, deckRepository.ReplacePokemonSprites(ctx, deckId, []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("0887", 1),
			entity.NewPokemonSpriteWithPosition("0006", 2),
		}))

		deck, err := deckRepository.FindById(ctx, deckId)
		require.NoError(t, err)
		require.Equal(t, "ドラパルト", deck.Name, "本体には触れない")
		require.Len(t, deck.PokemonSprites, 2)
		require.Equal(t, "0006", deck.PokemonSprites[1].ID)

		err = deckRepository.ReplacePokemonSprites(ctx, deckId, []*entity.PokemonSprite{
			entity.NewPokemonSpriteWithPosition("9999", 1),
		})
		require.ErrorIs(t, err, apperror.ErrInvalidReference)

		deck, err = deckRepository.FindById(ctx, deckId)
		require.NoError(t, err)
		require.Len(t, deck.PokemonSprites, 2)
	})

	t.Run("正常系_退会時にバージョンのスプライトも消える", func(t *testing.T) {
		// deck_codes は論理削除のため、先に消し残すと辿れない行になる。
		// 投稿は deck_codes を参照しているので、ここでは退会処理と同じく先に片付ける。
		require.NoError(t, db.Exec("DELETE FROM deck_code_posts").Error)
		require.NoError(t, deckRepository.DeleteByUserId(ctx, uid))

		var count int64
		require.NoError(t, db.Table("deck_code_pokemon_sprites").Count(&count).Error)
		require.Zero(t, count)
	})
}
