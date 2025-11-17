package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func TestDeckUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckInterface(mockCtrl)
	usecase := NewDeck(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockDeckInterface,
		usecase DeckInterface,
	){
		"Find":                 test_DeckUsecase_Find,
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
			fn(t, mockRepository, usecase)
		})
	}
}

func test_DeckUsecase_Find(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		limit := 10
		offset := 0

		id, err := generateId()
		require.NoError(t, err)

		deck := &entity.Deck{
			ID: id,
		}

		decks := []*entity.Deck{
			deck,
		}

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		limit := 10
		offset := 0

		decks := []*entity.Deck{}

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(decks, nil)

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, len(decks), len(ret))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		limit := 10
		offset := 0

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindOnCursor(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		id, err := generateId()
		require.NoError(t, err)

		deck := &entity.Deck{
			ID: id,
		}

		decks := []*entity.Deck{
			deck,
		}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(decks, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		decks := []*entity.Deck{}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(decks, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, len(decks), len(ret))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(nil, errors.New(""))

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deck := &entity.Deck{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_FindByUserId(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		offset := 0

		deck := &entity.Deck{
			ID:         id,
			ArchivedAt: time.Time{},
			UserId:     uid,
		}

		decks := []*entity.Deck{
			deck,
		}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, archivedFlg, limit, offset).Return(decks, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, archivedFlg, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Equal(t, time.Time{}, ret[0].ArchivedAt)
		require.Empty(t, ret[0].ArchivedAt)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedAt := time.Now().Local()
		archivedFlg := true
		limit := 10
		offset := 0

		deck := &entity.Deck{
			ID:         id,
			ArchivedAt: archivedAt,
			UserId:     uid,
		}

		decks := []*entity.Deck{
			deck,
		}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, archivedFlg, limit, offset).Return(decks, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, archivedFlg, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Equal(t, archivedAt, ret[0].ArchivedAt)
		require.NotEmpty(t, ret[0].ArchivedAt)
	})

	t.Run("正常系_#03", func(t *testing.T) {
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

	t.Run("異常系_#01", func(t *testing.T) {
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

func test_DeckUsecase_FindByUserIdOnCursor(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedFlg := false
		limit := 10
		cursor := time.Now().Local()

		deck := &entity.Deck{
			ID:         id,
			ArchivedAt: time.Time{},
			UserId:     uid,
		}

		decks := []*entity.Deck{
			deck,
		}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor).Return(decks, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Equal(t, time.Time{}, ret[0].ArchivedAt)
		require.Empty(t, ret[0].ArchivedAt)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		archivedAt := time.Now().Local()
		archivedFlg := true
		limit := 10
		cursor := time.Now().Local()

		deck := &entity.Deck{
			ID:         id,
			ArchivedAt: archivedAt,
			UserId:     uid,
		}

		decks := []*entity.Deck{
			deck,
		}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor).Return(decks, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, archivedFlg, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
		require.Equal(t, archivedAt, ret[0].ArchivedAt)
		require.NotEmpty(t, ret[0].ArchivedAt)
	})

	t.Run("正常系_#03", func(t *testing.T) {
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

	t.Run("異常系_#01", func(t *testing.T) {
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

func test_DeckUsecase_Create(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Time{}
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"",
			false,
		)

		param := NewDeckParam(
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.IsType(t, id, ret.ID)
		require.IsType(t, createdAt, ret.CreatedAt)
		require.Empty(t, ret.ArchivedAt)
		require.IsType(t, deck.UserId, ret.UserId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		param := NewDeckParam(
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Update(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Time{}
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"",
			false,
		)

		param := NewDeckParam(
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), deck).Return(nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.NoError(t, err)
		require.Equal(t, ret, deck)
		require.Empty(t, ret.ArchivedAt)
		require.Equal(t, time.Time{}, ret.ArchivedAt)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		param := NewDeckParam(
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, gorm.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Time{}
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"",
			false,
		)

		param := NewDeckParam(
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), deck).Return(errors.New(""))

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})

	t.Run("異常系_#03", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Time{}
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"5dbFbk-uBwjqP-VVk5Vv",
			false,
		)

		param := NewDeckParam(
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, errors.New("deck code change is not allowed"))
		require.Empty(t, ret)
	})

	t.Run("異常系_#04", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Time{}
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"5dbFbk-uBwjqP-VVk5Vv",
			false,
		)

		param := NewDeckParam(
			uid,
			"",
			"8cD84x-MmkrzS-D4cY8x",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, errors.New("deck code change is not allowed"))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Archive(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Time{}
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Archive(context.Background(), id)

		require.NoError(t, err)
		require.NotEmpty(t, ret.ArchivedAt)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		ret, err := usecase.Archive(context.Background(), id)

		require.Equal(t, err, gorm.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Time{}
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Archive(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Unarchive(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Now().Local()
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		archivedDeck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"",
			false,
		)

		deck := entity.NewDeck(
			id,
			createdAt,
			time.Time{},
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(archivedDeck, nil)
		mockRepository.EXPECT().Save(context.Background(), deck).Return(nil)

		ret, err := usecase.Unarchive(context.Background(), id)

		require.NoError(t, err)
		require.Empty(t, ret.ArchivedAt)
		require.Equal(t, time.Time{}, ret.ArchivedAt)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		ret, err := usecase.Unarchive(context.Background(), id)

		require.Equal(t, err, gorm.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		archivedAt := time.Now().Local()
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

		deck := entity.NewDeck(
			id,
			createdAt,
			archivedAt,
			uid,
			"",
			"",
			false,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(deck, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Unarchive(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_DeckUsecase_Delete(t *testing.T, mockRepository *mock_repository.MockDeckInterface, usecase DeckInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		err := usecase.Delete(context.Background(), id)

		require.NoError(t, err)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})
}
