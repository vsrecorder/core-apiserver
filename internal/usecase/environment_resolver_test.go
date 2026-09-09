package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestResolveEnvironmentForOfficialEvent(t *testing.T) {
	// チャンピオンズリーグ2027横浜(2026-09-20)は30th CELEBRATION(m6a, 09-16〜)の発売後だが
	// カードプールはストームエメラルダ(m6)まで、という実際のケースを模した値。
	basisTime := time.Date(2026, 9, 20, 0, 0, 0, 0, time.Local)
	m6 := entity.NewEnvironment(
		"m6", "ストームエメラルダ",
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local),
		time.Date(2026, 9, 15, 0, 0, 0, 0, time.Local),
	)
	m6a := entity.NewEnvironment(
		"m6a", "30th CELEBRATION",
		time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local),
		time.Date(2026, 11, 26, 0, 0, 0, 0, time.Local),
	)

	t.Run("正常系_例外登録があれば開催日ではなく登録された環境を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		officialEventId := uint(1113193)
		officialEventEnvironmentRepo.EXPECT().FindEnvironmentIdByOfficialEventId(gomock.Any(), officialEventId).Return("m6", nil)
		environmentRepo.EXPECT().FindById(gomock.Any(), "m6").Return(m6, nil)

		env, err := ResolveEnvironmentForOfficialEvent(t.Context(), environmentRepo, officialEventEnvironmentRepo, officialEventId, basisTime)

		require.NoError(t, err)
		require.Equal(t, m6, env)
	})

	t.Run("正常系_例外登録が無ければ開催日が属する環境を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		officialEventId := uint(9999999)
		officialEventEnvironmentRepo.EXPECT().FindEnvironmentIdByOfficialEventId(gomock.Any(), officialEventId).Return("", apperror.ErrRecordNotFound)
		environmentRepo.EXPECT().FindByDate(gomock.Any(), basisTime).Return(m6a, nil)

		env, err := ResolveEnvironmentForOfficialEvent(t.Context(), environmentRepo, officialEventEnvironmentRepo, officialEventId, basisTime)

		require.NoError(t, err)
		require.Equal(t, m6a, env)
	})

	t.Run("正常系_公式イベントに紐づかない場合は例外テーブルを引かない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		environmentRepo.EXPECT().FindByDate(gomock.Any(), basisTime).Return(m6a, nil)

		env, err := ResolveEnvironmentForOfficialEvent(t.Context(), environmentRepo, officialEventEnvironmentRepo, 0, basisTime)

		require.NoError(t, err)
		require.Equal(t, m6a, env)
	})

	t.Run("異常系_例外テーブルの取得に失敗したら日付にフォールバックせずエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		environmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
		officialEventEnvironmentRepo := mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl)

		officialEventId := uint(1113193)
		officialEventEnvironmentRepo.EXPECT().FindEnvironmentIdByOfficialEventId(gomock.Any(), officialEventId).Return("", errors.New("db error"))

		_, err := ResolveEnvironmentForOfficialEvent(t.Context(), environmentRepo, officialEventEnvironmentRepo, officialEventId, basisTime)

		require.Error(t, err)
	})
}
