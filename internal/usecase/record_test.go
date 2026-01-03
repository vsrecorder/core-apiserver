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

func TestRecordUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	usecase := NewRecord(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockRecordInterface,
		usecase RecordInterface,
	){
		"Find":                  test_RecordUsecase_Find,
		"FindOnCursor":          test_RecordUsecase_FindOnCursor,
		"FindById":              test_RecordUsecase_FindById,
		"FindByUserId":          test_RecordUsecase_FindByUserId,
		"FindByUserIdOnCursor":  test_RecordUsecase_FindByUserIdOnCursor,
		"FindByOfficialEventId": test_RecordUsecase_FindByOfficialEventId,
		"FindByTonamelEventId":  test_RecordUsecase_FindByTonamelEventId,
		"FindByDeckId":          test_RecordUsecase_FindByDeckId,
		"Create":                test_RecordUsecase_Create,
		"Update":                test_RecordUsecase_Update,
		"Delete":                test_RecordUsecase_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_RecordUsecase_Find(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		limit := 10
		offset := 0

		id, err := generateId()
		require.NoError(t, err)

		record := &entity.Record{
			ID: id,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(records, nil)

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		limit := 10
		offset := 0

		records := []*entity.Record{}

		mockRepository.EXPECT().Find(context.Background(), limit, offset).Return(records, nil)

		ret, err := usecase.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
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

func test_RecordUsecase_FindOnCursor(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		id, err := generateId()
		require.NoError(t, err)

		record := &entity.Record{
			ID: id,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(records, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		limit := 10
		cursor := time.Now().Local()

		records := []*entity.Record{}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursor).Return(records, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
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

func test_RecordUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		record := &entity.Record{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

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

func test_RecordUsecase_FindByUserId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		offset := 0

		record := &entity.Record{
			ID:     id,
			UserId: uid,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(records, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		offset := 0

		records := []*entity.Record{}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(records, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, limit, offset)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		offset := 0

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.FindByUserId(context.Background(), uid, limit, offset)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByUserIdOnCursor(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		cursor := time.Now().Local()

		record := &entity.Record{
			ID:     id,
			UserId: uid,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, limit, cursor).Return(records, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
	})

	t.Run("正常系_#02", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		cursor := time.Now().Local()

		records := []*entity.Record{}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, limit, cursor).Return(records, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
	})

	t.Run("異常系_#01", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		cursor := time.Now().Local()

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, limit, cursor).Return(nil, errors.New(""))

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursor)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByOfficialEventId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		officialEventId := uint(10000)
		limit := 10
		offset := 0

		record := &entity.Record{
			ID:              id,
			OfficialEventId: officialEventId,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByOfficialEventId(context.Background(), officialEventId, limit, offset).Return(records, nil)

		ret, err := usecase.FindByOfficialEventId(context.Background(), officialEventId, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, officialEventId, ret[0].OfficialEventId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		officialEventId := uint(10000)
		limit := 10
		offset := 0

		mockRepository.EXPECT().FindByOfficialEventId(context.Background(), officialEventId, limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.FindByOfficialEventId(context.Background(), officialEventId, limit, offset)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByTonamelEventId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		tonamelEventId := "61ozP"
		limit := 10
		offset := 0

		record := &entity.Record{
			ID:             id,
			TonamelEventId: tonamelEventId,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset).Return(records, nil)

		ret, err := usecase.FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, tonamelEventId, ret[0].TonamelEventId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		tonamelEventId := "61ozP"
		limit := 10
		offset := 0

		mockRepository.EXPECT().FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByDeckId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deckId, err := generateId()
		require.NoError(t, err)

		limit := 10
		offset := 0

		record := &entity.Record{
			ID:     id,
			DeckId: deckId,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByDeckId(context.Background(), deckId, limit, offset).Return(records, nil)

		ret, err := usecase.FindByDeckId(context.Background(), deckId, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, deckId, ret[0].DeckId)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		deckId, err := generateId()
		require.NoError(t, err)

		limit := 10
		offset := 0

		mockRepository.EXPECT().FindByDeckId(context.Background(), deckId, limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.FindByDeckId(context.Background(), deckId, limit, offset)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_Create(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()

		record := entity.NewRecord(
			id,
			createdAt,
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.IsType(t, id, ret.ID)
		require.IsType(t, createdAt, ret.CreatedAt)
		require.Equal(t, record.OfficialEventId, ret.OfficialEventId)
		require.Equal(t, record.PrivateFlg, ret.PrivateFlg)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_Update(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()

		record := entity.NewRecord(
			id,
			createdAt,
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)
		mockRepository.EXPECT().Save(context.Background(), record).Return(nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.NoError(t, err)
		require.Equal(t, ret, record)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, gorm.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_#02", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()

		record := entity.NewRecord(
			id,
			createdAt,
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)
		mockRepository.EXPECT().Save(context.Background(), record).Return(errors.New(""))

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_Delete(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
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
