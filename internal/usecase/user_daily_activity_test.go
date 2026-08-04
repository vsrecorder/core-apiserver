package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestUserDailyActivity_Record(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 日付境界に近い時刻を固定し、時刻部分が切り落とされて当日の00:00になることを確認する。
	fixed := time.Date(2026, 8, 4, 23, 30, 0, 0, time.Local)
	today := time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)

	t.Run("正常系_カテゴリごとに当日の行を作る", func(t *testing.T) {
		overrideTimeNow(t, fixed)

		mockCtrl := gomock.NewController(t)
		repository := mock_repository.NewMockUserDailyActivityInterface(mockCtrl)
		u := NewUserDailyActivity(repository)

		var got []*entity.UserDailyActivity
		repository.EXPECT().Touch(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, entities []*entity.UserDailyActivity) error {
				got = entities
				return nil
			},
		)

		err := u.Record(context.Background(), uid, []string{
			entity.UserDailyActivityCategoryVisit,
			entity.UserDailyActivityCategoryReview,
		})

		require.NoError(t, err)
		require.Len(t, got, 2)

		require.Equal(t, uid, got[0].UserId)
		require.Equal(t, entity.UserDailyActivityCategoryVisit, got[0].Category)
		require.Equal(t, today, got[0].Date)
		require.Equal(t, fixed, got[0].UpdatedAt)

		require.Equal(t, entity.UserDailyActivityCategoryReview, got[1].Category)
		require.Equal(t, today, got[1].Date)
	})

	t.Run("正常系_未知のカテゴリは捨てて既知のぶんだけ記録する", func(t *testing.T) {
		overrideTimeNow(t, fixed)

		mockCtrl := gomock.NewController(t)
		repository := mock_repository.NewMockUserDailyActivityInterface(mockCtrl)
		u := NewUserDailyActivity(repository)

		var got []*entity.UserDailyActivity
		repository.EXPECT().Touch(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, entities []*entity.UserDailyActivity) error {
				got = entities
				return nil
			},
		)

		// webappが先にデプロイされ、APIサーバがまだ知らないカテゴリを送ってくる状況。
		// 全体を弾かず、既知のぶんは記録し続ける。
		err := u.Record(context.Background(), uid, []string{
			entity.UserDailyActivityCategoryVisit,
			"event",
		})

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, entity.UserDailyActivityCategoryVisit, got[0].Category)
	})

	t.Run("正常系_同一カテゴリの重複は1件に丸める", func(t *testing.T) {
		overrideTimeNow(t, fixed)

		mockCtrl := gomock.NewController(t)
		repository := mock_repository.NewMockUserDailyActivityInterface(mockCtrl)
		u := NewUserDailyActivity(repository)

		var got []*entity.UserDailyActivity
		repository.EXPECT().Touch(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, entities []*entity.UserDailyActivity) error {
				got = entities
				return nil
			},
		)

		err := u.Record(context.Background(), uid, []string{
			entity.UserDailyActivityCategoryVisit,
			entity.UserDailyActivityCategoryVisit,
		})

		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("異常系_既知のカテゴリが1つも無ければErrNoKnownActivityCategory", func(t *testing.T) {
		overrideTimeNow(t, fixed)

		mockCtrl := gomock.NewController(t)
		repository := mock_repository.NewMockUserDailyActivityInterface(mockCtrl)
		u := NewUserDailyActivity(repository)

		// 1件も記録しないので Touch は呼ばれない
		repository.EXPECT().Touch(gomock.Any(), gomock.Any()).Times(0)

		err := u.Record(context.Background(), uid, []string{"unknown"})

		require.ErrorIs(t, err, apperror.ErrNoKnownActivityCategory)
	})
}
