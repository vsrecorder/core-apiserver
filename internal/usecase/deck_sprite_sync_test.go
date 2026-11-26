package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// デッキのスプライトと最新バージョンのスプライトが、書き込みのたびに揃うことを検証する。
// ここでは「デッキへ写す・写さない」「過去のバージョンへ書き残す」の判定を見る
// (保存の順序やエラーの扱いは deck_code_test.go / deck_test.go 側)。

const (
	syncUid    = "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	syncDeckId = "01HD7Y3K8D6FDHMHTZ2GT41TD1"
)

func syncSprites(pairs ...string) []*entity.PokemonSprite {
	ret := make([]*entity.PokemonSprite, 0, len(pairs))
	for index, id := range pairs {
		ret = append(ret, entity.NewPokemonSpriteWithPosition(id, uint(index+1)))
	}
	return ret
}

func syncSpriteParams(pairs ...string) []*PokemonSpriteParam {
	ret := make([]*PokemonSpriteParam, 0, len(pairs))
	for index, id := range pairs {
		ret = append(ret, NewPokemonSpriteParamWithPosition(id, uint(index+1)))
	}
	return ret
}

func syncVersion(id string, pokemonSprites []*entity.PokemonSprite) *entity.DeckCode {
	deckcode := entity.NewDeckCode(id, time.Now().Local(), syncUid, syncDeckId, "", false, "")
	deckcode.PokemonSprites = pokemonSprites
	return deckcode
}

type syncMocks struct {
	deckCode *mock_repository.MockDeckCodeInterface
	deck     *mock_repository.MockDeckInterface
	usecase  DeckCodeInterface
}

// デッキのスプライトを deckSprites として返すデッキコードのユースケースを作る。
// デッキ・バージョンへの書き込みは各ケースが期待として置く(置いていない書き込みは失敗になる)。
func setupSync(t *testing.T, deckSprites []*entity.PokemonSprite) syncMocks {
	mockCtrl := gomock.NewController(t)
	mockDeckCode := mock_repository.NewMockDeckCodeInterface(mockCtrl)
	mockDeck := mock_repository.NewMockDeckInterface(mockCtrl)

	mockDeck.EXPECT().
		FindById(gomock.Any(), syncDeckId).
		Return(&entity.Deck{ID: syncDeckId, UserId: syncUid, PokemonSprites: deckSprites}, nil).AnyTimes()

	mockTag := mock_repository.NewMockTagInterface(mockCtrl)
	mockTag.EXPECT().FindAttachableByIds(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*entity.Tag{}, nil).AnyTimes()
	mockTag.EXPECT().ReplaceDeckCodeTags(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// デッキのタグを揃える処理(deck_tag_sync.go)はここでは見ない
	mockTag.EXPECT().ReplaceDeckTags(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	mockDeckCodePost := mock_repository.NewMockDeckCodePostInterface(mockCtrl)
	mockDeckCodePost.EXPECT().UnpublishByDeckCodeId(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	usecase := NewDeckCode(
		mockDeckCode,
		mockDeck,
		mock_repository.NewMockDeckAssetInterface(mockCtrl),
		mockTag,
		spyDeckCodeBadgeEvaluation{called: new(bool)},
		stubTransactionManager{},
		mockDeckCodePost,
	)

	return syncMocks{mockDeckCode, mockDeck, usecase}
}

func TestDeckSpriteSync_CreateVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("正常系_相棒を替えた新バージョンを作るとデッキへ写し未設定の過去バージョンには元のスプライトを書き残す", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0477"))

		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{
			syncVersion("v2", nil),
			syncVersion("v1", syncSprites("0025")), // 設定済みのバージョンには触れない
		}, nil)
		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		gomock.InOrder(
			// 新しいバージョン自身
			m.deckCode.EXPECT().ReplacePokemonSprites(ctx, gomock.Not("v2"), syncSprites("0887", "0006")).Return(nil),
			// 未設定だった v2 は、デッキを置き換える前に元のデッキのスプライトで固定する
			m.deckCode.EXPECT().ReplacePokemonSprites(ctx, "v2", syncSprites("0887", "0477")).Return(nil),
			m.deck.EXPECT().ReplacePokemonSprites(ctx, syncDeckId, syncSprites("0887", "0006")).Return(nil),
		)

		_, err := m.usecase.Create(ctx, NewDeckCodeCreateParam(syncUid, syncDeckId, "", false, "", nil, syncSpriteParams("0887", "0006")))
		require.NoError(t, err)
	})

	t.Run("正常系_デッキと同じスプライトの新バージョンではデッキにも過去バージョンにも書かない", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0477"))

		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{syncVersion("v1", nil)}, nil)
		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		m.deckCode.EXPECT().ReplacePokemonSprites(ctx, gomock.Not("v1"), syncSprites("0887", "0477")).Return(nil)

		_, err := m.usecase.Create(ctx, NewDeckCodeCreateParam(syncUid, syncDeckId, "", false, "", nil, syncSpriteParams("0887", "0477")))
		require.NoError(t, err)
	})

	t.Run("正常系_未設定で作った新バージョンではデッキのスプライトを変えない", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0477"))

		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		_, err := m.usecase.Create(ctx, NewDeckCodeCreateParam(syncUid, syncDeckId, "", false, "", nil, []*PokemonSpriteParam{}))
		require.NoError(t, err)
	})

	t.Run("正常系_デッキにスプライトが無いときは過去バージョンへ書き残さずデッキへ写す", func(t *testing.T) {
		m := setupSync(t, nil)

		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{syncVersion("v1", nil)}, nil)
		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		m.deckCode.EXPECT().ReplacePokemonSprites(ctx, gomock.Not("v1"), syncSprites("0006")).Return(nil)
		m.deck.EXPECT().ReplacePokemonSprites(ctx, syncDeckId, syncSprites("0006")).Return(nil)

		_, err := m.usecase.Create(ctx, NewDeckCodeCreateParam(syncUid, syncDeckId, "", false, "", nil, syncSpriteParams("0006")))
		require.NoError(t, err)
	})

	t.Run("異常系_デッキへ写せなければ作成も失敗する", func(t *testing.T) {
		m := setupSync(t, nil)

		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return(nil, nil)
		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		m.deckCode.EXPECT().ReplacePokemonSprites(ctx, gomock.Any(), gomock.Any()).Return(nil)
		m.deck.EXPECT().ReplacePokemonSprites(ctx, syncDeckId, gomock.Any()).Return(apperror.ErrInvalidReference)

		ret, err := m.usecase.Create(ctx, NewDeckCodeCreateParam(syncUid, syncDeckId, "", false, "", nil, syncSpriteParams("0006")))
		require.ErrorIs(t, err, apperror.ErrInvalidReference)
		require.Nil(t, ret)
	})
}

func TestDeckSpriteSync_UpdateVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("正常系_最新バージョンのスプライトを変えるとデッキへ写し未設定の過去バージョンを固定する", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0477"))

		m.deckCode.EXPECT().FindById(ctx, "v2").Return(syncVersion("v2", nil), nil)
		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{
			syncVersion("v2", nil),
			syncVersion("v1", nil),
		}, nil)
		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		gomock.InOrder(
			m.deckCode.EXPECT().ReplacePokemonSprites(ctx, "v2", syncSprites("0887", "0006")).Return(nil),
			m.deckCode.EXPECT().ReplacePokemonSprites(ctx, "v1", syncSprites("0887", "0477")).Return(nil),
			m.deck.EXPECT().ReplacePokemonSprites(ctx, syncDeckId, syncSprites("0887", "0006")).Return(nil),
		)

		_, err := m.usecase.Update(ctx, "v2", NewDeckCodeUpdateParam(false, "", nil, syncSpriteParams("0887", "0006")))
		require.NoError(t, err)
	})

	t.Run("正常系_過去のバージョンのスプライトを変えてもデッキは変えない", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0477"))

		m.deckCode.EXPECT().FindById(ctx, "v1").Return(syncVersion("v1", nil), nil)
		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{
			syncVersion("v2", nil),
			syncVersion("v1", nil),
		}, nil)
		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		m.deckCode.EXPECT().ReplacePokemonSprites(ctx, "v1", syncSprites("0006")).Return(nil)

		_, err := m.usecase.Update(ctx, "v1", NewDeckCodeUpdateParam(false, "", nil, syncSpriteParams("0006")))
		require.NoError(t, err)
	})

	// 未設定の最新バージョンはデッキのスプライトで表示されるので、戻しただけで揃っている
	t.Run("正常系_最新バージョンを未設定に戻してもデッキは変えない", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0006"))

		m.deckCode.EXPECT().FindById(ctx, "v2").Return(syncVersion("v2", syncSprites("0887", "0006")), nil)
		// 最新バージョン(v2)を未設定に戻す
		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{
			syncVersion("v2", syncSprites("0887", "0006")),
		}, nil)
		m.deckCode.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		m.deckCode.EXPECT().ReplacePokemonSprites(ctx, "v2", []*entity.PokemonSprite{}).Return(nil)

		_, err := m.usecase.Update(ctx, "v2", NewDeckCodeUpdateParam(false, "", nil, []*PokemonSpriteParam{}))
		require.NoError(t, err)
	})
}

func TestDeckSpriteSync_DeleteVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("正常系_最新バージョンを消すと繰り上がったバージョンのスプライトをデッキへ写す", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0006"))

		m.deckCode.EXPECT().FindById(ctx, "v3").Return(syncVersion("v3", syncSprites("0887", "0006")), nil)
		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{
			syncVersion("v3", syncSprites("0887", "0006")),
			syncVersion("v2", syncSprites("0887", "0477")),
			syncVersion("v1", nil),
		}, nil)

		gomock.InOrder(
			m.deckCode.EXPECT().Delete(ctx, "v3").Return(nil),
			// 未設定の v1 は、消したバージョンの相棒(いまのデッキのスプライト)で固定する
			m.deckCode.EXPECT().ReplacePokemonSprites(ctx, "v1", syncSprites("0887", "0006")).Return(nil),
			m.deck.EXPECT().ReplacePokemonSprites(ctx, syncDeckId, syncSprites("0887", "0477")).Return(nil),
		)

		require.NoError(t, m.usecase.Delete(ctx, "v3"))
	})

	t.Run("正常系_繰り上がったバージョンが未設定ならデッキは変えない", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0006"))

		m.deckCode.EXPECT().FindById(ctx, "v2").Return(syncVersion("v2", syncSprites("0887", "0006")), nil)
		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{
			syncVersion("v2", syncSprites("0887", "0006")),
			syncVersion("v1", nil),
		}, nil)
		m.deckCode.EXPECT().Delete(ctx, "v2").Return(nil)

		require.NoError(t, m.usecase.Delete(ctx, "v2"))
	})

	t.Run("正常系_過去のバージョンを消してもデッキは変えない", func(t *testing.T) {
		m := setupSync(t, syncSprites("0887", "0006"))

		m.deckCode.EXPECT().FindById(ctx, "v1").Return(syncVersion("v1", syncSprites("0025")), nil)
		m.deckCode.EXPECT().FindByDeckId(ctx, syncDeckId).Return([]*entity.DeckCode{
			syncVersion("v2", syncSprites("0887", "0006")),
			syncVersion("v1", syncSprites("0025")),
		}, nil)
		m.deckCode.EXPECT().Delete(ctx, "v1").Return(nil)

		require.NoError(t, m.usecase.Delete(ctx, "v1"))
	})
}

func TestDeckSpriteSync_UpdateDeck(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T, deck *entity.Deck) (*mock_repository.MockDeckCodeInterface, DeckInterface) {
		mockCtrl := gomock.NewController(t)
		mockDeck := mock_repository.NewMockDeckInterface(mockCtrl)
		mockDeckCode := mock_repository.NewMockDeckCodeInterface(mockCtrl)

		mockDeck.EXPECT().FindById(ctx, deck.ID).Return(deck, nil)
		mockDeck.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		usecase := NewDeck(
			mockDeck,
			mock_repository.NewMockDeckAssetInterface(mockCtrl),
			mock_repository.NewMockUserFavoriteDeckInterface(mockCtrl),
			stubTagRepository{},
			stubTransactionManager{},
			stubBadgeEvaluation{},
			mock_repository.NewMockDeckCodePostInterface(mockCtrl),
			mockDeckCode,
		)

		return mockDeckCode, usecase
	}

	deckWithLatest := func(deckSprites, latestSprites []*entity.PokemonSprite) *entity.Deck {
		return &entity.Deck{
			ID:             syncDeckId,
			UserId:         syncUid,
			PokemonSprites: deckSprites,
			LatestDeckCode: syncVersion("v2", latestSprites),
		}
	}

	t.Run("正常系_デッキのスプライトを変えると設定済みの最新バージョンも同じになる", func(t *testing.T) {
		mockDeckCode, usecase := setup(t, deckWithLatest(syncSprites("0887", "0006"), syncSprites("0887", "0006")))

		mockDeckCode.EXPECT().ReplacePokemonSprites(ctx, "v2", syncSprites("0025", "0006")).Return(nil)

		ret, err := usecase.Update(ctx, syncDeckId, NewDeckUpdateParam("デッキ", false, syncSpriteParams("0025", "0006"), nil))
		require.NoError(t, err)
		require.True(t, samePokemonSprites(ret.PokemonSprites, ret.LatestDeckCode.PokemonSprites))
	})

	t.Run("正常系_デッキのスプライトを外すと設定済みの最新バージョンは未設定に戻る", func(t *testing.T) {
		mockDeckCode, usecase := setup(t, deckWithLatest(syncSprites("0887"), syncSprites("0887")))

		mockDeckCode.EXPECT().ReplacePokemonSprites(ctx, "v2", []*entity.PokemonSprite{}).Return(nil)

		_, err := usecase.Update(ctx, syncDeckId, NewDeckUpdateParam("デッキ", false, nil, nil))
		require.NoError(t, err)
	})

	t.Run("正常系_最新バージョンが未設定ならバージョンには書かない", func(t *testing.T) {
		_, usecase := setup(t, deckWithLatest(syncSprites("0887"), nil))

		_, err := usecase.Update(ctx, syncDeckId, NewDeckUpdateParam("デッキ", false, syncSpriteParams("0025"), nil))
		require.NoError(t, err)
	})

	t.Run("正常系_スプライトを変えない更新ではバージョンに書かない", func(t *testing.T) {
		_, usecase := setup(t, deckWithLatest(syncSprites("0887", "0006"), syncSprites("0887", "0006")))

		_, err := usecase.Update(ctx, syncDeckId, NewDeckUpdateParam("名前だけ変更", false, syncSpriteParams("0887", "0006"), nil))
		require.NoError(t, err)
	})
}

func TestDeckSpriteSync_CreateDeck(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) (*mock_repository.MockDeckCodeInterface, DeckInterface) {
		mockCtrl := gomock.NewController(t)
		mockDeck := mock_repository.NewMockDeckInterface(mockCtrl)
		mockDeckCode := mock_repository.NewMockDeckCodeInterface(mockCtrl)
		mockDeckAsset := mock_repository.NewMockDeckAssetInterface(mockCtrl)

		mockDeck.EXPECT().Save(ctx, gomock.Any()).Return(nil)
		mockDeckAsset.EXPECT().UploadDeckResultHTML(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockDeckAsset.EXPECT().UploadDeckImage(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		usecase := NewDeck(
			mockDeck,
			mockDeckAsset,
			mock_repository.NewMockUserFavoriteDeckInterface(mockCtrl),
			stubTagRepository{},
			stubTransactionManager{},
			stubBadgeEvaluation{},
			mock_repository.NewMockDeckCodePostInterface(mockCtrl),
			mockDeckCode,
		)

		return mockDeckCode, usecase
	}

	const deckCode = "5dbFbk-uBwjqP-VVk5Vv"

	t.Run("正常系_デッキ登録と同時に作る最初のバージョンにもデッキと同じスプライトを設定する", func(t *testing.T) {
		mockDeckCode, usecase := setup(t)

		mockDeckCode.EXPECT().ReplacePokemonSprites(ctx, gomock.Any(), syncSprites("0887", "0006")).Return(nil)

		ret, err := usecase.Create(ctx, NewDeckCreateParam(syncUid, "デッキ", false, deckCode, false, syncSpriteParams("0887", "0006"), nil))
		require.NoError(t, err)
		require.True(t, samePokemonSprites(ret.PokemonSprites, ret.LatestDeckCode.PokemonSprites))
		// デッキとバージョンでエンティティを共有しない
		require.NotSame(t, ret.PokemonSprites[0], ret.LatestDeckCode.PokemonSprites[0])
	})

	t.Run("正常系_スプライトの無いデッキではバージョンに書かない", func(t *testing.T) {
		_, usecase := setup(t)

		_, err := usecase.Create(ctx, NewDeckCreateParam(syncUid, "デッキ", false, deckCode, false, nil, nil))
		require.NoError(t, err)
	})

	t.Run("正常系_デッキコード無しの登録ではバージョンが無いので書かない", func(t *testing.T) {
		_, usecase := setup(t)

		_, err := usecase.Create(ctx, NewDeckCreateParam(syncUid, "デッキ", false, "", false, syncSpriteParams("0887"), nil))
		require.NoError(t, err)
	})
}

func TestSamePokemonSprites(t *testing.T) {
	require.True(t, samePokemonSprites(nil, []*entity.PokemonSprite{}))
	require.True(t, samePokemonSprites(syncSprites("0887", "0006"), syncSprites("0887", "0006")))
	// 枠が未指定(0)なら配列順として比べる
	require.True(t, samePokemonSprites(
		[]*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 0)},
		syncSprites("0887"),
	))
	require.False(t, samePokemonSprites(syncSprites("0887", "0006"), syncSprites("0006", "0887")))
	require.False(t, samePokemonSprites(syncSprites("0887"), syncSprites("0887", "0006")))
	// 同じスプライトでも枠が違えば別物(1枠目が空で2枠目だけ、など)
	require.False(t, samePokemonSprites(
		[]*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 2)},
		syncSprites("0887"),
	))
}
