package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// spyDeckCodeBadgeEvaluation はEvaluateOnDeckCodeCreatedの呼び出し有無だけを
// 記録する手書きスタブ(stubBadgeEvaluationと同じ理由でgomockを使わない)。
type spyDeckCodeBadgeEvaluation struct {
	stubBadgeEvaluation
	called *bool
}

func (s spyDeckCodeBadgeEvaluation) EvaluateOnDeckCodeCreated(
	ctx context.Context,
	userId string,
	deckCode *entity.DeckCode,
) {
	*s.called = true
}

func setup4DeckCodeUsecase(t *testing.T) (
	*mock_repository.MockDeckCodeInterface,
	*mock_repository.MockDeckAssetInterface,
	*bool,
	*mock_repository.MockDeckCodePostInterface,
	DeckCodeInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)
	mockDeckAsset := mock_repository.NewMockDeckAssetInterface(mockCtrl)

	// タグ同期は Create/Update のたびに呼ばれる。タグ自体の検証は別テストで行うため、
	// ここでは呼び出しを素通り(付与なし)にする。
	mockTagRepository := mock_repository.NewMockTagInterface(mockCtrl)
	mockTagRepository.EXPECT().
		FindAttachableByIds(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*entity.Tag{}, nil).AnyTimes()
	mockTagRepository.EXPECT().
		ReplaceDeckCodeTags(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()
	// 最新バージョンのタグをデッキへ写す処理(deck_tag_sync.go)。写す・写さないの判定は
	// TestDeckCodeUsecase_SyncDeckTags で検証する。
	mockTagRepository.EXPECT().
		ReplaceDeckTags(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	// みんなの公開デッキの投稿。バージョン削除の連動(取り下げ)を Delete のテストで検証する。
	mockDeckCodePost := mock_repository.NewMockDeckCodePostInterface(mockCtrl)

	// 作成時の所有者検証(ownership.go)が deck_id のデッキを引く。ここでは常に本人のデッキとして
	// 返し、他人・存在しないデッキの扱いは Create の専用ケースで別のモックを使って検証する。
	mockDeckRepository := mock_repository.NewMockDeckInterface(mockCtrl)
	mockDeckRepository.EXPECT().
		FindById(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) (*entity.Deck, error) {
			return &entity.Deck{ID: id, UserId: deckCodeTestUid}, nil
		}).AnyTimes()
	// 最新バージョンのスプライトをデッキへ写す処理(deck_sprite_sync.go)。ここでは素通りにし、
	// 写す・写さないの判定は deck_sprite_sync_test.go で検証する。
	mockDeckRepository.EXPECT().
		ReplacePokemonSprites(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	badgeEvaluationCalled := false
	usecase := NewDeckCode(mockRepository, mockDeckRepository, mockDeckAsset, mockTagRepository, spyDeckCodeBadgeEvaluation{called: &badgeEvaluationCalled}, stubTransactionManager{}, mockDeckCodePost)

	return mockRepository, mockDeckAsset, &badgeEvaluationCalled, mockDeckCodePost, usecase
}

// deckCodeTestUid はデッキコードのテストで一貫して使うユーザーID。setup4DeckCodeUsecase の
// デッキ所有者スタブもこの値を持ち主として返す。
const deckCodeTestUid = "zor5SLfEfwfZ90yRVXzlxBEFARy2"

func TestDeckCodeUsecase(t *testing.T) {
	uid := deckCodeTestUid
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	code := "5dbFbk-uBwjqP-VVk5Vv"

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDのデッキコードを返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			deckCode := &entity.DeckCode{ID: id}

			mockRepository.EXPECT().FindById(context.Background(), id).Return(deckCode, nil)

			ret, err := usecase.FindById(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

			ret, err := usecase.FindById(context.Background(), id)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindByDeckId", func(t *testing.T) {
		t.Run("正常系_指定デッキのデッキコード一覧を返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			deckCodes := []*entity.DeckCode{{ID: "01HD7Y3K8D6FDHMHTZ2GT41TC1", DeckId: deckId}}

			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(deckCodes, nil)

			ret, err := usecase.FindByDeckId(context.Background(), deckId)

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, deckId, ret[0].DeckId)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, errors.New(""))

			ret, err := usecase.FindByDeckId(context.Background(), deckId)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_コード未指定なら外部アップロードと称号評価なしで保存する", func(t *testing.T) {
			mockRepository, _, badgeEvaluationCalled, _, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, nil)

			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), param)

			require.NoError(t, err)
			require.NotEmpty(t, ret.ID)
			require.NotEmpty(t, ret.CreatedAt)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, deckId, ret.DeckId)
			require.Empty(t, ret.Code)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("正常系_コード指定時はHTMLと画像をアップロードして保存し称号評価する", func(t *testing.T) {
			mockRepository, mockDeckAsset, badgeEvaluationCalled, _, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, code, true, "メモ", nil, nil)

			gomock.InOrder(
				mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), code).Return(nil),
				mockDeckAsset.EXPECT().UploadDeckImage(context.Background(), code).Return(nil),
				mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil),
				mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil),
			)

			ret, err := usecase.Create(context.Background(), param)

			require.NoError(t, err)
			require.Equal(t, code, ret.Code)
			require.True(t, ret.PrivateCodeFlg)
			require.Equal(t, "メモ", ret.Memo)
			require.True(t, *badgeEvaluationCalled)
		})

		t.Run("正常系_スプライト指定時は本体の保存後にスプライトを保存する", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			sprites := []*PokemonSpriteParam{
				NewPokemonSpriteParamWithPosition("0887", 1),
				NewPokemonSpriteParamWithPosition("0006", 2),
			}
			param := NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, sprites)

			gomock.InOrder(
				mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil),
				mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil),
				mockRepository.EXPECT().
					ReplacePokemonSprites(context.Background(), gomock.Any(), []*entity.PokemonSprite{
						entity.NewPokemonSpriteWithPosition("0887", 1),
						entity.NewPokemonSpriteWithPosition("0006", 2),
					}).
					Return(nil),
			)

			ret, err := usecase.Create(context.Background(), param)

			require.NoError(t, err)
			require.Len(t, ret.PokemonSprites, 2)
			require.Equal(t, "0887", ret.PokemonSprites[0].ID)
			require.Equal(t, uint(2), ret.PokemonSprites[1].Position)
		})

		// 空配列は「未設定で作る」。行を作らず(未設定＝親デッキのスプライトを表示)、
		// 直前のバージョンも引かない。レスポンスは nil ではなく空で返る。
		t.Run("正常系_スプライトが空スライスなら引き継がず保存もせず空で返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(
				context.Background(),
				NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, []*PokemonSpriteParam{}),
			)

			require.NoError(t, err)
			require.NotNil(t, ret.PokemonSprites)
			require.Empty(t, ret.PokemonSprites)
		})

		// nil(項目ごと省略)は、この項目を知らない古いクライアントからの作成。
		// 直前のバージョン(一覧の先頭＝最も新しいもの)の設定を引き継ぐ。
		t.Run("正常系_スプライトがnilなら直前のバージョンの設定を引き継ぐ", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			latest := &entity.DeckCode{ID: "latest", PokemonSprites: []*entity.PokemonSprite{
				entity.NewPokemonSpriteWithPosition("0887", 1),
				entity.NewPokemonSpriteWithPosition("0006", 2),
			}}
			older := &entity.DeckCode{ID: "older", PokemonSprites: []*entity.PokemonSprite{
				entity.NewPokemonSpriteWithPosition("0477", 1),
			}}

			gomock.InOrder(
				mockRepository.EXPECT().
					FindByDeckId(context.Background(), deckId).
					Return([]*entity.DeckCode{latest, older}, nil),
				mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil),
				mockRepository.EXPECT().
					ReplacePokemonSprites(context.Background(), gomock.Any(), []*entity.PokemonSprite{
						entity.NewPokemonSpriteWithPosition("0887", 1),
						entity.NewPokemonSpriteWithPosition("0006", 2),
					}).
					Return(nil),
			)

			ret, err := usecase.Create(
				context.Background(),
				NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, nil),
			)

			require.NoError(t, err)
			require.Len(t, ret.PokemonSprites, 2)
			require.Equal(t, "0006", ret.PokemonSprites[1].ID)
			// 引き継ぎ元のエンティティを共有しない(片方の書き換えがもう片方へ漏れない)
			require.NotSame(t, latest.PokemonSprites[0], ret.PokemonSprites[0])
		})

		t.Run("正常系_スプライトがnilで直前のバージョンが未設定なら保存せず空で返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			mockRepository.EXPECT().
				FindByDeckId(context.Background(), deckId).
				Return([]*entity.DeckCode{{ID: "latest"}}, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(
				context.Background(),
				NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, nil),
			)

			require.NoError(t, err)
			require.NotNil(t, ret.PokemonSprites)
			require.Empty(t, ret.PokemonSprites)
		})

		t.Run("異常系_引き継ぎ元の取得に失敗したら保存しない", func(t *testing.T) {
			mockRepository, _, badgeEvaluationCalled, _, usecase := setup4DeckCodeUsecase(t)

			mockRepository.EXPECT().
				FindByDeckId(context.Background(), deckId).
				Return(nil, errors.New(""))

			ret, err := usecase.Create(
				context.Background(),
				NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, nil),
			)

			require.Error(t, err)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("異常系_存在しないスプライトIDはErrInvalidReferenceを返し称号評価しない", func(t *testing.T) {
			mockRepository, mockDeckAsset, badgeEvaluationCalled, _, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, code, false, "", nil, []*PokemonSpriteParam{
				NewPokemonSpriteParamWithPosition("9999", 1),
			})

			mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), code).Return(nil)
			mockDeckAsset.EXPECT().UploadDeckImage(context.Background(), code).Return(nil)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)
			mockRepository.EXPECT().
				ReplacePokemonSprites(context.Background(), gomock.Any(), gomock.Any()).
				Return(apperror.ErrInvalidReference)

			ret, err := usecase.Create(context.Background(), param)

			require.ErrorIs(t, err, apperror.ErrInvalidReference)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("異常系_HTMLアップロード失敗時は画像アップロードも保存も行わない", func(t *testing.T) {
			_, mockDeckAsset, badgeEvaluationCalled, _, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, code, false, "", nil, nil)

			mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), code).Return(errors.New(""))

			ret, err := usecase.Create(context.Background(), param)

			require.Error(t, err)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("異常系_画像アップロード失敗時は保存を行わない", func(t *testing.T) {
			_, mockDeckAsset, badgeEvaluationCalled, _, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, code, false, "", nil, nil)

			gomock.InOrder(
				mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), code).Return(nil),
				mockDeckAsset.EXPECT().UploadDeckImage(context.Background(), code).Return(errors.New("")),
			)

			ret, err := usecase.Create(context.Background(), param)

			require.Error(t, err)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("異常系_保存失敗時はエラーを返し称号評価しない", func(t *testing.T) {
			mockRepository, _, badgeEvaluationCalled, _, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, nil)

			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

			ret, err := usecase.Create(context.Background(), param)

			require.Error(t, err)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("正常系_公開設定とメモのみ更新され他は維持される", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			createdAt := time.Now().Local()
			existing := entity.NewDeckCode(id, createdAt, uid, deckId, code, false, "更新前のメモ")

			mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
			// 最新バージョンかどうかの確認。ここでは最新ではない(デッキへは写さない)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

			param := NewDeckCodeUpdateParam(true, "更新後のメモ", nil, nil)

			ret, err := usecase.Update(context.Background(), id, param)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
			require.Equal(t, createdAt, ret.CreatedAt)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, deckId, ret.DeckId)
			require.Equal(t, code, ret.Code)
			require.True(t, ret.PrivateCodeFlg)
			require.Equal(t, "更新後のメモ", ret.Memo)
		})

		// スプライトの項目を知らない古いクライアントは nil で届く。メモだけの更新で
		// 設定済みのスプライトが消えないこと(ReplacePokemonSprites を呼ばない)を確かめる。
		t.Run("正常系_スプライトがnilなら置き換えず現在の設定を返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			existing := entity.NewDeckCode(id, time.Now().Local(), uid, deckId, code, false, "")
			existing.PokemonSprites = []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 1)}

			mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
			// 最新バージョンかどうかの確認。ここでは最新ではない(デッキへは写さない)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

			ret, err := usecase.Update(context.Background(), id, NewDeckCodeUpdateParam(false, "メモ", nil, nil))

			require.NoError(t, err)
			require.Len(t, ret.PokemonSprites, 1)
			require.Equal(t, "0887", ret.PokemonSprites[0].ID)
		})

		t.Run("正常系_スプライト指定時は置き換える", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			existing := entity.NewDeckCode(id, time.Now().Local(), uid, deckId, code, false, "")
			existing.PokemonSprites = []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 1)}

			mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
			// 最新バージョンかどうかの確認。ここでは最新ではない(デッキへは写さない)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)
			mockRepository.EXPECT().
				ReplacePokemonSprites(context.Background(), id, []*entity.PokemonSprite{
					entity.NewPokemonSpriteWithPosition("0006", 2),
				}).
				Return(nil)

			param := NewDeckCodeUpdateParam(false, "", nil, []*PokemonSpriteParam{
				NewPokemonSpriteParamWithPosition("0006", 2),
			})

			ret, err := usecase.Update(context.Background(), id, param)

			require.NoError(t, err)
			require.Len(t, ret.PokemonSprites, 1)
			require.Equal(t, "0006", ret.PokemonSprites[0].ID)
		})

		// 空スライス(nil ではない)は「未設定に戻す」。空で置き換えを呼ぶ。
		t.Run("正常系_スプライトが空スライスなら未設定に戻す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			existing := entity.NewDeckCode(id, time.Now().Local(), uid, deckId, code, false, "")
			existing.PokemonSprites = []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 1)}

			mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
			// 最新バージョンかどうかの確認。ここでは最新ではない(デッキへは写さない)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)
			mockRepository.EXPECT().
				ReplacePokemonSprites(context.Background(), id, []*entity.PokemonSprite{}).
				Return(nil)

			ret, err := usecase.Update(
				context.Background(),
				id,
				NewDeckCodeUpdateParam(false, "", nil, []*PokemonSpriteParam{}),
			)

			require.NoError(t, err)
			require.Empty(t, ret.PokemonSprites)
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.Update(context.Background(), id, NewDeckCodeUpdateParam(false, "", nil, nil))

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
		})

		t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			existing := entity.NewDeckCode(id, time.Now().Local(), uid, deckId, code, false, "")

			mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
			// 最新バージョンかどうかの確認。ここでは最新ではない(デッキへは写さない)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

			ret, err := usecase.Update(context.Background(), id, NewDeckCodeUpdateParam(false, "", nil, nil))

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_投稿を取り下げてからリポジトリのDeleteを呼び出す", func(t *testing.T) {
			mockRepository, _, _, mockDeckCodePost, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			gomock.InOrder(
				// 最新バージョンの繰り上がりを確かめるために、削除の前にバージョンを引く
				mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, DeckId: deckId}, nil),
				mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return([]*entity.DeckCode{{ID: id, DeckId: deckId}}, nil),
				// 削除したバージョンがみんなの公開デッキに残らないよう、先に取り下げる
				mockDeckCodePost.EXPECT().UnpublishByDeckCodeId(context.Background(), id, gomock.Any()).Return(nil),
				mockRepository.EXPECT().Delete(context.Background(), id).Return(nil),
			)

			require.NoError(t, usecase.Delete(context.Background(), id))
		})

		t.Run("異常系_取り下げに失敗したら削除しない", func(t *testing.T) {
			mockRepository, _, _, mockDeckCodePost, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, DeckId: deckId}, nil)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockDeckCodePost.EXPECT().UnpublishByDeckCodeId(context.Background(), id, gomock.Any()).Return(errors.New(""))

			require.Error(t, usecase.Delete(context.Background(), id))
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundを返し何も消さない", func(t *testing.T) {
			mockRepository, _, _, _, usecase := setup4DeckCodeUsecase(t)

			mockRepository.EXPECT().FindById(context.Background(), "missing").Return(nil, apperror.ErrRecordNotFound)

			require.ErrorIs(t, usecase.Delete(context.Background(), "missing"), apperror.ErrRecordNotFound)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, _, _, mockDeckCodePost, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, DeckId: deckId}, nil)
			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, nil)
			mockDeckCodePost.EXPECT().UnpublishByDeckCodeId(context.Background(), id, gomock.Any()).Return(nil)
			mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

			require.Error(t, usecase.Delete(context.Background(), id))
		})
	})
}

// setup4DeckCodeTagSyncUsecase は、デッキと最新バージョンのタグを揃える処理を確かめるための組み立て。
// setup4DeckCodeUsecase はタグのリポジトリを素通り(AnyTimes)にしているため、
// ReplaceDeckTags の呼ばれ方を検証できるよう、タグのモックを返す別の組み立てにする。
// attachable は付与できるタグ(FindAttachableByIds の戻り)、deckTags はデッキのいまのタグ。
func setup4DeckCodeTagSyncUsecase(t *testing.T, attachable []*entity.Tag, deckTags []*entity.Tag) (
	*mock_repository.MockDeckCodeInterface,
	*mock_repository.MockTagInterface,
	DeckCodeInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)
	mockDeckAsset := mock_repository.NewMockDeckAssetInterface(mockCtrl)

	mockTagRepository := mock_repository.NewMockTagInterface(mockCtrl)
	mockTagRepository.EXPECT().
		FindAttachableByIds(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(attachable, nil).AnyTimes()
	mockTagRepository.EXPECT().
		ReplaceDeckCodeTags(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	mockDeckRepository := mock_repository.NewMockDeckInterface(mockCtrl)
	mockDeckRepository.EXPECT().
		FindById(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) (*entity.Deck, error) {
			return &entity.Deck{ID: id, UserId: deckCodeTestUid, Tags: deckTags}, nil
		}).AnyTimes()

	mockRepository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	mockDeckCodePost := mock_repository.NewMockDeckCodePostInterface(mockCtrl)
	mockDeckCodePost.EXPECT().UnpublishByDeckCodeId(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	usecase := NewDeckCode(mockRepository, mockDeckRepository, mockDeckAsset, mockTagRepository, stubBadgeEvaluation{}, stubTransactionManager{}, mockDeckCodePost)

	return mockRepository, mockTagRepository, usecase
}

// デッキのタグと最新バージョンのタグを揃える(deck_tag_sync.go)。
// バージョンの作成・最新の編集・最新の削除(繰り上がり)で、デッキのタグを最新バージョンのタグにする。
func TestDeckCodeUsecase_SyncDeckTags(t *testing.T) {
	uid := deckCodeTestUid
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	ctx := context.Background()

	aceSpec := &entity.Tag{ID: "01HD7Y3K8D6FDHMHTZ2GT41TA1", PresetFlg: true, PresetCategory: entity.TagPresetCategoryAceSpec}
	mine := &entity.Tag{ID: "01HD7Y3K8D6FDHMHTZ2GT41TA2", UserId: uid}

	version := func(id string, tags ...*entity.Tag) *entity.DeckCode {
		deckcode := entity.NewDeckCode(id, time.Now().Local(), uid, deckId, "", false, "")
		deckcode.Tags = tags
		return deckcode
	}

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_付けたタグでデッキのタグを付与順のまま置き換える", func(t *testing.T) {
			// 付与できるタグの戻り順は不定。付与順(指定順)に並べ直したものがデッキにも付く
			_, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{aceSpec, mine}, nil)

			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{mine.ID, aceSpec.ID}).Return(nil)

			ret, err := usecase.Create(ctx, NewDeckCodeCreateParam(uid, deckId, "", false, "", []string{mine.ID, aceSpec.ID}, []*PokemonSpriteParam{}))

			require.NoError(t, err)
			require.Len(t, ret.Tags, 2)
		})

		t.Run("正常系_付与できないタグはデッキにも付けない", func(t *testing.T) {
			// 他人のタグ(付与できない)は FindAttachableByIds で落ちる
			_, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{mine}, nil)

			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{mine.ID}).Return(nil)

			_, err := usecase.Create(ctx, NewDeckCodeCreateParam(uid, deckId, "", false, "", []string{"01HD7Y3K8D6FDHMHTZ2GT41TZZ", mine.ID}, []*PokemonSpriteParam{}))

			require.NoError(t, err)
		})

		// 空はユーザーがタグを外した結果。スプライトの「空＝未設定」と違い、デッキのタグも外す
		t.Run("正常系_タグが空スライスならデッキのタグも外す", func(t *testing.T) {
			_, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{mine}, []*entity.Tag{mine})

			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{}).Return(nil)

			ret, err := usecase.Create(ctx, NewDeckCodeCreateParam(uid, deckId, "", false, "", []string{}, []*PokemonSpriteParam{}))

			require.NoError(t, err)
			require.Empty(t, ret.Tags)
		})

		// nil(項目ごと省略)は、この項目を知らない古いクライアントからの作成。
		// デッキのいまのタグを引き継ぐので、デッキのタグは変わらない(消えない)
		t.Run("正常系_タグがnilならデッキのいまのタグを引き継ぐ", func(t *testing.T) {
			_, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{mine}, []*entity.Tag{mine})

			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{mine.ID}).Return(nil)

			ret, err := usecase.Create(ctx, NewDeckCodeCreateParam(uid, deckId, "", false, "", nil, []*PokemonSpriteParam{}))

			require.NoError(t, err)
			require.Len(t, ret.Tags, 1)
			require.Equal(t, mine.ID, ret.Tags[0].ID)
		})

		t.Run("異常系_デッキのタグの置き換えに失敗したらエラーを返す", func(t *testing.T) {
			_, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{mine}, nil)

			mockTag.EXPECT().ReplaceDeckTags(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New(""))

			ret, err := usecase.Create(ctx, NewDeckCodeCreateParam(uid, deckId, "", false, "", []string{mine.ID}, []*PokemonSpriteParam{}))

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("正常系_最新バージョンのタグを変えるとデッキへ写す", func(t *testing.T) {
			mockRepository, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{aceSpec}, nil)

			mockRepository.EXPECT().FindById(ctx, "v2").Return(version("v2", mine), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return([]*entity.DeckCode{version("v2", mine), version("v1")}, nil)
			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{aceSpec.ID}).Return(nil)

			ret, err := usecase.Update(ctx, "v2", NewDeckCodeUpdateParam(false, "", []string{aceSpec.ID}, nil))

			require.NoError(t, err)
			require.Len(t, ret.Tags, 1)
		})

		t.Run("正常系_最新バージョンのタグを全部外すとデッキのタグも外す", func(t *testing.T) {
			mockRepository, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{}, nil)

			mockRepository.EXPECT().FindById(ctx, "v2").Return(version("v2", mine), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return([]*entity.DeckCode{version("v2", mine)}, nil)
			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{}).Return(nil)

			_, err := usecase.Update(ctx, "v2", NewDeckCodeUpdateParam(false, "", []string{}, nil))

			require.NoError(t, err)
		})

		t.Run("正常系_過去のバージョンのタグを変えてもデッキは変えない", func(t *testing.T) {
			mockRepository, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{aceSpec}, nil)

			mockRepository.EXPECT().FindById(ctx, "v1").Return(version("v1"), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return([]*entity.DeckCode{version("v2", mine), version("v1")}, nil)
			mockTag.EXPECT().ReplaceDeckTags(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, err := usecase.Update(ctx, "v1", NewDeckCodeUpdateParam(false, "", []string{aceSpec.ID}, nil))

			require.NoError(t, err)
		})

		t.Run("異常系_バージョン一覧を引けなければ保存しない", func(t *testing.T) {
			mockRepository, _, usecase := setup4DeckCodeTagSyncUsecase(t, []*entity.Tag{}, nil)

			mockRepository.EXPECT().FindById(ctx, "v2").Return(version("v2"), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return(nil, errors.New(""))

			ret, err := usecase.Update(ctx, "v2", NewDeckCodeUpdateParam(false, "", []string{}, nil))

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_最新バージョンを消すと繰り上がったバージョンのタグをデッキへ写す", func(t *testing.T) {
			mockRepository, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, nil, nil)

			mockRepository.EXPECT().FindById(ctx, "v3").Return(version("v3", aceSpec), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return([]*entity.DeckCode{version("v3", aceSpec), version("v2", mine), version("v1")}, nil)
			mockRepository.EXPECT().Delete(ctx, "v3").Return(nil)
			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{mine.ID}).Return(nil)

			require.NoError(t, usecase.Delete(ctx, "v3"))
		})

		t.Run("正常系_繰り上がったバージョンがタグなしならデッキのタグも外す", func(t *testing.T) {
			mockRepository, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, nil, nil)

			mockRepository.EXPECT().FindById(ctx, "v2").Return(version("v2", aceSpec), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return([]*entity.DeckCode{version("v2", aceSpec), version("v1")}, nil)
			mockRepository.EXPECT().Delete(ctx, "v2").Return(nil)
			mockTag.EXPECT().ReplaceDeckTags(ctx, deckId, []string{}).Return(nil)

			require.NoError(t, usecase.Delete(ctx, "v2"))
		})

		t.Run("正常系_過去のバージョンや最後の1件を消してもデッキは変えない", func(t *testing.T) {
			mockRepository, mockTag, usecase := setup4DeckCodeTagSyncUsecase(t, nil, nil)

			mockRepository.EXPECT().FindById(ctx, "v1").Return(version("v1"), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return([]*entity.DeckCode{version("v2", mine), version("v1")}, nil)
			mockRepository.EXPECT().Delete(ctx, "v1").Return(nil)
			mockRepository.EXPECT().FindById(ctx, "v2").Return(version("v2", mine), nil)
			mockRepository.EXPECT().FindByDeckId(ctx, deckId).Return([]*entity.DeckCode{version("v2", mine)}, nil)
			mockRepository.EXPECT().Delete(ctx, "v2").Return(nil)
			mockTag.EXPECT().ReplaceDeckTags(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			require.NoError(t, usecase.Delete(ctx, "v1"))
			require.NoError(t, usecase.Delete(ctx, "v2"))
		})
	})
}
