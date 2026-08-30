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

// weeklyReportFixedNow は判定を安定させるための固定「現在時刻」。2026-08-24(月) 08:00。
// 配信対象は先週 2026-08-17(月)〜08-23(日) で、weekRange は月曜0時〜翌月曜0時を返す。
var weeklyReportFixedNow = time.Date(2026, 8, 24, 8, 0, 0, 0, time.Local)

const weeklyReportTestWeek = "2026-08-17"

type weeklyReportMocks struct {
	userStat      *mock_repository.MockUserStatInterface
	deckUsageStat *mock_repository.MockDeckUsageStatInterface
	notification  *mock_repository.MockNotificationInterface
	pushNotifier  *stubPushNotifier
}

func setup4WeeklyReportNotifier(t *testing.T) (
	*mock_repository.MockUserStatInterface,
	*mock_repository.MockDeckUsageStatInterface,
	*mock_repository.MockNotificationInterface,
	WeeklyReportNotifierInterface,
) {
	t.Helper()
	m, u := setup4WeeklyReportNotifierWithPush(t)

	return m.userStat, m.deckUsageStat, m.notification, u
}

// setup4WeeklyReportNotifierWithPush は push 配達器のモックまで返す。
// 通知を作成するケースでは Deliver が呼ばれたことを campaigns() で確かめる。
func setup4WeeklyReportNotifierWithPush(t *testing.T) (*weeklyReportMocks, WeeklyReportNotifierInterface) {
	t.Helper()
	overrideTimeNow(t, weeklyReportFixedNow)

	mockCtrl := gomock.NewController(t)
	m := &weeklyReportMocks{
		userStat:      mock_repository.NewMockUserStatInterface(mockCtrl),
		deckUsageStat: mock_repository.NewMockDeckUsageStatInterface(mockCtrl),
		notification:  mock_repository.NewMockNotificationInterface(mockCtrl),
		pushNotifier:  &stubPushNotifier{sent: 1},
	}

	return m, NewWeeklyReportNotifier(m.userStat, m.deckUsageStat, m.notification, m.pushNotifier)
}

func TestWeeklyReportNotifier_NotifyUser(t *testing.T) {
	fromDate := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)
	wantLink := "/users/report/weeks/2026-08-17"

	t.Run("正常系_1戦以上あればレポートへ誘導する通知を作成する", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		userStatRepo, deckUsageStatRepo, notificationRepo := m.userStat, m.deckUsageStat, m.notification

		stat := entity.NewUserStat("user-1", 4, 2, 1, 1, 10, 6, 4, 0.6)
		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryWeeklyReport, wantLink).Return(false, nil)
		// 使用回数が最多のデッキが相棒になる(並び順には依存しない)
		deckUsageStatRepo.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(
			entity.NewDeckUsageStat("user-1", 4, []*entity.DeckUsage{
				{DeckId: "deck-2", Name: "ドラパルトex", Count: 3},
				{DeckId: "deck-1", Name: " リザードンex ", Count: 7},
			}), nil)

		var saved *entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.True(t, sent)
		require.NotNil(t, saved)
		require.Equal(t, NotificationCategoryWeeklyReport, saved.Category)
		require.Equal(t, weeklyReportTitle, saved.Title)
		require.Equal(t, wantLink, saved.LinkUrl)
		require.Equal(t, weeklyReportFixedNow, saved.CreatedAt)
		require.Equal(t, "先週は10戦 6勝4敗（勝率 60.0%）。相棒デッキは『リザードンex』でした。", saved.Body)
		// 通知を作った上で push を撃つ(D2)
		require.Len(t, m.pushNotifier.calls, 1)
		require.Same(t, saved, m.pushNotifier.calls[0].notification)
		require.Equal(t, PushCampaignWeeklyReport, m.pushNotifier.calls[0].campaign)
	})

	t.Run("正常系_週内の任意日を渡しても月曜に正規化したリンクになる", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		userStatRepo, deckUsageStatRepo, notificationRepo := m.userStat, m.deckUsageStat, m.notification

		stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 3, 2, 1, 0.667)
		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryWeeklyReport, wantLink).Return(false, nil)
		deckUsageStatRepo.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(
			entity.NewDeckUsageStat("user-1", 1, []*entity.DeckUsage{}), nil)

		var saved *entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		)

		// 水曜を渡す
		sent, err := u.NotifyUser(context.Background(), "user-1", "2026-08-19", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, wantLink, saved.LinkUrl)
		// デッキが1つも無ければ相棒デッキの文は付かない
		require.Equal(t, "先週は3戦 2勝1敗（勝率 66.7%）。", saved.Body)
	})

	t.Run("正常系_引き分けがあれば本文の内訳に添える", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		userStatRepo, deckUsageStatRepo, notificationRepo := m.userStat, m.deckUsageStat, m.notification

		stat := entity.NewUserStat("user-1", 3, 3, 0, 0, 10, 6, 3, 0.6)
		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryWeeklyReport, wantLink).Return(false, nil)
		deckUsageStatRepo.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(
			entity.NewDeckUsageStat("user-1", 3, []*entity.DeckUsage{{DeckId: "deck-1", Name: "サーナイトex", Count: 10}}), nil)

		var saved *entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, "先週は10戦 6勝3敗1分（勝率 60.0%）。相棒デッキは『サーナイトex』でした。", saved.Body)
	})

	t.Run("正常系_デッキ集計に失敗しても相棒デッキ抜きで通知を出す", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		userStatRepo, deckUsageStatRepo, notificationRepo := m.userStat, m.deckUsageStat, m.notification

		stat := entity.NewUserStat("user-1", 2, 2, 0, 0, 5, 3, 2, 0.6)
		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryWeeklyReport, wantLink).Return(false, nil)
		deckUsageStatRepo.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(nil, errors.New("db down"))

		var saved *entity.Notification
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, "先週は5戦 3勝2敗（勝率 60.0%）。", saved.Body)
	})

	t.Run("正常系_dryRunなら対象判定だけ行い通知は作らない", func(t *testing.T) {
		userStatRepo, _, notificationRepo, u := setup4WeeklyReportNotifier(t)

		stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 2, 1, 1, 0.5)
		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryWeeklyReport, wantLink).Return(false, nil)
		// デッキ集計も保存も呼ばれない

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, true)

		require.NoError(t, err)
		require.True(t, sent)
	})

	t.Run("対象外_その週に0戦なら送らない", func(t *testing.T) {
		userStatRepo, _, _, u := setup4WeeklyReportNotifier(t)

		stat := entity.NewUserStat("user-1", 0, 0, 0, 0, 0, 0, 0, 0)
		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		// 二重送信判定にも進まない

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_戦績が存在しない(ErrRecordNotFound)なら送らない", func(t *testing.T) {
		userStatRepo, _, _, u := setup4WeeklyReportNotifier(t)

		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(nil, apperror.ErrRecordNotFound)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_同じ週のレポート通知が既にあれば送らない", func(t *testing.T) {
		userStatRepo, _, notificationRepo, u := setup4WeeklyReportNotifier(t)

		stat := entity.NewUserStat("user-1", 2, 2, 0, 0, 4, 3, 1, 0.75)
		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		// 同じ週(同じリンク先)のレポート通知が既にあれば送信済み(直近N件ではなく全期間で見る)
		notificationRepo.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryWeeklyReport, wantLink).Return(true, nil)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("異常系_weekの形式が不正ならエラーを返す", func(t *testing.T) {
		_, _, _, u := setup4WeeklyReportNotifier(t)

		sent, err := u.NotifyUser(context.Background(), "user-1", "2026/08/17", false)

		require.Error(t, err)
		require.False(t, sent)
	})

	t.Run("異常系_戦績集計のエラーはそのまま返す", func(t *testing.T) {
		userStatRepo, _, _, u := setup4WeeklyReportNotifier(t)

		userStatRepo.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(nil, errors.New("db down"))

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.Error(t, err)
		require.False(t, sent)
	})
}
