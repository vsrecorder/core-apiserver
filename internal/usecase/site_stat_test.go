package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func setup4TestSiteStatUsecase(t *testing.T) (SiteStatInterface, *mock_repository.MockSiteStatInterface) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockSiteStatInterface(mockCtrl)

	return NewSiteStat(mockRepository), mockRepository
}

func TestSiteStatUsecase(t *testing.T) {
	t.Run("Find", func(t *testing.T) {
		now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.Local)

		t.Run("正常系_TTL内の2回目はDBを引かずに前回の結果を返す", func(t *testing.T) {
			usecase, mockRepository := setup4TestSiteStatUsecase(t)

			// Times(1): 2回目は保持した結果を返すので、リポジトリは1度しか呼ばれない
			mockRepository.EXPECT().Find(gomock.Any()).Return(entity.NewSiteStat(1, 2, 3), nil).Times(1)

			overrideTimeNow(t, now)
			first, err := usecase.Find(context.Background())
			require.NoError(t, err)

			overrideTimeNow(t, now.Add(siteStatCacheTTL-time.Second))
			second, err := usecase.Find(context.Background())
			require.NoError(t, err)

			require.Equal(t, first, second)
		})

		t.Run("正常系_TTLを過ぎたら引き直す", func(t *testing.T) {
			usecase, mockRepository := setup4TestSiteStatUsecase(t)

			gomock.InOrder(
				mockRepository.EXPECT().Find(gomock.Any()).Return(entity.NewSiteStat(1, 2, 3), nil),
				mockRepository.EXPECT().Find(gomock.Any()).Return(entity.NewSiteStat(4, 5, 6), nil),
			)

			overrideTimeNow(t, now)
			_, err := usecase.Find(context.Background())
			require.NoError(t, err)

			overrideTimeNow(t, now.Add(siteStatCacheTTL))
			stat, err := usecase.Find(context.Background())
			require.NoError(t, err)

			require.Equal(t, int64(4), stat.UserCount)
		})

		t.Run("異常系_失敗は保持せず次の呼び出しで引き直す", func(t *testing.T) {
			usecase, mockRepository := setup4TestSiteStatUsecase(t)

			gomock.InOrder(
				mockRepository.EXPECT().Find(gomock.Any()).Return(nil, errors.New("connection refused")),
				mockRepository.EXPECT().Find(gomock.Any()).Return(entity.NewSiteStat(1, 2, 3), nil),
			)

			overrideTimeNow(t, now)
			_, err := usecase.Find(context.Background())
			require.Error(t, err)

			stat, err := usecase.Find(context.Background())
			require.NoError(t, err)
			require.Equal(t, int64(1), stat.UserCount)
		})
	})
}
