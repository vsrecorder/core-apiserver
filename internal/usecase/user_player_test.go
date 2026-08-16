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

type userPlayerUsecaseMocks struct {
	userPlayer         *mock_repository.MockUserPlayerInterface
	cityleagueResult   *mock_repository.MockCityleagueResultInterface
	championshipSeries *mock_repository.MockChampionshipSeriesInterface
}

func setup4UserPlayerUsecaseWithMocks(t *testing.T) (
	userPlayerUsecaseMocks,
	UserPlayerInterface,
) {
	mockCtrl := gomock.NewController(t)
	mocks := userPlayerUsecaseMocks{
		userPlayer:         mock_repository.NewMockUserPlayerInterface(mockCtrl),
		cityleagueResult:   mock_repository.NewMockCityleagueResultInterface(mockCtrl),
		championshipSeries: mock_repository.NewMockChampionshipSeriesInterface(mockCtrl),
	}
	mockTransactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	mockTransactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()

	usecase := NewUserPlayer(
		mocks.userPlayer,
		mocks.cityleagueResult,
		mocks.championshipSeries,
		mockTransactionManager,
	)

	return mocks, usecase
}

// 紐付けの取得・作成しか使わないテスト向けの短縮版。
func setup4UserPlayerUsecase(t *testing.T) (
	*mock_repository.MockUserPlayerInterface,
	UserPlayerInterface,
) {
	mocks, usecase := setup4UserPlayerUsecaseWithMocks(t)

	return mocks.userPlayer, usecase
}

func TestUserPlayerUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	playerId := "1234567890123456"

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("正常系_指定ユーザの紐付けを返す", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			userPlayer := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN2", time.Now().Local(), uid, playerId)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(userPlayer, nil)

			ret, err := usecase.FindByUserId(context.Background(), uid)

			require.NoError(t, err)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, playerId, ret.PlayerId)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.FindByUserId(context.Background(), uid)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_紐付けを作成する", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)
			mockRepository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId))

			require.NoError(t, err)
			require.NotEmpty(t, ret.ID)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, playerId, ret.PlayerId)
		})

		// 所有権の確認を行わない方針のため、他ユーザーが登録済みの player_id も登録できる。
		// 重複を禁止すると、先に登録した人が正しい持ち主とは限らない状態で
		// 正当な利用者を締め出してしまうため。
		t.Run("正常系_別ユーザに登録済みのプレイヤーIDでも作成できる", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)
			mockRepository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId))

			require.NoError(t, err)
			require.Equal(t, playerId, ret.PlayerId)
		})

		// 実在確認を行わないため、存在しない player_id でもそのまま保存される
		t.Run("正常系_実在しないプレイヤーIDでも作成できる", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)
			mockRepository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, "0000000000000000"))

			require.NoError(t, err)
			require.Equal(t, "0000000000000000", ret.PlayerId)
		})

		t.Run("正常系_1ヶ月経過後の変更は旧紐付けを削除してから作成する", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			// 2ヶ月前の紐付け(別のplayer_id)が存在する
			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local().AddDate(0, -2, 0), uid, "9999999999999999")

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)
			mockRepository.EXPECT().Delete(gomock.Any(), existing.ID).Return(nil)
			mockRepository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId))

			require.NoError(t, err)
			require.Equal(t, playerId, ret.PlayerId)
		})

		t.Run("正常系_同じプレイヤーIDなら変更不要として既存の紐付けを返す", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local(), uid, playerId)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId))

			require.NoError(t, err)
			require.Equal(t, existing, ret)
		})

		t.Run("異常系_紐付けから1ヶ月未満の変更はErrLockedを返す", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			// 直近に別のplayer_idを紐付けたばかり
			existing := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN1", time.Now().Local(), uid, "9999999999999999")

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(existing, nil)

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId))

			require.ErrorIs(t, err, apperror.ErrLocked)
			require.Nil(t, ret)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, usecase := setup4UserPlayerUsecase(t)

			mockRepository.EXPECT().FindByUserId(context.Background(), uid).Return(nil, errors.New(""))

			ret, err := usecase.Create(context.Background(), NewUserPlayerCreateParam(uid, playerId))

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindCityleagueResultsByUserId", func(t *testing.T) {
		t.Run("正常系_連携済みプレイヤーIDの入賞をシーズン期間で引く", func(t *testing.T) {
			overrideTimeNow(t, time.Date(2026, 8, 17, 12, 0, 0, 0, time.Local))

			mocks, usecase := setup4UserPlayerUsecaseWithMocks(t)

			userPlayer := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN2", time.Now().Local(), uid, playerId)
			championshipSeries := entity.NewChampionshipSeries(
				"series_2026",
				"チャンピオンシップシリーズ2026",
				time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
				time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local),
			)
			playerCityleagueResult := entity.NewPlayerCityleagueResult(
				"250607", 952749, 1,
				time.Date(2026, 6, 7, 0, 0, 0, 0, time.Local),
				1, 15, "gnnHHn-Vg3aWc-LHNnHH",
				"シティリーグ2026 シーズン4", "ポケモンカードステーション・渋谷", "東京都", "ニンジャスピナー",
			)

			mocks.userPlayer.EXPECT().FindByUserId(context.Background(), uid).Return(userPlayer, nil)
			mocks.championshipSeries.EXPECT().FindById(context.Background(), "series_2026").Return(championshipSeries, nil)
			// to_date は含む日付なので、翌日0時のexclusive上限に変換して渡す
			mocks.cityleagueResult.EXPECT().FindByPlayerId(
				context.Background(),
				playerId,
				time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
				time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local),
			).Return([]*entity.PlayerCityleagueResult{playerCityleagueResult}, nil)

			ret, err := usecase.FindCityleagueResultsByUserId(context.Background(), uid, "2026")

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, uint(952749), ret[0].OfficialEventId)
			require.Equal(t, uint(1), ret[0].Rank)
			require.Equal(t, "gnnHHn-Vg3aWc-LHNnHH", ret[0].DeckCode)
		})

		t.Run("正常系_入賞0件でも空スライスを返す", func(t *testing.T) {
			overrideTimeNow(t, time.Date(2026, 8, 17, 12, 0, 0, 0, time.Local))

			mocks, usecase := setup4UserPlayerUsecaseWithMocks(t)

			userPlayer := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN2", time.Now().Local(), uid, playerId)
			championshipSeries := entity.NewChampionshipSeries(
				"series_2026",
				"チャンピオンシップシリーズ2026",
				time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local),
				time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local),
			)

			mocks.userPlayer.EXPECT().FindByUserId(context.Background(), uid).Return(userPlayer, nil)
			mocks.championshipSeries.EXPECT().FindById(context.Background(), "series_2026").Return(championshipSeries, nil)
			mocks.cityleagueResult.EXPECT().FindByPlayerId(
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			).Return([]*entity.PlayerCityleagueResult{}, nil)

			ret, err := usecase.FindCityleagueResultsByUserId(context.Background(), uid, "2026")

			require.NoError(t, err)
			require.Empty(t, ret)
		})

		// 連携が無いユーザに他人の入賞を返さないよう、プレイヤーIDが引けない時点で打ち切る
		t.Run("異常系_紐付けが無ければErrRecordNotFoundを返す", func(t *testing.T) {
			mocks, usecase := setup4UserPlayerUsecaseWithMocks(t)

			mocks.userPlayer.EXPECT().FindByUserId(context.Background(), uid).Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.FindCityleagueResultsByUserId(context.Background(), uid, "2026")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
		})

		t.Run("異常系_存在しないシーズンならErrRecordNotFoundを返す", func(t *testing.T) {
			overrideTimeNow(t, time.Date(2026, 8, 17, 12, 0, 0, 0, time.Local))

			mocks, usecase := setup4UserPlayerUsecaseWithMocks(t)

			userPlayer := entity.NewUserPlayer("01HD7Y3K8D6FDHMHTZ2GT41TN2", time.Now().Local(), uid, playerId)

			mocks.userPlayer.EXPECT().FindByUserId(context.Background(), uid).Return(userPlayer, nil)
			mocks.championshipSeries.EXPECT().FindById(context.Background(), "series_1999").Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.FindCityleagueResultsByUserId(context.Background(), uid, "1999")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
		})
	})
}
