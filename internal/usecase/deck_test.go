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

// errBadgeEvaluation はデッキ作成時の称号評価が失敗するケースを再現するスタブ。
// stubBadgeEvaluation(record_test.go)を埋め込み、EvaluateOnDeckCreatedだけを差し替える。
type errBadgeEvaluation struct {
	stubBadgeEvaluation
}

func (errBadgeEvaluation) EvaluateOnDeckCreated(
	ctx context.Context,
	userId string,
	deck *entity.Deck,
) ([]*entity.UserBadge, error) {
	return nil, errors.New("")
}

func TestDeckUsecase(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockDeckInterface,
		mockDeckAsset *mock_repository.MockDeckAssetInterface,
		usecase DeckInterface,
	){
		"Find":                 test_DeckUsecase_Find,
		"FindAll":              test_DeckUsecase_FindAll,
		"FindOnCursor":         test_DeckUsecase_FindOnCursor,
		"FindById":             test_DeckUsecase_FindById,
		"FindByUserId":         test_DeckUsecase_FindByUserId,
		"FindByUserIdOnCursor": test_DeckUsecase_FindByUserIdOnCursor,
		"Create":               test_DeckUsecase_Create,
		"Update":               test_DeckUsecase_Update,
		"Archive":              test_DeckUsecase_Archive,
		"Unarchive":            test_DeckUsecase_Unarchive,
		"Delete":               test_DeckUsecase_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockRepository := mock_repository.NewMockDeckInterface(mockCtrl)
			mockDeckAsset := mock_repository.NewMockDeckAssetInterface(mockCtrl)
			usecase := NewDeck(mockRepository, mockDeckAsset, stubBadgeEvaluation{})

			fn(t, mockRepository, mockDeckAsset, usecase)
		})
	}
}

func test_DeckUsecase_Find(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	t.Run("正常系_デッキ一覧をそのまま返す", func(t *testing.T) {
		limit := 10
		offset := 0

		id, err := generateId()
		require.NoError(t, err)

		decks := []*entity.Deck{
			{ID: id},
		}

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		limit := 10
		offset := 0

		decks := []*entity.Deck{}

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, len(decks), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		limit := 10
		offset := 0

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindAll(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	t.Run("正常系_指定ユーザの全デッキを返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		id, err := generateId()
		require.NoError(t, err)

		decks := []*entity.Deck{
			{ID: id, UserId: uid},
		}

		mockRepository.EXPECT().FindAll(context.Background(), uid).Return(decks, nil)

		ret, err := usecase.FindAll(context.Background(), uid)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		decks := []*entity.Deck{}

		mockRepository.EXPECT().FindAll(context.Background(), uid).Return(decks, nil)

		ret, err := usecase.FindAll(context.Background(), uid)

		require.NoError(t, err)
		require.Equal(t, len(decks), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		mockRepository.EXPECT().FindAll(context.Background(), uid).Return(nil, errors.New(""))

		ret, err := usecase.FindAll(context.Background(), uid)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindOnCursor(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	t.Run("正常系_カーソル以降のデッキ一覧を返す", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		id, err := generateId()
		require.NoError(t, err)

		decks := []*entity.Deck{
			{ID: id},
		}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(decks, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		decks := []*entity.Deck{}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(decks, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, len(decks), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(nil, errors.New(""))

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindById(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	t.Run("正常系_指定IDのデッキを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deck := &entity.Deck{ID: id}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, apperror.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindByUserId(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	t.Run("正常系_未アーカイブのデッキ一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		offset := 0

		decks := []*entity.Deck{
			{ID: id, ArchivedAt: time.Time{}, UserId: uid},
		}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, archivedFlg, limit, offset).Return(decks, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, archivedFlg, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Empty(t, ret[0].ArchivedAt)
	})

	t.Run("正常系_アーカイブ済み指定でアーカイブ日時付きのデッキを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedAt := time.Now().Local()
		archivedFlg := true
		limit := 10
		offset := 0

		decks := []*entity.Deck{
			{ID: id, ArchivedAt: archivedAt, UserId: uid},
		}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, archivedFlg, limit, offset).Return(decks, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, archivedFlg, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Equal(t, archivedAt, ret[0].ArchivedAt)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		offset := 0

		decks := []*entity.Deck{}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, archivedFlg, limit, offset).Return(decks, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, archivedFlg, limit, offset)

		require.NoError(t, err)
		require.Equal(t, len(decks), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		offset := 0

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, archivedFlg, limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.FindByUserId(context.Background(), uid, archivedFlg, limit, offset)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindByUserIdOnCursor(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	t.Run("正常系_カーソル以降の未アーカイブデッキ一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		cursor := time.Now().Local()

		decks := []*entity.Deck{
			{ID: id, ArchivedAt: time.Time{}, UserId: uid},
		}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor).Return(decks, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Empty(t, ret[0].ArchivedAt)
	})

	t.Run("正常系_アーカイブ済み指定でアーカイブ日時付きのデッキを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedAt := time.Now().Local()
		archivedFlg := true
		limit := 10
		cursor := time.Now().Local()

		decks := []*entity.Deck{
			{ID: id, ArchivedAt: archivedAt, UserId: uid},
		}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor).Return(decks, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Equal(t, archivedAt, ret[0].ArchivedAt)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		cursor := time.Now().Local()

		decks := []*entity.Deck{}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor).Return(decks, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, len(decks), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		cursor := time.Now().Local()

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor).Return(nil, errors.New(""))

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Create(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	deckCode := "5dbFbk-uBwjqP-VVk5Vv"

	// デッキコードを指定しない場合、外部リソース(HTML・画像)のアップロードは行われない
	t.Run("正常系_デッキコード未指定なら外部アップロードなしで保存される", func(t *testing.T) {
		param := NewDeckCreateParam(uid, "テストデッキ", false, "", false, nil)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.NotEmpty(t, ret.ID)
		require.NotEmpty(t, ret.CreatedAt)
		require.Empty(t, ret.ArchivedAt)
		require.Equal(t, uid, ret.UserId)
		require.Equal(t, "テストデッキ", ret.Name)
		require.False(t, ret.PrivateFlg)
		require.Empty(t, ret.LatestDeckCode.Code)
		require.Empty(t, ret.PokemonSprites)
	})

	// デッキコードを指定した場合、HTML→画像の順にアップロードした上でデッキが保存される
	t.Run("正常系_デッキコード指定時はHTMLと画像をアップロードして保存される", func(t *testing.T) {
		param := NewDeckCreateParam(uid, "テストデッキ", true, deckCode, true, nil)

		gomock.InOrder(
			mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), deckCode).Return(nil),
			mockDeckAsset.EXPECT().UploadDeckImage(context.Background(), deckCode).Return(nil),
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil),
		)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.True(t, ret.PrivateFlg)
		require.NotEmpty(t, ret.LatestDeckCode.ID)
		require.Equal(t, deckCode, ret.LatestDeckCode.Code)
		require.Equal(t, ret.ID, ret.LatestDeckCode.DeckId)
		require.Equal(t, uid, ret.LatestDeckCode.UserId)
		require.True(t, ret.LatestDeckCode.PrivateCodeFlg)
	})

	// 指定されたポケモンスプライトがエンティティへ引き継がれる
	t.Run("正常系_指定したポケモンスプライトがエンティティへ引き継がれる", func(t *testing.T) {
		pokemonSprites := []*PokemonSpriteParam{
			NewPokemonSpriteParam("pikachu"),
			NewPokemonSpriteParam("raichu"),
		}
		param := NewDeckCreateParam(uid, "テストデッキ", false, "", false, pokemonSprites)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.Len(t, ret.PokemonSprites, 2)
		require.Equal(t, "pikachu", ret.PokemonSprites[0].ID)
		require.Equal(t, "raichu", ret.PokemonSprites[1].ID)
	})

	// デッキコードのHTMLアップロードに失敗した場合(=不正なデッキコード)、
	// 画像アップロードもデッキ保存も行わずに中止する
	t.Run("異常系_HTMLアップロード失敗時は画像アップロードも保存も行わない", func(t *testing.T) {
		param := NewDeckCreateParam(uid, "テストデッキ", false, deckCode, false, nil)

		mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), deckCode).Return(errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	// デッキ画像のアップロードに失敗した場合もデッキ保存は行わずに中止する
	t.Run("異常系_画像アップロード失敗時は保存を行わない", func(t *testing.T) {
		param := NewDeckCreateParam(uid, "テストデッキ", false, deckCode, false, nil)

		gomock.InOrder(
			mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), deckCode).Return(nil),
			mockDeckAsset.EXPECT().UploadDeckImage(context.Background(), deckCode).Return(errors.New("")),
		)

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
		param := NewDeckCreateParam(uid, "テストデッキ", false, "", false, nil)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	// 称号評価に失敗した場合はエラーを返す
	t.Run("異常系_称号評価失敗時はエラーを返す", func(t *testing.T) {
		usecase := NewDeck(mockRepository, mockDeckAsset, errBadgeEvaluation{})
		param := NewDeckCreateParam(uid, "テストデッキ", false, "", false, nil)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Update(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 名前・公開設定・ポケモンスプライトのみが更新され、
	// ID・作成日時・ユーザID・デッキコードは更新前の値が引き継がれる
	t.Run("正常系_名前と公開設定とスプライトのみ更新され他は維持される", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Local()
		latestDeckCode := entity.NewDeckCode("01HD7Y3K8D6FDHMHTZ2GT41TN2", createdAt, uid, id, "5dbFbk-uBwjqP-VVk5Vv", false, "")

		deck := entity.NewDeck(
			id,
			createdAt,
			time.Time{},
			uid,
			"更新前のデッキ",
			false,
			latestDeckCode,
			[]*entity.PokemonSprite{entity.NewPokemonSprite("pikachu")},
		)

		param := NewDeckUpdateParam("更新後のデッキ", true, []*PokemonSpriteParam{NewPokemonSpriteParam("raichu")})

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
		require.Equal(t, createdAt, ret.CreatedAt)
		require.Equal(t, uid, ret.UserId)
		require.Equal(t, "更新後のデッキ", ret.Name)
		require.True(t, ret.PrivateFlg)
		require.Equal(t, latestDeckCode, ret.LatestDeckCode)
		require.Len(t, ret.PokemonSprites, 1)
		require.Equal(t, "raichu", ret.PokemonSprites[0].ID)
		require.Empty(t, ret.ArchivedAt)
	})

	// アーカイブ済みのデッキを更新してもアーカイブ日時は維持される
	t.Run("正常系_アーカイブ済みデッキを更新してもアーカイブ日時は維持される", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Local()
		archivedAt := time.Now().Local()

		deck := entity.NewDeck(id, createdAt, archivedAt, uid, "更新前のデッキ", false, nil, nil)
		param := NewDeckUpdateParam("更新後のデッキ", false, nil)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.NoError(t, err)
		require.Equal(t, archivedAt, ret.ArchivedAt)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		param := NewDeckUpdateParam("更新後のデッキ", false, nil)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, apperror.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deck := entity.NewDeck(id, time.Now().Local(), time.Time{}, uid, "更新前のデッキ", false, nil, nil)
		param := NewDeckUpdateParam("更新後のデッキ", false, nil)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Archive(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// アーカイブ日時が設定され、その他の値は維持される
	t.Run("正常系_アーカイブ日時が設定され他の値は維持される", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Local()
		deck := entity.NewDeck(id, createdAt, time.Time{}, uid, "テストデッキ", false, nil, nil)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Archive(context.Background(), id)

		require.NoError(t, err)
		require.NotEmpty(t, ret.ArchivedAt)
		require.Equal(t, id, ret.ID)
		require.Equal(t, createdAt, ret.CreatedAt)
		require.Equal(t, "テストデッキ", ret.Name)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Archive(context.Background(), id)

		require.Equal(t, err, apperror.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deck := entity.NewDeck(id, time.Now().Local(), time.Time{}, uid, "テストデッキ", false, nil, nil)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Archive(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	// DB接続エラーなど ErrRecordNotFound 以外のエラーもそのまま返す
	// (取得できていないDeckを参照してnilパニックを起こさないこと)
	t.Run("異常系_NotFound以外の取得エラーもそのまま返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.Archive(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Unarchive(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// アーカイブ日時がクリアされ、その他の値は維持される
	t.Run("正常系_アーカイブ日時がクリアされ他の値は維持される", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		createdAt := time.Now().Local()
		archivedDeck := entity.NewDeck(id, createdAt, time.Now().Local(), uid, "テストデッキ", false, nil, nil)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(archivedDeck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Unarchive(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, time.Time{}, ret.ArchivedAt)
		require.Equal(t, id, ret.ID)
		require.Equal(t, createdAt, ret.CreatedAt)
		require.Equal(t, "テストデッキ", ret.Name)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Unarchive(context.Background(), id)

		require.Equal(t, err, apperror.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deck := entity.NewDeck(id, time.Now().Local(), time.Now().Local(), uid, "テストデッキ", false, nil, nil)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Unarchive(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	// DB接続エラーなど ErrRecordNotFound 以外のエラーもそのまま返す
	// (取得できていないDeckを参照してnilパニックを起こさないこと)
	t.Run("異常系_NotFound以外の取得エラーもそのまま返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.Unarchive(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Delete(
	t *testing.T,
	mockRepository *mock_repository.MockDeckInterface,
	mockDeckAsset *mock_repository.MockDeckAssetInterface,
	usecase DeckInterface,
) {
	t.Run("正常系_リポジトリのDeleteを呼び出す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		require.NoError(t, usecase.Delete(context.Background(), id))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		require.Equal(t, usecase.Delete(context.Background(), id), errors.New(""))
	})
}
