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

func TestMondayOf(t *testing.T) {
	// 2026-07-03 は金曜日 → 週の月曜日は 2026-06-29
	fri := time.Date(2026, 7, 3, 15, 30, 0, 0, time.Local)
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(fri))

	// 月曜日自身は同じ日を返す
	mon := time.Date(2026, 6, 29, 9, 0, 0, 0, time.Local)
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(mon))

	// 日曜日は前週の月曜日
	sun := time.Date(2026, 7, 5, 9, 0, 0, 0, time.Local)
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(sun))

	// DBの DATE カラム(event_date / last_recorded_week)は UTC の 0時 として読み出されるが、
	// 暦日は同じ。ローカル時刻の TIMESTAMP(created_at)や time.Now() と同じ週に畳めるよう、
	// 値の Location ではなく値が持つ暦日を基準に、ローカルの月曜 0時 へ揃える。
	utcDate := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(utcDate))

	jstStamp := time.Date(2026, 7, 3, 23, 30, 0, 0, time.FixedZone("JST", 9*60*60))
	require.Equal(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.Local), mondayOf(jstStamp))
}

func TestWeeksBetween(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)

	t.Run("正常系_同じ週なら0", func(t *testing.T) {
		require.Equal(t, 0, weeksBetween(time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 6, 7, 23, 0, 0, 0, time.Local)))
	})

	t.Run("正常系_翌週なら1で以降は週数どおり", func(t *testing.T) {
		from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		require.Equal(t, 1, weeksBetween(from, from.AddDate(0, 0, 7)))
		require.Equal(t, 3, weeksBetween(from, from.AddDate(0, 0, 21)))
	})

	t.Run("正常系_過去方向なら負になる", func(t *testing.T) {
		from := time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local)
		require.Equal(t, -1, weeksBetween(from, from.AddDate(0, 0, -7)))
	})

	t.Run("正常系_UTCの暦日とローカル時刻を混ぜても暦日の差で数える", func(t *testing.T) {
		// DATE 由来(UTC 0時)の3週前の月曜と、ローカル時刻の今週。瞬間の差で数えると
		// 9時間分だけ短くなって2週に見えるが、暦日で数えれば3週。
		from := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 8, 30, 23, 0, 0, 0, jst)
		require.Equal(t, 3, weeksBetween(from, to))
	})
}

func newBadgeEvaluationTestUsecase(mockCtrl *gomock.Controller) (
	*BadgeEvaluation,
	*mock_repository.MockBadgeDefinitionInterface,
	*mock_repository.MockUserBadgeInterface,
	*mock_repository.MockUserStreakInterface,
	*mock_repository.MockBadgeStatsInterface,
	*mock_repository.MockNotificationInterface,
	*mock_repository.MockChampionshipSeriesInterface,
) {
	badgeDefinitionRepo := mock_repository.NewMockBadgeDefinitionInterface(mockCtrl)
	userBadgeRepo := mock_repository.NewMockUserBadgeInterface(mockCtrl)
	userStreakRepo := mock_repository.NewMockUserStreakInterface(mockCtrl)
	badgeStatsRepo := mock_repository.NewMockBadgeStatsInterface(mockCtrl)
	notificationRepo := mock_repository.NewMockNotificationInterface(mockCtrl)
	championshipSeriesRepo := mock_repository.NewMockChampionshipSeriesInterface(mockCtrl)

	u := &BadgeEvaluation{
		badgeDefinitionRepo:    badgeDefinitionRepo,
		userBadgeRepo:          userBadgeRepo,
		userStreakRepo:         userStreakRepo,
		badgeStatsRepo:         badgeStatsRepo,
		notificationRepo:       notificationRepo,
		championshipSeriesRepo: championshipSeriesRepo,
		transactionManager:     stubTransactionManager{},
	}

	return u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo
}

func TestBadgeEvaluation_RecomputeStreak(t *testing.T) {
	// runRecompute は指定した記録日付から user_streaks を作り直し、保存された状態を返す。
	// ストリーク継続通知の取り消しはシーズンが取れないケースとしてスキップさせ、ここでは
	// 状態の計算だけを見る(取り消し自体は TestBadgeEvaluation_RevokeStaleStreakNotifications)。
	runRecompute := func(t *testing.T, dates []time.Time) *entity.UserStreak {
		t.Helper()

		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(dates, nil)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		var saved *entity.UserStreak
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				saved = streak
				return nil
			},
		)

		require.NoError(t, u.recomputeStreak(context.Background(), "user-1"))
		require.NotNil(t, saved)

		return saved
	}

	monday := func(offsetWeeks int) time.Time {
		return time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, 7*offsetWeeks)
	}

	t.Run("正常系_初回記録は1週目として作成される", func(t *testing.T) {
		streak := runRecompute(t, []time.Time{monday(0)})

		require.Equal(t, 1, streak.CurrentWeeks)
		require.Equal(t, 1, streak.LongestWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
		require.Equal(t, mondayOf(monday(0)), streak.LastRecordedWeek)
	})

	t.Run("正常系_連続した週の分だけ連続数が増える", func(t *testing.T) {
		streak := runRecompute(t, []time.Time{monday(0), monday(1), monday(2)})

		require.Equal(t, 3, streak.CurrentWeeks)
		require.Equal(t, 3, streak.LongestWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
	})

	t.Run("正常系_同じ週内の2件目は連続数に影響しない", func(t *testing.T) {
		sameWeekLater := monday(1).AddDate(0, 0, 2)
		streak := runRecompute(t, []time.Time{monday(0), monday(1), sameWeekLater})

		require.Equal(t, 2, streak.CurrentWeeks)
		require.Equal(t, mondayOf(monday(1)), streak.LastRecordedWeek)
	})

	t.Run("正常系_1週分の空白はフリーズ枠を消費して連続扱いになる", func(t *testing.T) {
		// monday(1) を飛ばして monday(0) → monday(2)
		streak := runRecompute(t, []time.Time{monday(0), monday(2)})

		require.Equal(t, 2, streak.CurrentWeeks)
		require.Equal(t, 1, streak.FreezeUsedCount)
	})

	t.Run("正常系_過去の空き週を後から埋めると連続週数が伸びフリーズ消費も戻る", func(t *testing.T) {
		// monday(0)・monday(2) だけの状態(フリーズ1枠消費で2週連続)に、後から
		// monday(1) を入力したケース。差分更新では追えないので再計算が要る。
		streak := runRecompute(t, []time.Time{monday(0), monday(1), monday(2)})

		require.Equal(t, 3, streak.CurrentWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
	})

	t.Run("正常系_フリーズ枠を使い切ったあとの空白でリセットされる", func(t *testing.T) {
		// 1週おきに記録し、空白のたびにフリーズを消費する。上限を超えた空白でリセット。
		dates := []time.Time{monday(0)}
		for i := 1; i <= StreakMaxFreezeCount+1; i++ {
			dates = append(dates, monday(2*i))
		}

		streak := runRecompute(t, dates)

		require.Equal(t, 1, streak.CurrentWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
		require.Equal(t, StreakMaxFreezeCount+1, streak.LongestWeeks)
	})

	t.Run("正常系_フリーズ猶予を超えて空くとリセットされ最長記録は残る", func(t *testing.T) {
		streak := runRecompute(t, []time.Time{monday(0), monday(1), monday(2), monday(6)})

		require.Equal(t, 1, streak.CurrentWeeks)
		require.Equal(t, 3, streak.LongestWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
	})

	t.Run("正常系_記録が1件も無くなれば全てゼロになる", func(t *testing.T) {
		streak := runRecompute(t, nil)

		require.Equal(t, 0, streak.CurrentWeeks)
		require.Equal(t, 0, streak.LongestWeeks)
		require.Equal(t, 0, streak.FreezeUsedCount)
		require.True(t, streak.LastRecordedWeek.IsZero())
	})
}

// expectStreakRecompute は、記録の作成時に走る「現存する記録からのストリーク作り直し」と
// 「成立しなくなったストリーク継続通知の取り消し」の書き込み・問い合わせを受け流す。
// この2つは TestBadgeEvaluation_RecomputeStreak と
// TestBadgeEvaluation_RevokeStaleStreakNotifications が確かめるので、ここでは
// バッジ付与と通知作成の検証の邪魔にならないよう素通りさせる。
func expectStreakRecompute(
	userStreakRepo *mock_repository.MockUserStreakInterface,
	badgeStatsRepo *mock_repository.MockBadgeStatsInterface,
	notificationRepo *mock_repository.MockNotificationInterface,
) {
	// 全期間(fromDate/toDateがゼロ値)の取得は作り直し用。シーズン範囲の取得は
	// バッジ判定用で各テストが自前の期待を持つため、ここでは引数で棲み分ける。
	badgeStatsRepo.EXPECT().FindRecordDatesByUserId(
		gomock.Any(), gomock.Any(), time.Time{}, time.Time{},
	).Return(nil, nil).AnyTimes()
	userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	notificationRepo.EXPECT().FindByUserIdAndCategoryAndBodies(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(nil, nil).AnyTimes()
}

func TestBadgeEvaluation_EvaluateOnRecordCreated(t *testing.T) {
	t.Run("正常系_閾値に到達したバッジのみ新規付与する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-record-10", "record_count_10", "milestone", "10戦", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "3週連続", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()

		// シーズンが見つからない場合はマイルストーン/週次ストリークの判定を黙ってスキップする
		// (record作成自体は失敗させない)ことをここで併せて確認する。
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound).AnyTimes()

		// record_count=1 なので "初記録" のみ付与され、"10戦"・"3週連続" は付与されない
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.Equal(t, "def-first-record", ub.BadgeDefinitionId)
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				require.Equal(t, "user-1", n.UserId)
				require.Equal(t, NotificationCategoryBadge, n.Category)
				return nil
			},
		).Times(1)

		record := entity.NewRecord("record-1", time.Now(), 0, "", "", "", "user-1", "", "", time.Now(), false, false, entity.RegulationIdStandard, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-record", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_既に獲得済みのバッジは再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-first-record", "record-0", now),
			}, nil,
		).AnyTimes()

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil).AnyTimes()
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない(=notificationRepo.Saveも呼ばれない)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound).AnyTimes()

		record := entity.NewRecord("record-2", now, 0, "", "", "", "user-1", "", "", now, false, false, entity.RegulationIdStandard, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})

	t.Run("正常系_backfill等で過去日のrecordを再生した場合でも、achieved_atはcreated_at(記録した日時)になりevent_dateにならない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		now := time.Now()
		pastEventDate := time.Date(2020, 1, 15, 0, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound).AnyTimes()

		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.True(t, ub.AchievedAt.Equal(now))
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// event_dateは過去の対戦日(backfill入力値)だが、初記録バッジのachieved_atは
		// first_deck/first_match/signupと同様、実際に記録した日時(created_at)を採用すべき
		record := entity.NewRecord("record-1", now, 0, "", "", "", "user-1", "", "", pastEventDate, false, false, entity.RegulationIdStandard, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
	})

	t.Run("正常系_マイルストーン系(record_count)は今回の記録でシーズン内の閾値をまたいだ場合のみ通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", "milestone", "10戦達成", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).AnyTimes()

		// 1回目=オンボーディング判定用(全期間)、2回目=マイルストーン判定用(シーズンスコープ)。
		// milestone定義は無いためonboardingDefinitionsは空になり、award()の戻り値には影響しない。
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(10, nil).AnyTimes()
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return([]time.Time{now}, nil).AnyTimes()

		eventDate := now.AddDate(0, 0, -3) // 実際に対戦した(=達成した)日。記録の登録日時(now)とは別

		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				require.Equal(t, "user-1", n.UserId)
				require.Equal(t, NotificationCategoryBadge, n.Category)
				require.Contains(t, n.Body, "10戦達成")
				require.Contains(t, n.Body, "2026シーズン") // どのシーズンの実績かを明記する
				// 通知一覧の並び順が崩れないよう、通知のcreated_atは対戦日(event_date)ではなく
				// 実際の処理時刻(record.CreatedAt)を使う。
				require.True(t, n.CreatedAt.Equal(now))
				return nil
			},
		).Times(1)

		record := entity.NewRecord("record-10", now, 0, "", "", "", "user-1", "", "", eventDate, false, false, entity.RegulationIdStandard, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Empty(t, awarded) // マイルストーン系はuser_badgesに永続化されない
	})

	t.Run("正常系_マイルストーン系(record_count)は閾値をまたいでいなければ通知しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-record-10", "record_count_10", "milestone", "10戦達成", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).AnyTimes()

		// まだ6件目(閾値10に届いていない) → notificationRepo.Saveは呼ばれない(EXPECT未設定=呼ばれたら失敗)
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(6, nil).AnyTimes()
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return([]time.Time{now}, nil).AnyTimes()

		record := entity.NewRecord("record-6", now, 0, "", "", "", "user-1", "", "", now, false, false, entity.RegulationIdStandard, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})

	t.Run("正常系_週次ストリーク系はこの記録で到達した週数の閾値を通知する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		// 2026-06-15は月曜日(TestMondayOfで確認済みの2026-06-29から7日単位で遡って算出)
		thisWeekRecord := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "3週連続達成", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, thisWeekRecord, thisWeekRecord),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).AnyTimes()

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(3, nil).AnyTimes()

		// 過去2週+今回の記録(その週で最初)で3週連続
		seasonRecordDates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			thisWeekRecord,
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(seasonRecordDates, nil).AnyTimes()

		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				require.Equal(t, NotificationCategoryStreak, n.Category)
				require.Contains(t, n.Body, "3週連続達成")
				return nil
			},
		).Times(1)

		record := entity.NewRecord("record-x", thisWeekRecord, 0, "", "", "", "user-1", "", "", thisWeekRecord, false, false, entity.RegulationIdStandard, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
	})

	t.Run("正常系_onboarding系とマイルストーン系が同時に達成した場合、onboarding系の通知が先(=通知一覧では下)になる", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		// 2026-06-15は月曜日(TestMondayOfで確認済みの2026-06-29から7日単位で遡って算出)
		thisWeekRecord := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-record", "first_record", "onboarding", "初記録", "", "", BadgeCriteriaTypeRecordCount, 1, time.Time{}, time.Time{}, thisWeekRecord, thisWeekRecord),
			entity.NewBadgeDefinition("def-streak-1", "streak_week_1", "streak", "初週達成", "", "", BadgeCriteriaTypeStreakWeeks, 1, time.Time{}, time.Time{}, thisWeekRecord, thisWeekRecord),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).AnyTimes()

		// 初めての記録: onboarding判定用(全期間)・マイルストーン判定用(シーズンスコープ)とも1件
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return([]time.Time{thisWeekRecord}, nil).AnyTimes()

		var savedCategories []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				// onboarding系・マイルストーン系とも通知のcreated_atはrecord.CreatedAt
				// (=thisWeekRecord)で揃う。id DESCタイブレークで呼び出し順を制御するため。
				require.True(t, n.CreatedAt.Equal(thisWeekRecord))
				savedCategories = append(savedCategories, n.Category)
				return nil
			},
		).Times(2)

		record := entity.NewRecord("record-x", thisWeekRecord, 0, "", "", "", "user-1", "", "", thisWeekRecord, false, false, entity.RegulationIdStandard, "", "")

		awarded, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
		require.Len(t, awarded, 1)

		// onboarding系(badge)が先に、マイルストーン系(streak)が後に保存される必要がある。
		// created_atが同値のため、id DESCタイブレークにより後に保存されたマイルストーン系の
		// 通知が通知一覧で上に、onboarding系の通知が一番下に表示される。
		require.Equal(t, []string{NotificationCategoryBadge, NotificationCategoryStreak}, savedCategories)
	})

	t.Run("正常系_連続週数が一度に複数進んでも間の閾値の通知が欠落しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		// 6/1 と 6/22 だけがあった状態(3週空いて連続はリセット済み=1週)に、間の 6/15 を
		// 後から入力したケース。6/1→6/15 はフリーズ1枠で繋がり、6/15→6/22 は連続するので
		// 連続週数が 1 から 3 へ一度に進む。閾値ちょうどの一致で判定すると、飛び越えた
		// 2週バッジの通知が永久に欠落してしまう。
		pastRecord := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		thisRecord := time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)
		laterRecord := time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local)

		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-2", "streak_week_2", "streak", "2週連続達成", "", "", BadgeCriteriaTypeStreakWeeks, 2, time.Time{}, time.Time{}, thisRecord, thisRecord),
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "3週連続達成", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, thisRecord, thisRecord),
			entity.NewBadgeDefinition("def-streak-7", "streak_week_7", "streak", "7週連続達成", "", "", BadgeCriteriaTypeStreakWeeks, 7, time.Time{}, time.Time{}, thisRecord, thisRecord),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).AnyTimes()
		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(3, nil).AnyTimes()
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Any()).Return(
			[]time.Time{pastRecord, thisRecord, laterRecord}, nil,
		).AnyTimes()

		var notifiedBodies []string
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, n *entity.Notification) error {
				notifiedBodies = append(notifiedBodies, n.Body)
				return nil
			},
		).Times(2)

		record := entity.NewRecord("record-x", thisRecord, 0, "", "", "", "user-1", "", "", thisRecord, false, false, entity.RegulationIdStandard, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)

		// またいだ2週・3週の両方が通知され、届いていない7週は通知されない
		require.Len(t, notifiedBodies, 2)
		require.Contains(t, notifiedBodies[0], "2週連続達成")
		require.Contains(t, notifiedBodies[1], "3週連続達成")
	})

	t.Run("正常系_同じ週の2件目の記録では週次ストリーク通知が重複しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)
		expectStreakRecompute(userStreakRepo, badgeStatsRepo, notificationRepo)

		earlierThisWeek := time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local)
		secondRecordThisWeek := time.Date(2026, 6, 15, 18, 0, 0, 0, time.Local)
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "3週連続達成", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, secondRecordThisWeek, secondRecordThisWeek),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil).AnyTimes()
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil).AnyTimes()

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).AnyTimes()

		badgeStatsRepo.EXPECT().CountRecordsByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(4, nil).AnyTimes()

		// 同じ週に既に1件記録があるため、今回(2件目)は連続週数を進めていない
		seasonRecordDates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			earlierThisWeek,
			secondRecordThisWeek,
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(seasonRecordDates, nil).AnyTimes()

		// notificationRepo.Saveは呼ばれない(EXPECT未設定=呼ばれたら失敗)

		record := entity.NewRecord("record-y", secondRecordThisWeek, 0, "", "", "", "user-1", "", "", secondRecordThisWeek, false, false, entity.RegulationIdStandard, "", "")

		_, err := u.EvaluateOnRecordCreated(context.Background(), "user-1", record)
		require.NoError(t, err)
	})
}

func TestBadgeEvaluation_EvaluateOnMatchCreated(t *testing.T) {
	t.Run("正常系_勝敗によらず初対戦バッジが付与される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-match", "first_match", "onboarding", "初対戦", "", "", BadgeCriteriaTypeMatchCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		// 敗北した対戦(victoryFlg=false)でも「初対戦」は付与される
		match := entity.NewMatch("match-1", now, "record-1", "", "", "user-1", "", false, false, false, false, false, false, false, false, false, "", "", nil, nil)

		awarded, err := u.EvaluateOnMatchCreated(context.Background(), "user-1", match)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-match", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_既に獲得済みなら再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-match", "first_match", "onboarding", "初対戦", "", "", BadgeCriteriaTypeMatchCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-first-match", "", now),
			}, nil,
		)
		badgeStatsRepo.EXPECT().CountMatchesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(2, nil)
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		match := entity.NewMatch("match-2", now, "record-1", "", "", "user-1", "", false, false, false, false, false, false, true, false, false, "", "", nil, nil)

		awarded, err := u.EvaluateOnMatchCreated(context.Background(), "user-1", match)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})
}

func TestBadgeEvaluation_EvaluateOnDeckCreated(t *testing.T) {
	t.Run("正常系_初デッキバッジが付与される(デッキコード無し)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, notificationRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-deck", "first_deck", "onboarding", "初デッキ", "", "", BadgeCriteriaTypeDeckCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.Equal(t, "def-first-deck", ub.BadgeDefinitionId)
				require.Empty(t, ub.RecordId)
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// デッキコード無しで作成した場合、deck_codes は増えないため
		// マイルストーン系(deck_code_count)の判定は行われない
		// (=CountDeckCodesByUserId・championshipSeriesRepo は一切呼ばれない)。
		deck := entity.NewDeck("deck-1", now, time.Time{}, time.Time{}, "user-1", "リザードンex", false, nil, nil)

		awarded, err := u.EvaluateOnDeckCreated(context.Background(), "user-1", deck)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-deck", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_デッキコード付きで作成した場合はマイルストーン系(deck_code_count)も判定する", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-first-deck", "first_deck", "onboarding", "初デッキ", "", "", BadgeCriteriaTypeDeckCount, 1, time.Time{}, time.Time{}, now, now),
			entity.NewBadgeDefinition("def-deck-code-1", "deck_code_count_1", "milestone", "駆け出しビルダー", "", "", BadgeCriteriaTypeDeckCodeCount, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		badgeStatsRepo.EXPECT().CountDecksByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		badgeStatsRepo.EXPECT().CountDeckCodesByUserId(gomock.Any(), "user-1", gomock.Any(), gomock.Any()).Return(1, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(2)
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(currentChampionshipSeries(), nil).Times(2)

		deckCode := entity.NewDeckCode("deckcode-1", now, "user-1", "deck-1", "AbCdEfGhIj12", false, "")
		deck := entity.NewDeck("deck-1", now, time.Time{}, time.Time{}, "user-1", "リザードンex", false, deckCode, nil)

		awarded, err := u.EvaluateOnDeckCreated(context.Background(), "user-1", deck)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-first-deck", awarded[0].BadgeDefinitionId)
	})
}

func TestBadgeEvaluation_EvaluateOnUserCreated(t *testing.T) {
	t.Run("正常系_ユーザー登録バッジが付与される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, _, notificationRepo, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-signup", "signup", "onboarding", "バトレコユーザー", "", "", BadgeCriteriaTypeSignup, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(nil, nil)
		userBadgeRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ub *entity.UserBadge) error {
				require.Equal(t, "def-signup", ub.BadgeDefinitionId)
				require.Empty(t, ub.RecordId)
				return nil
			},
		).Times(1)
		notificationRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		awarded, err := u.EvaluateOnUserCreated(context.Background(), "user-1", now)

		require.NoError(t, err)
		require.Len(t, awarded, 1)
		require.Equal(t, "def-signup", awarded[0].BadgeDefinitionId)
	})

	t.Run("正常系_既に獲得済みなら再付与しない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, userBadgeRepo, _, _, _, _ := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()
		definitions := []*entity.BadgeDefinition{
			entity.NewBadgeDefinition("def-signup", "signup", "onboarding", "バトレコユーザー", "", "", BadgeCriteriaTypeSignup, 1, time.Time{}, time.Time{}, now, now),
		}

		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(definitions, nil)
		userBadgeRepo.EXPECT().FindByUserId(gomock.Any(), "user-1").Return(
			[]*entity.UserBadge{
				entity.NewUserBadge("ub-1", now, "user-1", "def-signup", "", now),
			}, nil,
		)
		// 既に獲得済みなので userBadgeRepo.Save は呼ばれない(=notificationRepo.Saveも呼ばれない)

		awarded, err := u.EvaluateOnUserCreated(context.Background(), "user-1", now)

		require.NoError(t, err)
		require.Empty(t, awarded)
	})
}

func TestBadgeEvaluation_EvaluateOnRecordDeleted(t *testing.T) {
	t.Run("正常系_残っている記録の日付からストリークを作り直す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, _, _, userStreakRepo, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		// 3週連続していたうち、直近1週分の記録を削除して2週連続に減った想定
		remaining := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(remaining, nil)

		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 2, streak.CurrentWeeks)
				require.Equal(t, 2, streak.LongestWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				require.Equal(t, mondayOf(remaining[1]), streak.LastRecordedWeek)
				return nil
			},
		)

		// シーズンが取れない場合はストリーク通知の取り消しを黙ってスキップする
		// (記録削除自体は失敗させない)ことをここで併せて確認する。
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrRecordNotFound)

		err := u.EvaluateOnRecordDeleted(context.Background(), "user-1")

		require.NoError(t, err)
	})

	t.Run("正常系_最後の記録を削除すると連続数0で保存されストリーク通知も全て取り消される", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, _, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		now := time.Now()

		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(nil, nil)

		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 0, streak.CurrentWeeks)
				require.Equal(t, 0, streak.LongestWeeks)
				require.Equal(t, 0, streak.FreezeUsedCount)
				require.True(t, streak.LastRecordedWeek.IsZero())
				return nil
			},
		)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Any()).Return(nil, nil)
		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(streakTestBadgeDefinitions(), nil)

		stale := []*entity.Notification{
			entity.NewNotification("ntf-3", now, "user-1", NotificationCategoryStreak, streakAchievedNotificationTitle, "2026シーズンで「週次記録3週連続」バッジを獲得しました！", notificationLinkUrlForBadge),
			entity.NewNotification("ntf-7", now, "user-1", NotificationCategoryStreak, streakAchievedNotificationTitle, "2026シーズンで「週次記録7週連続」バッジを獲得しました！", notificationLinkUrlForBadge),
		}
		notificationRepo.EXPECT().FindByUserIdAndCategoryAndBodies(
			gomock.Any(),
			"user-1",
			NotificationCategoryStreak,
			[]string{
				"2026シーズンで「週次記録3週連続」バッジを獲得しました！",
				"2026シーズンで「週次記録7週連続」バッジを獲得しました！",
			},
		).Return(stale, nil)
		notificationRepo.EXPECT().DeleteByIds(gomock.Any(), []string{"ntf-3", "ntf-7"}).Return(nil)

		err := u.EvaluateOnRecordDeleted(context.Background(), "user-1")

		require.NoError(t, err)
	})

	t.Run("正常系_削除後もまだ満たしている週数のストリーク通知は残す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, _, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		// 4週連続していたうち直近1週分を削除し、3週連続に戻った想定。
		// 3週バッジはまだ成立しているので通知を残し、7週バッジの通知だけを取り消す。
		remaining := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(remaining, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Any()).Return(remaining, nil)
		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(streakTestBadgeDefinitions(), nil)

		stale := []*entity.Notification{
			entity.NewNotification("ntf-7", time.Now(), "user-1", NotificationCategoryStreak, streakAchievedNotificationTitle, "2026シーズンで「週次記録7週連続」バッジを獲得しました！", notificationLinkUrlForBadge),
		}
		notificationRepo.EXPECT().FindByUserIdAndCategoryAndBodies(
			gomock.Any(),
			"user-1",
			NotificationCategoryStreak,
			[]string{"2026シーズンで「週次記録7週連続」バッジを獲得しました！"},
		).Return(stale, nil)
		notificationRepo.EXPECT().DeleteByIds(gomock.Any(), []string{"ntf-7"}).Return(nil)

		err := u.EvaluateOnRecordDeleted(context.Background(), "user-1")

		require.NoError(t, err)
	})

	t.Run("正常系_全ての週数を満たしたままならストリーク通知は取り消さない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, _, userStreakRepo, badgeStatsRepo, _, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		// 8週連続のうち同じ週の2件目を削除しただけで連続週数は変わらない想定。
		// 取り消し対象が無いので notificationRepo は一切呼ばれない(EXPECT未設定=呼ばれたら失敗)。
		remaining := make([]time.Time, 0, 8)
		for w := 0; w < 8; w++ {
			remaining = append(remaining, time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, 7*w))
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(remaining, nil)
		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Any()).Return(remaining, nil)
		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(streakTestBadgeDefinitions(), nil)

		err := u.EvaluateOnRecordDeleted(context.Background(), "user-1")

		require.NoError(t, err)
	})
}

func TestBadgeEvaluation_EvaluateOnRecordUpdated(t *testing.T) {
	t.Run("正常系_対戦日の変更後の記録でストリークを作り直し届かなくなった通知を取り消す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, _, userStreakRepo, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		// 3週連続だった記録の対戦日を前にずらし、週が空いて1週連続に戻った想定。
		// 加算のみの updateStreak では追えない「減る」変化を再計算で反映する。
		remaining := []time.Time{
			time.Date(2026, 4, 6, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", time.Time{}, time.Time{}).Return(remaining, nil)

		userStreakRepo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, streak *entity.UserStreak) error {
				require.Equal(t, 1, streak.CurrentWeeks)
				require.Equal(t, mondayOf(remaining[1]), streak.LastRecordedWeek)
				return nil
			},
		)

		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Any()).Return(remaining, nil)
		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(streakTestBadgeDefinitions(), nil)

		stale := []*entity.Notification{
			entity.NewNotification("ntf-3", time.Now(), "user-1", NotificationCategoryStreak, streakAchievedNotificationTitle, "2026シーズンで「週次記録3週連続」バッジを獲得しました！", notificationLinkUrlForBadge),
		}
		notificationRepo.EXPECT().FindByUserIdAndCategoryAndBodies(
			gomock.Any(),
			"user-1",
			NotificationCategoryStreak,
			[]string{
				"2026シーズンで「週次記録3週連続」バッジを獲得しました！",
				"2026シーズンで「週次記録7週連続」バッジを獲得しました！",
			},
		).Return(stale, nil)
		notificationRepo.EXPECT().DeleteByIds(gomock.Any(), []string{"ntf-3"}).Return(nil)

		err := u.EvaluateOnRecordUpdated(context.Background(), "user-1")

		require.NoError(t, err)
	})
}

func TestBadgeEvaluation_RevokeStaleStreakNotifications(t *testing.T) {
	seasonRecordDates := []time.Time{
		time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
	}
	staleBodies := []string{
		"2026シーズンで「週次記録3週連続」バッジを獲得しました！",
		"2026シーズンで「週次記録7週連続」バッジを獲得しました！",
	}

	// expectSeasonLookups は、取り消し判定に必要な「現在のシーズン」と、そのシーズン内の
	// 記録日付・バッジ定義の取得をまとめて期待する。
	expectSeasonLookups := func(
		badgeDefinitionRepo *mock_repository.MockBadgeDefinitionInterface,
		badgeStatsRepo *mock_repository.MockBadgeStatsInterface,
		championshipSeriesRepo *mock_repository.MockChampionshipSeriesInterface,
	) {
		season := entity.NewChampionshipSeries("series_2026", "2026", time.Date(2025, 9, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local))
		championshipSeriesRepo.EXPECT().FindByDate(gomock.Any(), gomock.Any()).Return(season, nil).Times(2)
		badgeStatsRepo.EXPECT().FindRecordDatesByUserId(gomock.Any(), "user-1", gomock.Not(time.Time{}), gomock.Any()).Return(seasonRecordDates, nil)
		badgeDefinitionRepo.EXPECT().FindAll(gomock.Any()).Return(streakTestBadgeDefinitions(), nil)
	}

	t.Run("正常系_dryRunでは削除せず対象の通知を返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, _, _, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		expectSeasonLookups(badgeDefinitionRepo, badgeStatsRepo, championshipSeriesRepo)

		stale := []*entity.Notification{
			entity.NewNotification("ntf-3", time.Now(), "user-1", NotificationCategoryStreak, streakAchievedNotificationTitle, staleBodies[0], notificationLinkUrlForBadge),
		}
		notificationRepo.EXPECT().FindByUserIdAndCategoryAndBodies(gomock.Any(), "user-1", NotificationCategoryStreak, staleBodies).Return(stale, nil)
		// DeleteByIds はEXPECT未設定 = 呼ばれたら失敗

		ret, err := u.RevokeStaleStreakNotifications(context.Background(), "user-1", true)

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, "ntf-3", ret[0].ID)
	})

	t.Run("正常系_対象の通知が1件も無ければ削除を呼ばない", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		u, badgeDefinitionRepo, _, _, badgeStatsRepo, notificationRepo, championshipSeriesRepo := newBadgeEvaluationTestUsecase(mockCtrl)

		expectSeasonLookups(badgeDefinitionRepo, badgeStatsRepo, championshipSeriesRepo)

		// 過去シーズンのラベルが入った通知しか無い等、本文が一致しなければ0件になる
		notificationRepo.EXPECT().FindByUserIdAndCategoryAndBodies(gomock.Any(), "user-1", NotificationCategoryStreak, staleBodies).Return(nil, nil)

		ret, err := u.RevokeStaleStreakNotifications(context.Background(), "user-1", false)

		require.NoError(t, err)
		require.Empty(t, ret)
	})
}

// streakTestBadgeDefinitions は取り消し判定のテスト用のバッジ定義一式。
// マイルストーン系を混ぜてあるのは、記録件数などの「過去に達成した事実」の通知が
// 取り消し対象に混ざらないことを併せて確認するため。
func streakTestBadgeDefinitions() []*entity.BadgeDefinition {
	now := time.Now()

	return []*entity.BadgeDefinition{
		entity.NewBadgeDefinition("def-streak-3", "streak_week_3", "streak", "週次記録3週連続", "", "", BadgeCriteriaTypeStreakWeeks, 3, time.Time{}, time.Time{}, now, now),
		entity.NewBadgeDefinition("def-streak-7", "streak_week_7", "streak", "週次記録7週連続", "", "", BadgeCriteriaTypeStreakWeeks, 7, time.Time{}, time.Time{}, now, now),
		entity.NewBadgeDefinition("def-record-10", "record_count_10", "milestone", "10戦達成", "", "", BadgeCriteriaTypeRecordCount, 10, time.Time{}, time.Time{}, now, now),
	}
}

func TestWithoutOneRecord(t *testing.T) {
	jun1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
	jun8 := time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local)

	t.Run("正常系_同じ日時が複数あっても1件だけ取り除く", func(t *testing.T) {
		// 同じ日時の記録が2件ある週で1件を作ったケース。まとめて外すと「作る前の状態」を
		// 過小に見積もり、既に通知済みの週数をもう一度通知してしまう。
		ret := withoutOneRecord([]time.Time{jun1, jun8, jun8}, jun8)

		require.Equal(t, []time.Time{jun1, jun8}, ret)
	})

	t.Run("正常系_タイムゾーンが違っても同じ対戦日なら取り除く", func(t *testing.T) {
		// DBから読み直した基準日と、リクエストで渡された値は同じ日でも別の瞬間になりうる。
		// ここで取りこぼすと「作る前の状態」が現在と同じになり、通知が出なくなる。
		// 実行環境のタイムゾーンに依存しないよう、固定オフセットで差を作る。
		basis := time.Date(2026, 6, 8, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60))
		ret := withoutOneRecord([]time.Time{jun1, jun8}, basis)

		require.Equal(t, []time.Time{jun1}, ret)
	})

	t.Run("正常系_該当する日時が無ければそのまま返す", func(t *testing.T) {
		ret := withoutOneRecord([]time.Time{jun1}, jun8)

		require.Equal(t, []time.Time{jun1}, ret)
	})

	t.Run("正常系_空なら空を返す", func(t *testing.T) {
		require.Empty(t, withoutOneRecord(nil, jun8))
	})
}

func TestComputeStreakState(t *testing.T) {
	t.Run("正常系_記録が無ければ全てゼロ値", func(t *testing.T) {
		currentWeeks, longestWeeks, freezeUsedCount, _, lastRecordedWeek := ComputeStreakState(nil)
		require.Equal(t, 0, currentWeeks)
		require.Equal(t, 0, longestWeeks)
		require.Equal(t, 0, freezeUsedCount)
		require.True(t, lastRecordedWeek.IsZero())
	})

	t.Run("正常系_連続した週なら最長連続数もそのまま反映される", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		currentWeeks, longestWeeks, freezeUsedCount, _, lastRecordedWeek := ComputeStreakState(dates)
		require.Equal(t, 3, currentWeeks)
		require.Equal(t, 3, longestWeeks)
		require.Equal(t, 0, freezeUsedCount)
		require.Equal(t, mondayOf(dates[2]), lastRecordedWeek)
	})

	t.Run("正常系_途中で途切れても最長記録は過去の値を保持する", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 5, 4, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 18, 0, 0, 0, 0, time.Local),
			// フリーズ枠を超えて大きく空白 → リセット
			time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local),
		}
		currentWeeks, longestWeeks, _, _, _ := ComputeStreakState(dates)
		require.Equal(t, 1, currentWeeks)
		require.Equal(t, 3, longestWeeks)
	})

	t.Run("正常系_DATE由来のUTC日付とTIMESTAMP由来のローカル日時が混在しても同じ週として数える", func(t *testing.T) {
		// event_date(DATE)は UTC の 0時、created_at(TIMESTAMP)はローカル時刻で読み出される。
		// 同じ週でも瞬間としては別の値になるため、週の同一視を瞬間で行うと 9時間差が
		// 「週の差0」→途切れ扱いになり、連続週数もフリーズも1週目にリセットされてしまう。
		jst := time.FixedZone("JST", 9*60*60)
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 10, 23, 30, 0, 0, jst), // 6/8 週の created_at
		}

		currentWeeks, longestWeeks, freezeUsedCount, _, lastRecordedWeek := ComputeStreakState(dates)

		require.Equal(t, 2, currentWeeks)
		require.Equal(t, 2, longestWeeks)
		require.Equal(t, 0, freezeUsedCount)
		require.Equal(t, mondayOf(dates[1]), lastRecordedWeek)
	})

	t.Run("正常系_フリーズ消費後にstreakFreezeRegenWeeks週クリーン継続すると枠が回復する", func(t *testing.T) {
		// 6/8を飛ばして6/1→6/15でフリーズを1枠消費、その後クリーンな週を
		// streakFreezeRegenWeeks週続けると使用済み枠が0に戻る。
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			// 6/8 は未記録 → 6/15 でフリーズ消費
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		clean := time.Date(2026, 6, 22, 0, 0, 0, 0, time.Local)
		for w := 0; w < streakFreezeRegenWeeks; w++ {
			dates = append(dates, clean.AddDate(0, 0, 7*w))
		}

		_, _, freezeUsedCount, freezeRegenProgress, _ := ComputeStreakState(dates)
		require.Equal(t, 0, freezeUsedCount)
		require.Equal(t, 0, freezeRegenProgress)
	})
}

func TestStreakWeeksAchievedAt(t *testing.T) {
	t.Run("正常系_記録が無ければnil", func(t *testing.T) {
		require.Nil(t, StreakWeeksAchievedAt(nil))
	})

	t.Run("正常系_連続した週は週数ごとに初めて到達した日を返す", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 8, 0, 0, 0, 0, time.Local),
			time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local),
		}
		achievedAt := StreakWeeksAchievedAt(dates)
		require.Equal(t, dates[0], achievedAt[1])
		require.Equal(t, dates[1], achievedAt[2])
		require.Equal(t, dates[2], achievedAt[3])
	})

	t.Run("正常系_途切れてリセットされても最初に到達した日はそのまま残る", func(t *testing.T) {
		dates := []time.Time{
			time.Date(2026, 5, 4, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local),
			time.Date(2026, 5, 18, 0, 0, 0, 0, time.Local),
			// フリーズ枠を超えて大きく空白 → リセットして再び1週目から
			time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local),
		}
		achievedAt := StreakWeeksAchievedAt(dates)
		require.Equal(t, dates[0], achievedAt[1])
		require.Equal(t, dates[1], achievedAt[2])
		require.Equal(t, dates[2], achievedAt[3])
		require.NotContains(t, achievedAt, 4)
	})

	t.Run("正常系_同じ週内の複数記録はその週で最も早い日付を使う", func(t *testing.T) {
		monday := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
		dates := []time.Time{
			monday.AddDate(0, 0, 3), // 同じ週の木曜
			monday,                  // 同じ週の月曜(最も早い)
		}
		achievedAt := StreakWeeksAchievedAt(dates)
		require.Equal(t, monday, achievedAt[1])
	})
}
