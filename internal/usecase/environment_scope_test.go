package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestBuildStatPeriod(t *testing.T) {
	// チャンピオンズリーグ2027横浜(2026-09-20〜22)は m6a(09-16〜)の期間に開催されるが、
	// カードプールは m6(ストームエメラルダ)まで、という実際のケースを模した値。
	m6 := entity.NewEnvironment(
		"m6", "ストームエメラルダ",
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local),
		time.Date(2026, 9, 15, 0, 0, 0, 0, time.Local),
	)
	clYokohama := []uint{1113193, 1113194}
	// 例外テーブルの全件(CL2027横浜の2件だけが m6 として登録されている状態)
	overrides := map[uint]string{1113193: "m6", 1113194: "m6"}

	t.Run("正常系_環境未指定なら渡された期間をそのまま使う", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		to := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)

		period, err := buildStatPeriod(t.Context(), environmentRepo, officialEventEnvironmentRepo, "", from, to)

		require.NoError(t, err)
		require.Equal(t, statPeriodOf(from, to), period)
	})

	t.Run("正常系_環境指定時は期間と例外イベントを持たせる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		environmentRepo.EXPECT().FindById(gomock.Any(), "m6").Return(m6, nil)
		officialEventEnvironmentRepo.EXPECT().FindAll(gomock.Any()).Return(overrides, nil)

		period, err := buildStatPeriod(t.Context(), environmentRepo, officialEventEnvironmentRepo, "m6", time.Time{}, time.Time{})

		require.NoError(t, err)
		// to_dateの翌日0時がexclusive上限
		require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), period.From)
		require.Equal(t, time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local), period.To)
		// 環境以外の条件が無いので、例外イベントを拾い直す範囲は無制限のまま。
		// ここに環境の期間を入れてしまうと、期間外の例外イベントを二度と拾えなくなる。
		require.True(t, period.BaseFrom.IsZero())
		require.True(t, period.BaseTo.IsZero())
		// 環境の期間外(9/20〜22)に開催されるが、m6として集計する
		require.Equal(t, clYokohama, period.IncludeOfficialEventIds)
		// m6として登録されたイベントを除外してはいけない
		require.Empty(t, period.ExcludeOfficialEventIds)
	})

	// 別の環境(m6a)を見るときは、同じイベントが逆に除外側へ回る。
	t.Run("正常系_他の環境に登録されたイベントは除外側に入る", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		m6a := entity.NewEnvironment(
			"m6a", "30th CELEBRATION",
			time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local),
			time.Date(2026, 11, 26, 0, 0, 0, 0, time.Local),
		)

		environmentRepo.EXPECT().FindById(gomock.Any(), "m6a").Return(m6a, nil)
		officialEventEnvironmentRepo.EXPECT().FindAll(gomock.Any()).Return(overrides, nil)

		period, err := buildStatPeriod(t.Context(), environmentRepo, officialEventEnvironmentRepo, "m6a", time.Time{}, time.Time{})

		require.NoError(t, err)
		require.Empty(t, period.IncludeOfficialEventIds)
		// 開催日は m6a の期間内だが、m6として登録されているので集計から外す
		require.Equal(t, clYokohama, period.ExcludeOfficialEventIds)
	})

	// 環境と他の条件は交差を取る仕様なので、例外イベントを拾い直す範囲(BaseFrom/BaseTo)は
	// 環境の期間で広げてはいけない。
	t.Run("正常系_他の条件と併用時は交差を取りBaseは他の条件のまま", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		environmentRepo.EXPECT().FindById(gomock.Any(), "m6").Return(m6, nil)
		officialEventEnvironmentRepo.EXPECT().FindAll(gomock.Any()).Return(overrides, nil)

		// year_month=2026-09 相当
		baseFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		baseTo := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)

		period, err := buildStatPeriod(t.Context(), environmentRepo, officialEventEnvironmentRepo, "m6", baseFrom, baseTo)

		require.NoError(t, err)
		// 交差: from=月初(9/1)、to=環境の終端翌日(9/16)
		require.Equal(t, baseFrom, period.From)
		require.Equal(t, time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local), period.To)
		require.Equal(t, baseFrom, period.BaseFrom)
		require.Equal(t, baseTo, period.BaseTo)
	})

	// 例外テーブルが引けないまま期間だけで集計すると、環境の数字が黙ってズレる。
	t.Run("異常系_例外テーブルの取得に失敗したらエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		environmentRepo.EXPECT().FindById(gomock.Any(), "m6").Return(m6, nil)
		officialEventEnvironmentRepo.EXPECT().FindAll(gomock.Any()).Return(nil, errors.New("db error"))

		period, err := buildStatPeriod(t.Context(), environmentRepo, officialEventEnvironmentRepo, "m6", time.Time{}, time.Time{})

		require.Error(t, err)
		require.Equal(t, repository.StatPeriod{}, period)
	})
}
