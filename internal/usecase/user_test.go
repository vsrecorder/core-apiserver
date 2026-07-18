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

func TestUserUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	mockDeckRepository := mock_repository.NewMockDeckInterface(mockCtrl)
	mockDeckCodeRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)
	mockUserPlayerRepository := mock_repository.NewMockUserPlayerInterface(mockCtrl)
	mockTransactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	mockTransactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()
	usecase := NewUser(mockRepository, mockRecordRepository, mockDeckRepository, mockDeckCodeRepository, mockUserPlayerRepository, mockTransactionManager, stubBadgeEvaluation{})

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockUserInterface,
		mockRecordRepository *mock_repository.MockRecordInterface,
		mockDeckRepository *mock_repository.MockDeckInterface,
		mockDeckCodeRepository *mock_repository.MockDeckCodeInterface,
		mockUserPlayerRepository *mock_repository.MockUserPlayerInterface,
		usecase UserInterface,
	){
		"FindById": test_UserUsecase_FindById,
		"Create":   test_UserUsecase_Create,
		"Update":   test_UserUsecase_Update,
		"Delete":   test_UserUsecase_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, mockRecordRepository, mockDeckRepository, mockDeckCodeRepository, mockUserPlayerRepository, usecase)
		})
	}
}

func test_UserUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockUserInterface, _ *mock_repository.MockRecordInterface, _ *mock_repository.MockDeckInterface, _ *mock_repository.MockDeckCodeInterface, _ *mock_repository.MockUserPlayerInterface, usecase UserInterface) {
	t.Run("正常系_指定IDのユーザを返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		user := &entity.User{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
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

func test_UserUsecase_Create(t *testing.T, mockRepository *mock_repository.MockUserInterface, _ *mock_repository.MockRecordInterface, _ *mock_repository.MockDeckInterface, _ *mock_repository.MockDeckCodeInterface, _ *mock_repository.MockUserPlayerInterface, usecase UserInterface) {
	t.Run("正常系_未登録IDならユーザを作成する", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.IsType(t, id, ret.ID)
		require.IsType(t, createdAt, ret.CreatedAt)
		require.Equal(t, name, ret.Name)
		require.Equal(t, imageURL, ret.ImageURL)
	})

	t.Run("異常系_既存IDならErrAlreadyExistsを返す", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		user := &entity.User{
			ID:        id,
			CreatedAt: createdAt,
			Name:      name,
			ImageURL:  imageURL,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, apperror.ErrAlreadyExists)
		require.Empty(t, ret)
	})

	t.Run("異常系_存在確認でNotFound以外のエラーはそのまま返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_UserUsecase_Update(t *testing.T, mockRepository *mock_repository.MockUserInterface, _ *mock_repository.MockRecordInterface, _ *mock_repository.MockDeckInterface, _ *mock_repository.MockDeckCodeInterface, _ *mock_repository.MockUserPlayerInterface, usecase UserInterface) {
	t.Run("正常系_取得したユーザにパラメータを反映して保存する", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserUpdateParam(
			name,
			imageURL,
		)

		user := &entity.User{
			ID:        id,
			CreatedAt: createdAt,
			Name:      name,
			ImageURL:  imageURL,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(user, nil)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.NoError(t, err)
		require.IsType(t, id, ret.ID)
		require.IsType(t, createdAt, ret.CreatedAt)
		require.Equal(t, name, ret.Name)
		require.Equal(t, imageURL, ret.ImageURL)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserUpdateParam(
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, apperror.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_NotFound以外の取得エラーもそのまま返す", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserUpdateParam(
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_UserUsecase_Delete(t *testing.T, mockRepository *mock_repository.MockUserInterface, mockRecordRepository *mock_repository.MockRecordInterface, mockDeckRepository *mock_repository.MockDeckInterface, mockDeckCodeRepository *mock_repository.MockDeckCodeInterface, mockUserPlayerRepository *mock_repository.MockUserPlayerInterface, usecase UserInterface) {
	t.Run("正常系_プレイヤーIDの紐付けがある場合", func(t *testing.T) {
		id, _ := generateId()
		recordId, _ := generateId()
		deckId, _ := generateId()
		deckCodeId, _ := generateId()
		userPlayerId, _ := generateId()
		userPlayer := entity.NewUserPlayer(userPlayerId, time.Now().Local(), id, "1234567890123456")

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{recordId}, nil)
		mockRecordRepository.EXPECT().Delete(context.Background(), recordId).Return(nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{deckId}, nil)
		mockDeckRepository.EXPECT().Delete(context.Background(), deckId).Return(nil)
		mockDeckCodeRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{deckCodeId}, nil)
		mockDeckCodeRepository.EXPECT().Delete(context.Background(), deckCodeId).Return(nil)
		mockUserPlayerRepository.EXPECT().FindByUserId(context.Background(), id).Return(userPlayer, nil)
		mockUserPlayerRepository.EXPECT().Delete(context.Background(), userPlayerId).Return(nil)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		err := usecase.Delete(context.Background(), id)

		require.NoError(t, err)
	})

	t.Run("正常系_プレイヤーIDの紐付けがない場合", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckCodeRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockUserPlayerRepository.EXPECT().FindByUserId(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		err := usecase.Delete(context.Background(), id)

		require.NoError(t, err)
	})

	t.Run("異常系_対戦記録のID取得に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return(nil, errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_対戦記録の削除に失敗", func(t *testing.T) {
		id, _ := generateId()
		recordId, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{recordId}, nil)
		mockRecordRepository.EXPECT().Delete(context.Background(), recordId).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_デッキの削除に失敗", func(t *testing.T) {
		id, _ := generateId()
		deckId, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{deckId}, nil)
		mockDeckRepository.EXPECT().Delete(context.Background(), deckId).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_デッキコードのID取得に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckCodeRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return(nil, errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_デッキコードの削除に失敗", func(t *testing.T) {
		id, _ := generateId()
		deckCodeId, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckCodeRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{deckCodeId}, nil)
		mockDeckCodeRepository.EXPECT().Delete(context.Background(), deckCodeId).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_プレイヤーIDのID取得に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckCodeRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockUserPlayerRepository.EXPECT().FindByUserId(context.Background(), id).Return(nil, errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_プレイヤーIDの削除に失敗", func(t *testing.T) {
		id, _ := generateId()
		userPlayerId, _ := generateId()
		userPlayer := entity.NewUserPlayer(userPlayerId, time.Now().Local(), id, "1234567890123456")

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckCodeRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockUserPlayerRepository.EXPECT().FindByUserId(context.Background(), id).Return(userPlayer, nil)
		mockUserPlayerRepository.EXPECT().Delete(context.Background(), userPlayerId).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})

	t.Run("異常系_ユーザ本体の削除に失敗", func(t *testing.T) {
		id, _ := generateId()

		mockRecordRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockDeckCodeRepository.EXPECT().FindIdsByUserId(context.Background(), id).Return([]string{}, nil)
		mockUserPlayerRepository.EXPECT().FindByUserId(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Equal(t, err, errors.New(""))
	})
}
