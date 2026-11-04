package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// periodReportFixedNow は判定を安定させるための固定「現在時刻」。2026-11-01(日) 12:00。
var periodReportFixedNow = time.Date(2026, 11, 1, 12, 0, 0, 0, time.Local)

type periodReportMocks struct {
	userStat                 *mock_repository.MockUserStatInterface
	deckUsageStat            *mock_repository.MockDeckUsageStatInterface
	notification             *mock_repository.MockNotificationInterface
	pushNotifier             *stubPushNotifier
	environment              *mock_repository.MockEnvironmentInterface
	officialEventEnvironment *mock_repository.MockOfficialEventEnvironmentInterface
}

func setup4PeriodReportNotifier(t *testing.T) (*periodReportMocks, PeriodReportNotifierInterface) {
	t.Helper()
	overrideTimeNow(t, periodReportFixedNow)

	mockCtrl := gomock.NewController(t)
	m := &periodReportMocks{
		userStat:                 mock_repository.NewMockUserStatInterface(mockCtrl),
		deckUsageStat:            mock_repository.NewMockDeckUsageStatInterface(mockCtrl),
		notification:             mock_repository.NewMockNotificationInterface(mockCtrl),
		pushNotifier:             &stubPushNotifier{sent: 1},
		environment:              mock_repository.NewMockEnvironmentInterface(mockCtrl),
		officialEventEnvironment: mock_repository.NewMockOfficialEventEnvironmentInterface(mockCtrl),
	}

	return m, NewPeriodReportNotifier(
		m.userStat, m.deckUsageStat, m.notification, m.pushNotifier, m.environment, m.officialEventEnvironment,
	)
}

// captureSave は Save に渡された通知を受け取る。
func captureSave(m *periodReportMocks) **entity.Notification {
	var saved *entity.Notification
	m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, n *entity.Notification) error {
			saved = n
			return nil
		},
	)

	return &saved
}

func TestPeriodReportNotifier_NotifyMonthly(t *testing.T) {
	period := statPeriodOf(
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local),
		time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local),
	)
	wantLink := "/users/report/2026-10"

	t.Run("正常系_1戦以上あれば月のレポートへ誘導する通知を作りpushを送る", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		stat := entity.NewUserStat("user-1", 8, 4, 0, 4, 42, 25, 16, 0.595)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(stat, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryMonthlyReport, wantLink).Return(false, nil)
		m.deckUsageStat.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", period, uint(0), true).Return(
			entity.NewDeckUsageStat("user-1", 2, []*entity.DeckUsage{
				{DeckId: "deck-1", Name: "ドラパルトex", Count: 12},
				{DeckId: "deck-2", Name: "メガルカリオex", Count: 30},
			}), nil)
		saved := captureSave(m)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-10", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.NotNil(t, *saved)
		require.Equal(t, NotificationCategoryMonthlyReport, (*saved).Category)
		require.Equal(t, "10月のバトルレポートができました", (*saved).Title)
		// 引き分けがある月は内訳に添える(42戦 = 25勝16敗1分)
		require.Equal(t, "10月は42戦 25勝16敗1分（勝率 59.5%）。相棒デッキは『メガルカリオex』でした。", (*saved).Body)
		require.Equal(t, wantLink, (*saved).LinkUrl)
		require.Equal(t, periodReportFixedNow, (*saved).CreatedAt)
		require.Equal(t, []string{PushCampaignMonthlyReport}, m.pushNotifier.campaigns())
	})

	t.Run("正常系_1月1日に送る12月のレポートは年をまたいでも月だけを書く", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		december := statPeriodOf(
			time.Date(2026, 12, 1, 0, 0, 0, 0, time.Local),
			time.Date(2027, 1, 1, 0, 0, 0, 0, time.Local),
		)
		stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 3, 2, 1, 0.667)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", december, uint(0), true).Return(stat, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryMonthlyReport, "/users/report/2026-12").Return(false, nil)
		m.deckUsageStat.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", december, uint(0), true).Return(nil, nil)
		saved := captureSave(m)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-12", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, "12月のバトルレポートができました", (*saved).Title)
		// デッキが取れなければ相棒デッキの文は付かない
		require.Equal(t, "12月は3戦 2勝1敗（勝率 66.7%）。", (*saved).Body)
	})

	t.Run("正常系_その月が0戦なら何も作らない", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		zero := entity.NewUserStat("user-1", 0, 0, 0, 0, 0, 0, 0, 0)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(zero, nil)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-10", false)

		require.NoError(t, err)
		require.False(t, sent)
		require.Empty(t, m.pushNotifier.calls)
	})

	t.Run("正常系_戦績が見つからなければ0戦と同じく何も作らない", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(nil, apperror.ErrRecordNotFound)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-10", false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("正常系_同じ月の通知が既にあれば作らない", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 3, 2, 1, 0.667)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(stat, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryMonthlyReport, wantLink).Return(true, nil)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-10", false)

		require.NoError(t, err)
		require.False(t, sent)
		require.Empty(t, m.pushNotifier.calls)
	})

	t.Run("正常系_dryRunなら対象かどうかだけを返し通知もpushも作らない", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 3, 2, 1, 0.667)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(stat, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryMonthlyReport, wantLink).Return(false, nil)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-10", true)

		require.NoError(t, err)
		require.True(t, sent)
		require.Empty(t, m.pushNotifier.calls)
	})

	t.Run("正常系_pushに失敗しても通知は作ったものとして扱う", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)
		m.pushNotifier.err = errors.New("push down")

		stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 3, 2, 1, 0.667)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(stat, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryMonthlyReport, wantLink).Return(false, nil)
		m.deckUsageStat.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", period, uint(0), true).Return(nil, nil)
		captureSave(m)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-10", false)

		require.NoError(t, err)
		require.True(t, sent)
	})

	t.Run("異常系_戦績の集計に失敗したらエラーを返す", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(nil, errors.New("db down"))

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-10", false)

		require.Error(t, err)
		require.False(t, sent)
	})

	t.Run("異常系_年月の形式が不正ならエラーを返す", func(t *testing.T) {
		_, u := setup4PeriodReportNotifier(t)

		sent, err := u.NotifyMonthly(context.Background(), "user-1", "2026-13", false)

		require.Error(t, err)
		require.False(t, sent)
	})
}

func TestPeriodReportNotifier_NotifyEnvironment(t *testing.T) {
	// GORM の DATE は UTC 0時で返る(暦日だけを見る)
	env := entity.NewEnvironment(
		"m6a",
		"30th CELEBRATION",
		time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC),
	)
	// 1001 はこの環境として数える大型大会、2002 は別の環境として数える大会
	overrides := map[uint]string{1001: "m6a", 2002: "m7"}
	period := repository.StatPeriod{
		From:                    time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local),
		To:                      time.Date(2026, 11, 27, 0, 0, 0, 0, time.Local),
		IncludeOfficialEventIds: []uint{1001},
		ExcludeOfficialEventIds: []uint{2002},
	}
	wantLink := "/users/report/environments/m6a"

	t.Run("正常系_例外イベントを含む環境の期間で集計し環境レポートへ誘導する", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		m.environment.EXPECT().FindById(gomock.Any(), "m6a").Return(env, nil).AnyTimes()
		m.officialEventEnvironment.EXPECT().FindAll(gomock.Any()).Return(overrides, nil)

		stat := entity.NewUserStat("user-1", 5, 5, 0, 0, 20, 12, 8, 0.6)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", period, uint(0), true).Return(stat, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryEnvironmentReport, wantLink).Return(false, nil)
		m.deckUsageStat.EXPECT().FindDeckUsageStat(gomock.Any(), "user-1", period, uint(0), true).Return(
			entity.NewDeckUsageStat("user-1", 1, []*entity.DeckUsage{{DeckId: "deck-1", Name: "ゲッコウガ・メガユキメノコ", Count: 20}}), nil)
		saved := captureSave(m)

		sent, err := u.NotifyEnvironment(context.Background(), "user-1", "m6a", false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, NotificationCategoryEnvironmentReport, (*saved).Category)
		require.Equal(t, "『30th CELEBRATION』環境のバトルレポートができました", (*saved).Title)
		require.Equal(t, "この環境では20戦 12勝8敗（勝率 60.0%）。相棒デッキは『ゲッコウガ・メガユキメノコ』でした。", (*saved).Body)
		require.Equal(t, wantLink, (*saved).LinkUrl)
		require.Equal(t, []string{PushCampaignEnvironmentReport}, m.pushNotifier.campaigns())
	})

	t.Run("正常系_環境の期間は1回のバッチで1度だけ組み立てる", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		m.environment.EXPECT().FindById(gomock.Any(), "m6a").Return(env, nil).AnyTimes()
		// 例外テーブルの全件読みは対象者が何人いても1回
		m.officialEventEnvironment.EXPECT().FindAll(gomock.Any()).Return(overrides, nil).Times(1)

		zero := entity.NewUserStat("", 0, 0, 0, 0, 0, 0, 0, 0)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), gomock.Any(), period, uint(0), true).Return(zero, nil).Times(3)

		for _, userId := range []string{"user-1", "user-2", "user-3"} {
			sent, err := u.NotifyEnvironment(context.Background(), userId, "m6a", false)
			require.NoError(t, err)
			require.False(t, sent)
		}
	})

	t.Run("異常系_環境が見つからなければエラーを返す", func(t *testing.T) {
		m, u := setup4PeriodReportNotifier(t)

		m.environment.EXPECT().FindById(gomock.Any(), "zzz").Return(nil, apperror.ErrRecordNotFound)

		sent, err := u.NotifyEnvironment(context.Background(), "user-1", "zzz", false)

		require.Error(t, err)
		require.False(t, sent)
	})
}

func TestReportBody(t *testing.T) {
	stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 3, 2, 1, 0.667)

	t.Run("デッキ名が長すぎて上限を超えるなら相棒デッキの文を落とす", func(t *testing.T) {
		body := reportBody("10月は", stat, strings.Repeat("あ", notificationBodyMaxLength))

		require.Equal(t, "10月は3戦 2勝1敗（勝率 66.7%）。", body)
		require.LessOrEqual(t, utf8.RuneCountInString(body), notificationBodyMaxLength)
	})
}
