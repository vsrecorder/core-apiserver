package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// 環境バッジ獲得通知のcreated_atは、達成条件の判定に使うbasisTime(=対戦日/event_date)
// ではなく、match自体のCreatedAt(実際の処理時刻)を使う必要がある。event_dateを使うと、
// 過去日で記録した対戦の環境バッジ通知が、称号/ランクアップ通知や他のバッジ通知より
// 大幅に過去のcreated_atになり、通知一覧の並び順(created_at DESC)が崩れてしまうため。
func TestEnvironmentBadgeEvaluation_EvaluateOnMatchCreated_NotificationUsesMatchCreatedAt(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockEnvironmentRepo := mock_repository.NewMockEnvironmentInterface(mockCtrl)
	mockUserEnvironmentBadgeRepo := mock_repository.NewMockUserEnvironmentBadgeInterface(mockCtrl)
	mockNotificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
	mockTransactionManager := mock_repository.NewMockTransactionManager(mockCtrl)
	mockTransactionManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()

	usecase := NewEnvironmentBadgeEvaluation(
		mockEnvironmentRepo,
		mockUserEnvironmentBadgeRepo,
		mockNotificationRepo,
		mockTransactionManager,
	)

	userId := "user-1"
	env := entity.NewEnvironment("env-1", "アビスアイ", time.Time{}, time.Time{})

	// basisTime(対戦日/event_date)は過去日、match.CreatedAt(実際の処理時刻)は現在時刻とし、
	// 両者に差がある状況を再現する。
	basisTime := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	matchCreatedAt := time.Date(2026, 7, 8, 10, 53, 13, 49263000, time.Local)

	match := &entity.Match{
		ID:        "match-1",
		CreatedAt: matchCreatedAt,
		RecordId:  "record-1",
		UserId:    userId,
	}

	mockEnvironmentRepo.EXPECT().FindByDate(context.Background(), basisTime).Return(env, nil)
	mockUserEnvironmentBadgeRepo.EXPECT().FindByUserId(context.Background(), userId).Return(nil, nil)

	mockNotificationRepo.EXPECT().Save(context.Background(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, n *entity.Notification) error {
			require.True(t, n.CreatedAt.Equal(matchCreatedAt), "通知のcreated_atはmatch.CreatedAtを使うべき")
			require.False(t, n.CreatedAt.Equal(basisTime), "通知のcreated_atにbasisTime(event_date)を使ってはいけない")
			return nil
		},
	)
	mockUserEnvironmentBadgeRepo.EXPECT().Save(context.Background(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ub *entity.UserEnvironmentBadge) error {
			// 達成日時(achieved_at)自体はbasisTime(対戦日)基準のままでよい。
			require.True(t, ub.AchievedAt.Equal(basisTime))
			require.True(t, ub.CreatedAt.Equal(matchCreatedAt))
			return nil
		},
	)

	ret, err := usecase.EvaluateOnMatchCreated(context.Background(), userId, match, basisTime)

	require.NoError(t, err)
	require.Equal(t, env, ret)
}
