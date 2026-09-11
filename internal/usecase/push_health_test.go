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

func TestPushHealth(t *testing.T) {
	since := time.Date(2026, 9, 4, 0, 0, 0, 0, time.Local)

	setup := func(t *testing.T, stats []*entity.PushHealthStat, err error) PushHealthInterface {
		t.Helper()

		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		repo.EXPECT().AggregateHealthByPlatformSince(gomock.Any(), since).Return(stats, err)

		return NewPushHealth(repo)
	}

	// 全体の成功率で見ると Android の成功に薄まって見えるため、platform 別に判定する。
	// 実際に iOS だけ11日間全滅していたのを取り逃がした
	t.Run("正常系_成功が1件も無いplatformはcriticalとして返す", func(t *testing.T) {
		u := setup(t, []*entity.PushHealthStat{
			{Platform: entity.PushPlatformAndroid, Total: 10, Sent: 10},
			{Platform: entity.PushPlatformIOSPWA, Total: 7, Sent: 0, TopFailureStatusCode: 403},
		}, nil)

		_, problems, err := u.Check(context.Background(), since)

		require.NoError(t, err)
		require.Len(t, problems, 1)
		require.Equal(t, entity.PushPlatformIOSPWA, problems[0].Stat.Platform)
		require.Equal(t, PushHealthSeverityCritical, problems[0].Severity)
		require.Equal(t, 403, problems[0].Stat.TopFailureStatusCode)
	})

	t.Run("正常系_成功率が半分を切ったらwarningとして返す", func(t *testing.T) {
		u := setup(t, []*entity.PushHealthStat{
			{Platform: entity.PushPlatformAndroid, Total: 10, Sent: 4, TopFailureStatusCode: 410},
		}, nil)

		_, problems, err := u.Check(context.Background(), since)

		require.NoError(t, err)
		require.Len(t, problems, 1)
		require.Equal(t, PushHealthSeverityWarning, problems[0].Severity)
	})

	t.Run("正常系_全て成功していれば異常は無い", func(t *testing.T) {
		u := setup(t, []*entity.PushHealthStat{
			{Platform: entity.PushPlatformAndroid, Total: 10, Sent: 10},
			{Platform: entity.PushPlatformIOSPWA, Total: 5, Sent: 5},
		}, nil)

		stats, problems, err := u.Check(context.Background(), since)

		require.NoError(t, err)
		require.Empty(t, problems)
		require.Len(t, stats, 2)
	})

	// 新しい platform や、たまたま1件失敗しただけの端末で通知が飛ぶと、
	// 本当に壊れたときの通知が埋もれる
	t.Run("正常系_配達数が少ないうちは判定しない", func(t *testing.T) {
		u := setup(t, []*entity.PushHealthStat{
			{Platform: entity.PushPlatformIOSPWA, Total: pushHealthMinDeliveries - 1, Sent: 0, TopFailureStatusCode: 403},
		}, nil)

		_, problems, err := u.Check(context.Background(), since)

		require.NoError(t, err)
		require.Empty(t, problems)
	})

	t.Run("異常系_集計に失敗したらエラーを返す", func(t *testing.T) {
		u := setup(t, nil, errors.New("db down"))

		_, _, err := u.Check(context.Background(), since)

		require.Error(t, err)
	})
}
