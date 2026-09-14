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

func TestWeeklyDeckUsageStatUsecase_GetWeeklyDeckUsageStat(t *testing.T) {
	t.Run("正常系_週内の任意日から月曜始まりの週の期間で集計する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockWeeklyDeckUsageStatInterface(mockCtrl)
		usecase := NewWeeklyDeckUsageStat(mockRepository)

		// 2026-07-16は木曜。属する週の月曜は2026-07-13。
		fromDate := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)

		stat := &entity.WeeklyDeckUsageStat{}

		mockRepository.EXPECT().FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact).Return(stat, nil)

		ret, err := usecase.GetWeeklyDeckUsageStat(context.Background(), "2026-07-16", entity.DeckUsageGroupingExact)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_週未指定なら現在時刻が属する週の期間で集計する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockWeeklyDeckUsageStatInterface(mockCtrl)
		usecase := NewWeeklyDeckUsageStat(mockRepository)

		// 2026-07-19は日曜。属する週の月曜は2026-07-13。
		overrideTimeNow(t, time.Date(2026, 7, 19, 23, 0, 0, 0, time.Local))

		fromDate := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)

		stat := &entity.WeeklyDeckUsageStat{}

		mockRepository.EXPECT().FindWeeklyDeckUsageStat(context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact).Return(stat, nil)

		ret, err := usecase.GetWeeklyDeckUsageStat(context.Background(), "", entity.DeckUsageGroupingExact)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("正常系_集計単位をそのままリポジトリへ渡す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockWeeklyDeckUsageStatInterface(mockCtrl)
		usecase := NewWeeklyDeckUsageStat(mockRepository)

		fromDate := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)

		stat := &entity.WeeklyDeckUsageStat{}

		mockRepository.EXPECT().FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingFirstSprite,
		).Return(stat, nil)

		ret, err := usecase.GetWeeklyDeckUsageStat(
			context.Background(), "2026-07-16", entity.DeckUsageGroupingFirstSprite,
		)

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	// コントローラを経由しない呼び出し(バッチ等)で空が来ても既定で集計できるようにする。
	t.Run("正常系_集計単位が未指定なら組み合わせ一致で集計する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockWeeklyDeckUsageStatInterface(mockCtrl)
		usecase := NewWeeklyDeckUsageStat(mockRepository)

		fromDate := time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local)
		toDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local)

		stat := &entity.WeeklyDeckUsageStat{}

		mockRepository.EXPECT().FindWeeklyDeckUsageStat(
			context.Background(), fromDate, toDate, entity.DeckUsageGroupingExact,
		).Return(stat, nil)

		ret, err := usecase.GetWeeklyDeckUsageStat(context.Background(), "2026-07-16", "")

		require.NoError(t, err)
		require.Equal(t, stat, ret)
	})

	t.Run("異常系_週の形式が不正ならリポジトリを呼ばずエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockWeeklyDeckUsageStatInterface(mockCtrl)
		usecase := NewWeeklyDeckUsageStat(mockRepository)

		ret, err := usecase.GetWeeklyDeckUsageStat(context.Background(), "2026/07/16", entity.DeckUsageGroupingExact)

		require.Error(t, err)
		require.Nil(t, ret)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockWeeklyDeckUsageStatInterface(mockCtrl)
		usecase := NewWeeklyDeckUsageStat(mockRepository)

		mockRepository.EXPECT().FindWeeklyDeckUsageStat(context.Background(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		ret, err := usecase.GetWeeklyDeckUsageStat(context.Background(), "2026-07-16", entity.DeckUsageGroupingExact)

		require.Error(t, err)
		require.Nil(t, ret)
	})
}
