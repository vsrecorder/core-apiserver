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

func setup4UserPlayerUsecase(t *testing.T) (
	*mock_repository.MockUserPlayerInterface,
	UserPlayerInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockUserPlayerInterface(mockCtrl)
	mockTransactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	mockTransactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()

	usecase := NewUserPlayer(mockRepository, mockTransactionManager)

	return mockRepository, usecase
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
}
