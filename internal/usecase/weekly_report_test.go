package usecase

import (
	"context"
	"errors"
	"strings"
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
// 今週の月曜 08-24 は isEvenWeek で奇数週(隔週ガードのテストで使う)。
var weeklyReportFixedNow = time.Date(2026, 8, 24, 8, 0, 0, 0, time.Local)

const weeklyReportTestWeek = "2026-08-17"

type weeklyReportMocks struct {
	userStat            *mock_repository.MockUserStatInterface
	deckUsageStat       *mock_repository.MockDeckUsageStatInterface
	notification        *mock_repository.MockNotificationInterface
	pushNotifier        *stubPushNotifier
	pushSubscription    *mock_repository.MockPushSubscriptionInterface
	pushDelivery        *mock_repository.MockPushDeliveryInterface
	userStreak          *mock_repository.MockUserStreakInterface
	weeklyDeckUsageStat *mock_repository.MockWeeklyDeckUsageStatInterface
	pokemonSpriteName   *mock_repository.MockPokemonSpriteNameInterface
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

// setup4WeeklyReportNotifierWithPush はモック一式を返す。
// 通知を作成するケースでは Deliver が呼ばれたことを campaigns() で確かめる。
func setup4WeeklyReportNotifierWithPush(t *testing.T) (*weeklyReportMocks, WeeklyReportNotifierInterface) {
	t.Helper()
	overrideTimeNow(t, weeklyReportFixedNow)

	mockCtrl := gomock.NewController(t)
	m := &weeklyReportMocks{
		userStat:            mock_repository.NewMockUserStatInterface(mockCtrl),
		deckUsageStat:       mock_repository.NewMockDeckUsageStatInterface(mockCtrl),
		notification:        mock_repository.NewMockNotificationInterface(mockCtrl),
		pushNotifier:        &stubPushNotifier{sent: 1},
		pushSubscription:    mock_repository.NewMockPushSubscriptionInterface(mockCtrl),
		pushDelivery:        mock_repository.NewMockPushDeliveryInterface(mockCtrl),
		userStreak:          mock_repository.NewMockUserStreakInterface(mockCtrl),
		weeklyDeckUsageStat: mock_repository.NewMockWeeklyDeckUsageStatInterface(mockCtrl),
		pokemonSpriteName:   mock_repository.NewMockPokemonSpriteNameInterface(mockCtrl),
	}

	return m, NewWeeklyReportNotifier(
		m.userStat, m.deckUsageStat, m.notification, m.pushNotifier,
		m.pushSubscription, m.pushDelivery, m.userStreak, m.weeklyDeckUsageStat, m.pokemonSpriteName,
	)
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
		userStatRepo, deckUsageStatRepo, notificationRepo, u := setup4WeeklyReportNotifier(t)

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
		userStatRepo, deckUsageStatRepo, notificationRepo, u := setup4WeeklyReportNotifier(t)

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
		userStatRepo, deckUsageStatRepo, notificationRepo, u := setup4WeeklyReportNotifier(t)

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
		m, u := setup4WeeklyReportNotifierWithPush(t)

		stat := entity.NewUserStat("user-1", 1, 1, 0, 0, 2, 1, 1, 0.5)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(stat, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryWeeklyReport, wantLink).Return(false, nil)
		// デッキ集計も保存も呼ばれない

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, true)

		require.NoError(t, err)
		require.True(t, sent)
		require.Empty(t, m.pushNotifier.campaigns())
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

// ---- 記録ゼロの週: 環境ニュース ----

func sprites(ids ...string) []*entity.PokemonSprite {
	out := make([]*entity.PokemonSprite, 0, len(ids))
	for i, id := range ids {
		out = append(out, entity.NewPokemonSpriteWithPosition(id, uint(i+1)))
	}
	return out
}

func variant(fingerprint string, rate float64, count int, prevRate *float64, ids ...string) *entity.DeckUsageVariant {
	v := entity.NewDeckUsageVariant(fingerprint, count, rate, 0, 0, 0, sprites(ids...))
	v.PreviousUsageRate = prevRate
	return v
}

func ptr(f float64) *float64 { return &f }

// testEnvStat は 使用率1位 = ドラパルト＋ヨノワール(6.1%・前週6.4%)、
// いちばん伸びた = メガレックウザ＋ホウオウ(5.1%・前週1.2% → +3.9pt)、「その他」行付き。
func testEnvStat() *entity.WeeklyDeckUsageStat {
	return entity.NewWeeklyDeckUsageStat(
		time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), 690, 42,
		[]*entity.DeckUsageVariant{
			variant("fp-dra", 0.061, 42, ptr(0.064), "dragapult", "dusknoir"),
			variant("fp-ray", 0.051, 35, ptr(0.012), "rayquaza-mega", "ho-oh"),
			variant("fp-new", 0.040, 27, nil, "absol-mega", "kangaskhan-mega"), // 前週データ無し(新登場)
			variant("", 0.30, 200, ptr(0.31)),                                     // その他
		},
	)
}

var testSpriteNames = map[string]string{
	"dragapult": "ドラパルト", "dusknoir": "ヨノワール",
	"rayquaza-mega": "メガレックウザ", "ho-oh": "ホウオウ",
	"absol-mega": "メガアブソル", "kangaskhan-mega": "メガガルーラ",
}

func TestWeeklyReportNotifier_NotifyUser_EnvNews(t *testing.T) {
	fromDate := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)
	toDate := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)
	thisMonday := time.Date(2026, 8, 24, 0, 0, 0, 0, time.Local)
	wantLink := "/deck_meta?week=2026-08-17"
	zeroStat := entity.NewUserStat("user-1", 0, 0, 0, 0, 0, 0, 0, 0)
	live := []*entity.PushSubscription{
		entity.NewPushSubscription("sub-1", weeklyReportFixedNow, "user-1", "https://push.example.com/1", "p", "a", entity.PushPlatformAndroid),
	}

	// expectEnvNewsPreconditions は「0戦・購読あり・未送信・反応あり(直近の配信なし)」までを張る
	expectEnvNewsPreconditions := func(m *weeklyReportMocks) {
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(zeroStat, nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(live, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryEnvNews, wantLink).Return(false, nil)
		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 0, 3, 0, 0, time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local), time.Now()), nil)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignEnvNews, pushQuietRecentScanLimit).Return(nil, nil)
	}

	t.Run("正常系_記録ゼロの購読者には環境ニュースを作ってpushする", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		expectEnvNewsPreconditions(m)
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(testEnvStat(), nil)
		m.pokemonSpriteName.EXPECT().FindNamesByIds(gomock.Any(), []string{"dragapult", "dusknoir", "rayquaza-mega", "ho-oh"}).Return(testSpriteNames, nil)

		var saved *entity.Notification
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, n *entity.Notification) error {
				saved = n
				return nil
			},
		)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, NotificationCategoryEnvNews, saved.Category)
		require.Equal(t, envNewsTitle, saved.Title)
		require.Equal(t, wantLink, saved.LinkUrl)
		require.Equal(t,
			"先週の環境: 使用率1位は『ドラパルト＋ヨノワール』(6.1%)。いちばん伸びたのは『メガレックウザ＋ホウオウ』(+3.9pt)。今週1戦記録すると、来週はあなたのレポートが届きます",
			saved.Body)
		require.Equal(t, []string{PushCampaignEnvNews}, m.pushNotifier.campaigns())
		require.Same(t, saved, m.pushNotifier.calls[0].notification)
	})

	t.Run("正常系_伸びたデッキが1位と同じなら前週比として1文にまとめる", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		expectEnvNewsPreconditions(m)
		stat := entity.NewWeeklyDeckUsageStat(fromDate, 100, 10, []*entity.DeckUsageVariant{
			variant("fp-ray", 0.10, 10, ptr(0.04), "rayquaza-mega", "ho-oh"),
			variant("fp-dra", 0.08, 8, ptr(0.09), "dragapult", "dusknoir"),
		})
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(stat, nil)
		m.pokemonSpriteName.EXPECT().FindNamesByIds(gomock.Any(), []string{"rayquaza-mega", "ho-oh", "rayquaza-mega", "ho-oh"}).Return(testSpriteNames, nil)

		var saved *entity.Notification
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, n *entity.Notification) error {
			saved = n
			return nil
		})

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, "先週の環境: 使用率1位は『メガレックウザ＋ホウオウ』(10.0%・前週比+6.0pt)。今週1戦記録すると、来週はあなたのレポートが届きます", saved.Body)
	})

	t.Run("正常系_前週データが無ければ伸びたデッキの文を省く", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		expectEnvNewsPreconditions(m)
		stat := entity.NewWeeklyDeckUsageStat(fromDate, 50, 5, []*entity.DeckUsageVariant{
			variant("fp-dra", 0.12, 6, nil, "dragapult", "dusknoir"),
		})
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(stat, nil)
		m.pokemonSpriteName.EXPECT().FindNamesByIds(gomock.Any(), []string{"dragapult", "dusknoir"}).Return(testSpriteNames, nil)

		var saved *entity.Notification
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, n *entity.Notification) error {
			saved = n
			return nil
		})

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.True(t, sent)
		require.Equal(t, "先週の環境: 使用率1位は『ドラパルト＋ヨノワール』(12.0%)。今週1戦記録すると、来週はあなたのレポートが届きます", saved.Body)
	})

	t.Run("正常系_一度も記録していない(ストリーク行なし)購読者にも環境ニュースを送る", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(nil, apperror.ErrRecordNotFound)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(live, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryEnvNews, wantLink).Return(false, nil)
		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, apperror.ErrRecordNotFound)
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignEnvNews, pushQuietRecentScanLimit).Return(nil, nil)
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(testEnvStat(), nil)
		m.pokemonSpriteName.EXPECT().FindNamesByIds(gomock.Any(), gomock.Any()).Return(testSpriteNames, nil)
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.True(t, sent)
	})

	t.Run("正常系_環境データは週ごとに1回だけ集計し複数ユーザーで使い回す", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		for _, uid := range []string{"user-1", "user-2"} {
			m.userStat.EXPECT().FindUserStat(gomock.Any(), uid, fromDate, toDate, uint(0)).Return(entity.NewUserStat(uid, 0, 0, 0, 0, 0, 0, 0, 0), nil)
			m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), uid).Return(live, nil)
			m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), uid, NotificationCategoryEnvNews, wantLink).Return(false, nil)
			m.userStreak.EXPECT().FindByUserId(gomock.Any(), uid).Return(nil, apperror.ErrRecordNotFound)
			m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), uid, PushCampaignEnvNews, pushQuietRecentScanLimit).Return(nil, nil)
		}
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(testEnvStat(), nil).Times(1)
		m.pokemonSpriteName.EXPECT().FindNamesByIds(gomock.Any(), gomock.Any()).Return(testSpriteNames, nil).Times(1)
		m.notification.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(2)

		for _, uid := range []string{"user-1", "user-2"} {
			sent, err := u.NotifyUser(context.Background(), uid, weeklyReportTestWeek, false)
			require.NoError(t, err)
			require.True(t, sent)
		}
	})

	t.Run("正常系_dryRunは環境ニュースも保存しない", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		expectEnvNewsPreconditions(m)
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(testEnvStat(), nil)
		m.pokemonSpriteName.EXPECT().FindNamesByIds(gomock.Any(), gomock.Any()).Return(testSpriteNames, nil)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, true)

		require.NoError(t, err)
		require.True(t, sent)
		require.Empty(t, m.pushNotifier.campaigns())
	})

	t.Run("対象外_記録ゼロで購読が無ければ何も作らない", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(zeroStat, nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(nil, nil)
		// 以降は何も呼ばれない

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_同じ週の環境ニュースが既にあれば作らない", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(zeroStat, nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(live, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryEnvNews, wantLink).Return(true, nil)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_環境データが無い週(票が0)は送らない", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		expectEnvNewsPreconditions(m)
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(entity.NewWeeklyDeckUsageStat(fromDate, 0, 0, nil), nil)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("対象外_1位のスプライト名が引けなければ送らない", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		expectEnvNewsPreconditions(m)
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(testEnvStat(), nil)
		m.pokemonSpriteName.EXPECT().FindNamesByIds(gomock.Any(), gomock.Any()).Return(map[string]string{}, nil)

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("隔週ガード_直近4回未タップかつ4週以上記録なしなら奇数週は送らない", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		m.userStat.EXPECT().FindUserStat(gomock.Any(), "user-1", fromDate, toDate, uint(0)).Return(zeroStat, nil)
		m.pushSubscription.EXPECT().FindLiveByUserId(gomock.Any(), "user-1").Return(live, nil)
		m.notification.EXPECT().ExistsByUserIdAndCategoryAndLinkUrl(gomock.Any(), "user-1", NotificationCategoryEnvNews, wantLink).Return(false, nil)
		// 最終記録は6週前
		m.userStreak.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(entity.NewUserStreak("user-1", 0, 3, 0, 0, thisMonday.AddDate(0, 0, -42), time.Now()), nil)
		var deliveries []*entity.PushDelivery
		for i := 1; i <= 4; i++ {
			deliveries = append(deliveries, entity.NewPushDelivery("d-"+strings.Repeat("x", i), weeklyReportFixedNow.AddDate(0, 0, -7*i), "user-1", "sub-1", "n", PushCampaignEnvNews, entity.PushDeliveryStatusSent, 201))
		}
		m.pushDelivery.EXPECT().FindRecentByUserIdAndCampaign(gomock.Any(), "user-1", PushCampaignEnvNews, pushQuietRecentScanLimit).Return(deliveries, nil)
		// 奇数週なので環境データの集計にも進まない

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.NoError(t, err)
		require.False(t, sent)
	})

	t.Run("異常系_環境集計のエラーはそのまま返す", func(t *testing.T) {
		m, u := setup4WeeklyReportNotifierWithPush(t)
		expectEnvNewsPreconditions(m)
		m.weeklyDeckUsageStat.EXPECT().FindWeeklyDeckUsageStat(gomock.Any(), fromDate, toDate).Return(nil, errors.New("db down"))

		sent, err := u.NotifyUser(context.Background(), "user-1", weeklyReportTestWeek, false)

		require.Error(t, err)
		require.False(t, sent)
	})
}

func TestEnvNewsBody_LongNamesAreFolded(t *testing.T) {
	long := strings.Repeat("あ", 120)
	body := envNewsBody(&envNewsHeadline{TopName: long, TopRate: 0.1, RiserName: strings.Repeat("い", 120), RiserDelta: 0.02})

	// 伸びたデッキの文を落として256文字以内に収める
	require.LessOrEqual(t, len([]rune(body)), notificationBodyMaxLength)
	require.NotContains(t, body, "いちばん伸びた")
	require.Contains(t, body, "今週1戦記録すると")
}
