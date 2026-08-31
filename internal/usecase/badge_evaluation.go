package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	BadgeCriteriaTypeSignup        = "signup"
	BadgeCriteriaTypeRecordCount   = "record_count"
	BadgeCriteriaTypeMatchCount    = "match_count"
	BadgeCriteriaTypeDeckCount     = "deck_count"
	BadgeCriteriaTypeDeckCodeCount = "deck_code_count"
)

const (
	BadgeCategoryOnboarding = "onboarding"
	BadgeCategoryMilestone  = "milestone"
)

// NotificationCategoryBadge は通知(entity.Notification)のカテゴリ。
// webappのNotificationCategoryと一致させる。
const NotificationCategoryBadge = "badge"

// notificationLinkUrlForBadge はバッジ獲得通知のリンク先(バッジ一覧があるプロフィールページ)。
const notificationLinkUrlForBadge = "/users"

// badgeAchievedNotificationTitle はバッジ獲得通知の見出し。
const badgeAchievedNotificationTitle = "バッジを獲得しました"

// streakFreezeMaxGapWeeks は、記録が途切れてもフリーズ枠で連続扱いを維持できる
// 最大の空白週数(2週間分)。旅行・繁忙期等でのストリーク断絶による離脱を防ぐ
// (BADGE_STREAK_PLAN.md 2-4)。
const streakFreezeMaxGapWeeks = 2

// StreakMaxFreezeCount は同時に保持できるフリーズ枠の上限回数。
// ストリークがリセットされると再び上限までフリーズを使えるようになる。また、フリーズを
// 使わずに streakFreezeRegenWeeks 週連続で記録するごとに使用済み枠が1つ回復する
// (下記 streakFreezeRegenWeeks 参照)。
const StreakMaxFreezeCount = 3

// streakFreezeRegenWeeks は、フリーズを使わずに連続記録した週数がこの数に達するごとに
// 使用済みフリーズ枠を1つ回復する(回復後はカウンタを0に戻し、次の回復まで再び数え直す)。
// 1度の中断でフリーズを使い切ったユーザーが、以降ずっとフリーズ無しになってしまうのを防ぎ、
// 継続しているほど猶予が戻る設計にする。ストリークが途切れる・フリーズを消費すると
// 進捗(FreezeRegenProgress)は0に戻る。
// フリーズ猶予(streakFreezeMaxGapWeeks)と同じ2週にし、1回サボっても2週まじめに続ければ
// 枠が戻る軽めのテンポにしている。
const streakFreezeRegenWeeks = 2

type BadgeEvaluationInterface interface {
	// EvaluateOnRecordCreated は記録作成時にストリーク状態(user_streaks、StreakPanel用)を
	// 更新し、オンボーディング系バッジを判定する。マイルストーン系バッジはシーズンごとに
	// 再獲得可能なため、書き込み時ではなく一覧取得時(usecase/badge.go)に
	// 都度ライブ集計で判定する。
	EvaluateOnRecordCreated(
		ctx context.Context,
		userId string,
		record *entity.Record,
	) ([]*entity.UserBadge, error)

	// EvaluateOnMatchCreated は対戦結果の作成時、対戦系バッジを判定する。
	EvaluateOnMatchCreated(
		ctx context.Context,
		userId string,
		match *entity.Match,
	) ([]*entity.UserBadge, error)

	// EvaluateOnDeckCreated はデッキ登録時、デッキ系バッジを判定する。
	EvaluateOnDeckCreated(
		ctx context.Context,
		userId string,
		deck *entity.Deck,
	) ([]*entity.UserBadge, error)

	// EvaluateOnDeckCodeCreated はデッキ作成後に別途デッキコードを登録した時、マイルストーン系
	// (deck_code_count)バッジの新規達成を通知する。永続化しない(ライブ集計)ため戻り値は無い。
	EvaluateOnDeckCodeCreated(
		ctx context.Context,
		userId string,
		deckCode *entity.DeckCode,
	)

	// EvaluateOnUserCreated はユーザー登録時、サインアップ系バッジを判定する。
	// createdAt はユーザーの実際の登録日時(遡及バックフィル時は過去日、通常登録時は現在時刻)で、
	// 「達成日」として user_badges.achieved_at に記録される。
	EvaluateOnUserCreated(
		ctx context.Context,
		userId string,
		createdAt time.Time,
	) ([]*entity.UserBadge, error)

	// EvaluateOnRecordDeleted は記録削除時、残っている記録の日付から
	// ストリーク状態(user_streaks)を全期間分作り直す。updateStreak は加算のみの
	// 差分更新のため、削除時にそのまま流用すると連続週数が減らずに残ってしまう。
	EvaluateOnRecordDeleted(
		ctx context.Context,
		userId string,
	) error

	// EvaluateOnRecordUpdated は記録の対戦日(event_date)が変更されたとき、
	// EvaluateOnRecordDeleted と同じく現存する記録からストリーク状態を作り直す。
	// 日付の変更は連続週数を増やしも減らしもする(週が埋まる/空く)ため、加算のみの
	// updateStreak ではなくゼロからの再計算を使う。
	EvaluateOnRecordUpdated(
		ctx context.Context,
		userId string,
	) error
}

type BadgeEvaluation struct {
	badgeDefinitionRepo    repository.BadgeDefinitionInterface
	userBadgeRepo          repository.UserBadgeInterface
	userStreakRepo         repository.UserStreakInterface
	badgeStatsRepo         repository.BadgeStatsInterface
	notificationRepo       repository.NotificationInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
	// transactionManager はバッジの付与(user_badges)とその獲得通知(notifications)を
	// 1つのトランザクションにまとめるために使う。
	transactionManager repository.TransactionManager
}

func NewBadgeEvaluation(
	badgeDefinitionRepo repository.BadgeDefinitionInterface,
	userBadgeRepo repository.UserBadgeInterface,
	userStreakRepo repository.UserStreakInterface,
	badgeStatsRepo repository.BadgeStatsInterface,
	notificationRepo repository.NotificationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	transactionManager repository.TransactionManager,
) BadgeEvaluationInterface {
	return &BadgeEvaluation{
		badgeDefinitionRepo:    badgeDefinitionRepo,
		userBadgeRepo:          userBadgeRepo,
		userStreakRepo:         userStreakRepo,
		badgeStatsRepo:         badgeStatsRepo,
		notificationRepo:       notificationRepo,
		championshipSeriesRepo: championshipSeriesRepo,
		transactionManager:     transactionManager,
	}
}

// mondayOf は t が属する週(月曜始まり)の月曜日 00:00 をローカル時刻で返す。
//
// 週の同一視は「値が持つ暦日」で行い、t の Location は見ない。DBの DATE カラム
// (records.event_date / user_streaks.last_recorded_week)は UTC の 0時 として、
// TIMESTAMP カラム(created_at 等)や time.Now() はローカル時刻として読み出されるため、
// 同じ暦日でも time.Time としては別の瞬間になる。t.Location() のまま月曜を作ると
// 同じ週が map 上で別のキーになり(ComputeStreakState で「同じ週の2件目」が
// 「週の差0=途切れ」と数えられ連続週数とフリーズがリセットされる)、Sub での週差も
// 9時間分ずれて1週少なく数えてしまう(isStreakExpired / isLastChanceThisWeek)。
// 返り値を常にローカル時刻に揃えることで、由来の異なる日付同士を map のキーや
// Sub/Before で安全に突き合わせられる。
func mondayOf(t time.Time) time.Time {
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return day.AddDate(0, 0, -(weekday - 1))
}

// weeksBetween は from が属する週から to が属する週までの週差を返す(to が後の週なら正、
// 同じ週なら0、過去なら負)。瞬間の差(Sub)ではなく暦日の差から求めるため、from と to の
// Location が違っても(DATE 由来の UTC 0時 とローカル時刻の混在)、夏時間があっても
// 週数がずれない。週次ストリークの「何週あいたか」はすべてこれで数える。
func weeksBetween(from time.Time, to time.Time) int {
	return (calendarDays(mondayOf(to)) - calendarDays(mondayOf(from))) / 7
}

// calendarDays は t の暦日を Unix epoch からの経過日数に読み替える。
// t の Location によらず「年月日」だけを見る(isEvenWeek と同じ考え方)。
func calendarDays(t time.Time) int {
	return int(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix() / (24 * 60 * 60))
}

// advanceFreezeRegen はクリーンな週(フリーズ未使用で前週から途切れず継続した週)を
// 1週進めた際のフリーズ回復進捗を計算し、回復後の (freezeUsedCount, regenProgress) を返す。
// 回復間隔(streakFreezeRegenWeeks)に達し、かつ使用済み枠が残っていれば1枠回復して進捗を
// 0に戻す。回復対象が無い(freezeUsedCount<=0)場合は進捗を溜めない。updateStreak(増分更新)と
// ComputeStreakState等(記録日付からの再計算)の双方で同じ回復ルールを使うために切り出している。
func advanceFreezeRegen(freezeUsedCount, regenProgress int) (int, int) {
	if freezeUsedCount <= 0 {
		return freezeUsedCount, 0
	}

	regenProgress++
	if regenProgress >= streakFreezeRegenWeeks {
		freezeUsedCount--
		regenProgress = 0
	}

	return freezeUsedCount, regenProgress
}

// StreakFreezeRegenWeeks は、使用済みフリーズ枠を1つ回復するのに必要なクリーン継続週数
// (streakFreezeRegenWeeks)を返す。定数自体は非公開のまま、StreakPanel等の表示で
// 「何週続けるごとに1つ復活するか」を案内するために外部へ公開する。
func StreakFreezeRegenWeeks() int {
	return streakFreezeRegenWeeks
}

// FreezeRegenRemainingWeeks は、使用済みフリーズ枠が1つ回復するまでに、あと何週の
// クリーン記録(フリーズ未使用で前週から途切れず継続)が必要かを返す。回復対象が無い
// (freezeUsedCount<=0)場合は0を返す。回復間隔(streakFreezeRegenWeeks)を外部へ露出せず、
// StreakPanel等の表示でフリーズ回復の目安をユーザーに示すために使う。
func FreezeRegenRemainingWeeks(freezeUsedCount, freezeRegenProgress int) int {
	if freezeUsedCount <= 0 {
		return 0
	}

	remaining := streakFreezeRegenWeeks - freezeRegenProgress
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// RecordBasisTime は record の日時判定の基準となる時刻を返す。
// event_date が未入力の場合は記録作成日時を代わりに使う。
func RecordBasisTime(eventDate time.Time, createdAt time.Time) time.Time {
	if eventDate.IsZero() {
		return createdAt
	}
	return eventDate
}

// ComputeStreakState は記録日の集合(重複・順不同可)から、週次ストリークの状態
// (連続週数・最長連続週数・現在のストリークで使用済みのフリーズ回数・最終記録週)を
// ゼロから計算する。updateStreak のような加算方式の差分更新と違い、渡された dates
// だけから毎回作り直すため、記録削除等で過去の記録が減っても正しい状態に戻せる。
// cmd/repair-streaks のような、既存の user_streaks を全件再計算するツールから
// 再利用できるようexportしている。
func ComputeStreakState(dates []time.Time) (currentWeeks int, longestWeeks int, freezeUsedCount int, freezeRegenProgress int, lastRecordedWeek time.Time) {
	if len(dates) == 0 {
		return 0, 0, 0, 0, time.Time{}
	}

	weekSet := make(map[time.Time]struct{}, len(dates))
	for _, d := range dates {
		weekSet[mondayOf(d)] = struct{}{}
	}

	weeks := make([]time.Time, 0, len(weekSet))
	for w := range weekSet {
		weeks = append(weeks, w)
	}
	sort.Slice(weeks, func(i, j int) bool { return weeks[i].Before(weeks[j]) })

	currentWeeks = 1
	longestWeeks = 1

	for i := 1; i < len(weeks); i++ {
		diffWeeks := weeksBetween(weeks[i-1], weeks[i])

		switch {
		case diffWeeks == 1:
			currentWeeks++
			freezeUsedCount, freezeRegenProgress = advanceFreezeRegen(freezeUsedCount, freezeRegenProgress)
		case diffWeeks <= streakFreezeMaxGapWeeks && freezeUsedCount < StreakMaxFreezeCount:
			currentWeeks++
			freezeUsedCount++
			freezeRegenProgress = 0
		default:
			currentWeeks = 1
			freezeUsedCount = 0
			freezeRegenProgress = 0
		}

		if currentWeeks > longestWeeks {
			longestWeeks = currentWeeks
		}
	}

	lastRecordedWeek = weeks[len(weeks)-1]
	return
}

func (u *BadgeEvaluation) achievedBadgeDefinitionIds(
	ctx context.Context,
	userId string,
) (map[string]bool, error) {
	userBadges, err := u.userBadgeRepo.FindByUserId(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	achieved := make(map[string]bool, len(userBadges))
	for _, ub := range userBadges {
		achieved[ub.BadgeDefinitionId] = true
	}

	return achieved, nil
}

// onboardingDefinitions は定義一覧からオンボーディング系(category="onboarding")のみを返す。
// マイルストーン系は書き込み時に評価・永続化しないため、書き込み時の award() には
// 常にこの絞り込み済みの一覧を渡す。
func onboardingDefinitions(definitions []*entity.BadgeDefinition) []*entity.BadgeDefinition {
	filtered := make([]*entity.BadgeDefinition, 0, len(definitions))
	for _, def := range definitions {
		if def.Category == BadgeCategoryOnboarding {
			filtered = append(filtered, def)
		}
	}

	return filtered
}

// milestoneDefinitions は定義一覧からマイルストーン系(category="milestone")のみを返す。
func milestoneDefinitions(definitions []*entity.BadgeDefinition) []*entity.BadgeDefinition {
	filtered := make([]*entity.BadgeDefinition, 0, len(definitions))
	for _, def := range definitions {
		if def.Category == BadgeCategoryMilestone {
			filtered = append(filtered, def)
		}
	}

	return filtered
}

// notifyBadgeAchieved はバッジ獲得を知らせる通知を1件作成する。オンボーディング系
// (user_badgesに永続化される)・マイルストーン系(都度ライブ判定、user_badgesには
// 永続化されない)のどちらからも呼ばれる共通の通知生成ロジック。
// seasonLabel はシーズンごとにライブ集計する実績(マイルストーン系)の場合のみ空文字以外を
// 渡し、本文にどのシーズンの実績かを明記する。オンボーディング系は
// シーズンの概念が無いため常に空文字を渡す。achievedAt は通知のcreated_atに使う
// 「実際に達成した日時」(記録のevent_date等)。オンボーディング系は現在時刻を渡す。
func (u *BadgeEvaluation) notifyBadgeAchieved(
	ctx context.Context,
	userId string,
	def *entity.BadgeDefinition,
	seasonLabel string,
	achievedAt time.Time,
) error {
	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return err
	}

	body := badgeAchievedNotificationBody(def, seasonLabel)

	notification := entity.NewNotification(
		id,
		achievedAt,
		userId,
		NotificationCategoryBadge,
		badgeAchievedNotificationTitle,
		body,
		notificationLinkUrlForBadge,
	)

	return u.notificationRepo.Save(ctx, notification)
}

// badgeAchievedNotificationBody はバッジ獲得通知の本文を組み立てる。
func badgeAchievedNotificationBody(def *entity.BadgeDefinition, seasonLabel string) string {
	if seasonLabel != "" {
		return fmt.Sprintf("%sシーズンで「%s」バッジを獲得しました！", seasonLabel, def.Name)
	}

	return fmt.Sprintf("「%s」バッジを獲得しました！", def.Name)
}

// notifySeasonalCountMilestones は、この1件のCreateでシーズンスコープのcriteriaType別
// カウントがちょうど閾値をまたいだ(oldCount = newSeasonCount-1 < criteria_value <=
// newSeasonCount)マイルストーン系定義について通知を作成する。オンボーディング系のaward()と
// 異なりuser_badgesには永続化しない(シーズンが変わればライブに未達成へ戻る仕様のため)。
func (u *BadgeEvaluation) notifySeasonalCountMilestones(
	ctx context.Context,
	userId string,
	definitions []*entity.BadgeDefinition,
	criteriaType string,
	newSeasonCount int,
	seasonLabel string,
	achievedAt time.Time,
) error {
	oldSeasonCount := newSeasonCount - 1

	for _, def := range definitions {
		if def.CriteriaType != criteriaType {
			continue
		}
		if !(oldSeasonCount < def.CriteriaValue && def.CriteriaValue <= newSeasonCount) {
			continue
		}

		if err := u.notifyBadgeAchieved(ctx, userId, def, seasonLabel, achievedAt); err != nil {
			logError(ctx, err)
			return err
		}
	}

	return nil
}

// isSameEventDate は2つの対戦日が同じ暦日を指すかを返す。
// records.event_date は DATE カラムのため、DBから読み直した値と、リクエストで渡された
// 値(webappは "YYYY-MM-DDT00:00:00Z" と送る)は、同じ日でも time.Time としては別の
// 瞬間になりうる。対戦日の同一性を見たい場面では、瞬間ではなく年月日で比較する。
func isSameEventDate(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()

	return ay == by && am == bm && ad == bd
}

// notifySeasonalMilestonesOnRecordCreated は記録作成時、シーズンスコープのマイルストーン系
// (record_count)バッジについて新規達成があれば通知する。
// championship_seriesが見つからない等でシーズン範囲が定まらない場合は、記録作成自体を
// 失敗させないため何もせず処理を終える(通知は付随的な機能であり、本体の書き込みを
// 阻害してはならない)。
// createdAt は通知のcreated_atに使う実際の処理時刻。対戦日(event_date)ではなくこちらを
// 使うのは、対戦日だと他の通知(バッジ獲得等)とのcreated_at基準がずれ、通知一覧の
// 並び順が崩れるため。
func (u *BadgeEvaluation) notifySeasonalMilestonesOnRecordCreated(
	ctx context.Context,
	userId string,
	definitions []*entity.BadgeDefinition,
	createdAt time.Time,
) {
	now := time.Now().Local()

	fromDate, toDate, err := seasonRange(ctx, u.championshipSeriesRepo, "", now)
	if err != nil {
		logError(ctx, err)
		return
	}

	// 通知本文にどのシーズンの実績かを明記するためのラベル。取得できなくても
	// (通常発生しない)通知自体は空文字のまま作成し、判定を止めない。
	seasonLabel, err := CurrentSeasonLabel(ctx, u.championshipSeriesRepo, now)
	if err != nil {
		logWarn(ctx, err)
		seasonLabel = ""
	}

	if seasonRecordCount, err := u.badgeStatsRepo.CountRecordsByUserId(ctx, userId, fromDate, toDate); err == nil {
		_ = u.notifySeasonalCountMilestones(ctx, userId, milestoneDefinitions(definitions), BadgeCriteriaTypeRecordCount, seasonRecordCount, seasonLabel, createdAt)
	}
}

// notifySeasonalCountMilestonesForCriteria はmatch/deck作成時、シーズンスコープの
// マイルストーン系バッジについて新規達成があれば通知する。エラー処理方針は
// notifySeasonalMilestonesOnRecordCreatedと同様(記録作成自体は失敗させない)。
// achievedAt は通知のcreated_atに使う実際の達成日時(match/deckの作成日時)。
func (u *BadgeEvaluation) notifySeasonalCountMilestonesForCriteria(
	ctx context.Context,
	userId string,
	definitions []*entity.BadgeDefinition,
	criteriaType string,
	achievedAt time.Time,
) {
	now := time.Now().Local()

	fromDate, toDate, err := seasonRange(ctx, u.championshipSeriesRepo, "", now)
	if err != nil {
		logError(ctx, err)
		return
	}

	seasonLabel, err := CurrentSeasonLabel(ctx, u.championshipSeriesRepo, now)
	if err != nil {
		logWarn(ctx, err)
		seasonLabel = ""
	}

	var newSeasonCount int
	switch criteriaType {
	case BadgeCriteriaTypeMatchCount:
		newSeasonCount, err = u.badgeStatsRepo.CountMatchesByUserId(ctx, userId, fromDate, toDate)
	case BadgeCriteriaTypeDeckCodeCount:
		newSeasonCount, err = u.badgeStatsRepo.CountDeckCodesByUserId(ctx, userId, fromDate, toDate)
	default:
		return
	}
	if err != nil {
		logError(ctx, err)
		return
	}

	_ = u.notifySeasonalCountMilestones(ctx, userId, milestoneDefinitions(definitions), criteriaType, newSeasonCount, seasonLabel, achievedAt)
}

// award は criteriaType に該当する未獲得のバッジ定義のうち、
// currentValue が閾値に達したものを新規付与する。
// achievedAt には条件を満たした実際の日時(record/deck/matchの作成日時等)を渡す。
// 通常のリアルタイム評価では概ね現在時刻と一致するが、backfill-badges による
// 遡及計算では過去日になるため、achieved_at を time.Now() 固定にしてはならない。
func (u *BadgeEvaluation) award(
	ctx context.Context,
	userId string,
	recordId string,
	definitions []*entity.BadgeDefinition,
	criteriaType string,
	currentValue int,
	achieved map[string]bool,
	achievedAt time.Time,
) ([]*entity.UserBadge, error) {
	var awarded []*entity.UserBadge

	for _, def := range definitions {
		if def.CriteriaType != criteriaType {
			continue
		}
		if achieved[def.ID] {
			continue
		}
		if currentValue < def.CriteriaValue {
			continue
		}

		id, err := generateId()
		if err != nil {
			logError(ctx, err)
			return nil, err
		}

		userBadge := entity.NewUserBadge(id, time.Now().Local(), userId, def.ID, recordId, achievedAt)

		// 付与と獲得通知は1つのトランザクションにまとめる。付与だけ残って通知が
		// 落ちると、次回以降は「獲得済み」と判定されてここを通らないため、その
		// バッジの通知は二度と作られない。
		//
		// 通知のcreated_atにもachievedAtを使う(time.Now()を使わない)。他の通知
		// (マイルストーン系・環境バッジ・称号/ランクアップ)と同じ基準の時刻に揃えることで、
		// created_at同値時のid DESCタイブレークが機能し、通知一覧の並び順を呼び出し順で
		// 制御できるようにするため。
		if err := u.transactionManager.Do(ctx, func(ctx context.Context) error {
			if err := u.userBadgeRepo.Save(ctx, userBadge); err != nil {
				logError(ctx, err)
				return err
			}

			return u.notifyBadgeAchieved(ctx, userId, def, "", achievedAt)
		}); err != nil {
			logError(ctx, err)
			return nil, err
		}

		achieved[def.ID] = true
		awarded = append(awarded, userBadge)
	}

	return awarded, nil
}

func (u *BadgeEvaluation) EvaluateOnRecordCreated(
	ctx context.Context,
	userId string,
	record *entity.Record,
) ([]*entity.UserBadge, error) {
	// user_streaks(StreakPanel用の全期間ストリーク)はここで更新する。
	//
	// 差分更新(直前の状態に1週分を足す)ではなく、削除・更新時と同じく現存する記録から
	// 作り直す。過去の日付をあとから入力すると、既に記録済みの週より前の空き週が埋まって
	// 連続週数が伸びるが、これは差分では追えないため(records は Save 済みなので、この
	// 記録自身も再計算に含まれる)。
	if err := u.recomputeStreak(ctx, userId); err != nil {
		logError(ctx, err)
		return nil, err
	}

	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	recordCount, err := u.badgeStatsRepo.CountRecordsByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// onboarding系(初記録)の通知を最も古く(=通知一覧の一番下に)するため、onboarding系を
	// 先に評価し、マイルストーン系を後に評価する。award()・notifySeasonalMilestones...とも
	// 通知のcreated_atにはrecord.CreatedAt(実際の処理時刻)を使うため、同値時のid DESC
	// タイブレークにより、後に評価したマイルストーン系の通知が上に表示される。
	//
	// onboarding系(初記録)は他のオンボーディングバッジ(first_deck/first_match/signup)と
	// 同様、実際に記録した日時(created_at)を採用する。event_dateは過去の対戦日を
	// 表す入力値であり、backfill入力等でachieved_atが過去日にずれてしまうのを避ける。
	awarded, err := u.award(ctx, userId, record.ID, onboardingDefinitions(definitions), BadgeCriteriaTypeRecordCount, recordCount, achieved, record.CreatedAt)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// シーズン系マイルストーンの達成通知。created_atはrecord.CreatedAt(実際の処理時刻)を使う。
	u.notifySeasonalMilestonesOnRecordCreated(ctx, userId, definitions, record.CreatedAt)

	return awarded, nil
}

func (u *BadgeEvaluation) EvaluateOnRecordDeleted(
	ctx context.Context,
	userId string,
) error {
	return u.recomputeStreak(ctx, userId)
}

func (u *BadgeEvaluation) EvaluateOnRecordUpdated(
	ctx context.Context,
	userId string,
) error {
	return u.recomputeStreak(ctx, userId)
}

// recomputeStreak は現存する記録の日付から user_streaks を作り直す。記録の作成・削除・
// 更新のいずれもこの1本を通し、状態の求め方をここに集約する(差分更新を併用すると、
// 過去日付の後入力のように差分では追えない変化で食い違うため)。
func (u *BadgeEvaluation) recomputeStreak(
	ctx context.Context,
	userId string,
) error {
	dates, err := u.badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		logError(ctx, err)
		return err
	}

	currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek := ComputeStreakState(dates)

	streak := entity.NewUserStreak(userId, currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek, time.Now().Local())
	if err := u.userStreakRepo.Save(ctx, streak); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func (u *BadgeEvaluation) EvaluateOnMatchCreated(
	ctx context.Context,
	userId string,
	match *entity.Match,
) ([]*entity.UserBadge, error) {
	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	matchCount, err := u.badgeStatsRepo.CountMatchesByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// onboarding系(初対戦)の通知を最も古く(=通知一覧の一番下に)するため、onboarding系を
	// 先に評価する(record作成時のEvaluateOnRecordCreatedと同じ理由)。
	awarded, err := u.award(ctx, userId, match.RecordId, onboardingDefinitions(definitions), BadgeCriteriaTypeMatchCount, matchCount, achieved, match.CreatedAt)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	u.notifySeasonalCountMilestonesForCriteria(ctx, userId, definitions, BadgeCriteriaTypeMatchCount, match.CreatedAt)

	return awarded, nil
}

func (u *BadgeEvaluation) EvaluateOnDeckCreated(
	ctx context.Context,
	userId string,
	deck *entity.Deck,
) ([]*entity.UserBadge, error) {
	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	deckCount, err := u.badgeStatsRepo.CountDecksByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// onboarding系(初デッキ)の通知を最も古く(=通知一覧の一番下に)するため、onboarding系を
	// 先に評価する(record作成時のEvaluateOnRecordCreatedと同じ理由)。
	// デッキ起点のバッジ獲得のため、紐づく record は存在しない
	awarded, err := u.award(ctx, userId, "", onboardingDefinitions(definitions), BadgeCriteriaTypeDeckCount, deckCount, achieved, deck.CreatedAt)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// マイルストーン系(deck_code_count)はデッキ「登録」数ではなくデッキ「コード」登録数を
	// 見る仕様のため、デッキコード付きで作成された場合のみ判定する。コード無しで作成した
	// 場合は deck_codes が増えていないため判定不要(むしろ判定すると誤ってカウントされる)。
	if deck.LatestDeckCode != nil && deck.LatestDeckCode.Code != "" {
		u.notifySeasonalCountMilestonesForCriteria(ctx, userId, definitions, BadgeCriteriaTypeDeckCodeCount, deck.CreatedAt)
	}

	return awarded, nil
}

// EvaluateOnDeckCodeCreated はデッキ作成後に別途デッキコードを登録した時、マイルストーン系
// (deck_code_count)バッジの新規達成を通知する。デッキ作成時に既にコードがある場合は
// EvaluateOnDeckCreated 側で判定済みのため、ここでは呼ばれない。
func (u *BadgeEvaluation) EvaluateOnDeckCodeCreated(
	ctx context.Context,
	userId string,
	deckCode *entity.DeckCode,
) {
	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		logError(ctx, err)
		return
	}

	u.notifySeasonalCountMilestonesForCriteria(ctx, userId, definitions, BadgeCriteriaTypeDeckCodeCount, deckCode.CreatedAt)
}

func (u *BadgeEvaluation) EvaluateOnUserCreated(
	ctx context.Context,
	userId string,
	createdAt time.Time,
) ([]*entity.UserBadge, error) {
	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// ユーザー登録自体が条件のため、集計クエリを挟まずその場で「1」を満たしたものとして評価する
	return u.award(ctx, userId, "", onboardingDefinitions(definitions), BadgeCriteriaTypeSignup, 1, achieved, createdAt)
}
