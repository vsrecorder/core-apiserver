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

func TestUserUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserInterface(mockCtrl)
	usecase := NewUser(mockRepository)

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockUserInterface,
		usecase UserInterface,
	){
		"FindById": test_UserUsecase_FindById,
		"Create":   test_UserUsecase_Create,
		"Update":   test_UserUsecase_Update,
		"Delete":   test_UserUsecase_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_UserUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockUserInterface, usecase UserInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
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

	t.Run("異常系_#01", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Equal(t, err, errors.New(""))
		require.Empty(t, ret)
	})
}

func test_UserUsecase_Create(t *testing.T, mockRepository *mock_repository.MockUserInterface, usecase UserInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().UTC().Truncate(0)
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserCreateParam(
			id,
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.IsType(t, id, ret.ID)
		require.IsType(t, createdAt, ret.CreatedAt)
		require.Equal(t, name, ret.Name)
		require.Equal(t, imageURL, ret.ImageURL)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().UTC().Truncate(0)
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

		require.Equal(t, err, errors.New("already exists"))
		require.Empty(t, ret)
	})

	t.Run("異常系_#02", func(t *testing.T) {
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

func test_UserUsecase_Update(t *testing.T, mockRepository *mock_repository.MockUserInterface, usecase UserInterface) {
	t.Run("正常系_#01", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().UTC().Truncate(0)
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

	t.Run("異常系_#01", func(t *testing.T) {
		id, _ := generateId()
		name := "test"
		imageURL := "http://example.com/image.png"

		param := NewUserUpdateParam(
			name,
			imageURL,
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, gorm.ErrRecordNotFound)

		ret, err := usecase.Update(context.Background(), id, param)

		require.Equal(t, err, gorm.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_#01", func(t *testing.T) {
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

func test_UserUsecase_Delete(t *testing.T, mockRepository *mock_repository.MockUserInterface, usecase UserInterface) {
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
