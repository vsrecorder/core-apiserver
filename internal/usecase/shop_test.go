package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func setup4TestShopUsecase(t *testing.T) (ShopInterface, *mock_repository.MockShopInterface) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockShopInterface(mockCtrl)

	return NewShop(mockRepository), mockRepository
}

func TestShopUsecase(t *testing.T) {
	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_limit未指定なら既定件数で引く", func(t *testing.T) {
			usecase, mockRepository := setup4TestShopUsecase(t)

			mockRepository.EXPECT().Find(
				context.Background(), "町田", ShopSearchDefaultLimit,
			).Return([]*entity.Shop{}, nil)

			_, err := usecase.Find(context.Background(), "町田", 0)

			require.NoError(t, err)
		})

		t.Run("正常系_limitが上限を超えていれば上限まで切り詰める", func(t *testing.T) {
			usecase, mockRepository := setup4TestShopUsecase(t)

			mockRepository.EXPECT().Find(
				context.Background(), "カードショップ", ShopSearchMaxLimit,
			).Return([]*entity.Shop{}, nil)

			_, err := usecase.Find(context.Background(), "カードショップ", ShopSearchMaxLimit+1000)

			require.NoError(t, err)
		})

		t.Run("正常系_指定した条件をそのままリポジトリへ渡す", func(t *testing.T) {
			usecase, mockRepository := setup4TestShopUsecase(t)

			shops := []*entity.Shop{newTestShop(10317)}
			mockRepository.EXPECT().Find(
				context.Background(), "カードショップ", 20,
			).Return(shops, nil)

			ret, err := usecase.Find(context.Background(), "カードショップ", 20)

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, uint(10317), ret[0].ID)
		})
	})
}
