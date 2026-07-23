package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	BadgeCriteriaTypeSignup        = "signup"
	BadgeCriteriaTypeRecordCount   = "record_count"
	BadgeCriteriaTypeMatchCount    = "match_count"
	BadgeCriteriaTypeDeckCount     = "deck_count"
	BadgeCriteriaTypeDeckCodeCount = "deck_code_count"
	BadgeCriteriaTypeStreakWeeks   = "streak_weeks"
)

const (
	BadgeCategoryOnboarding = "onboarding"
	BadgeCategoryMilestone  = "milestone"
	BadgeCategoryStreak     = "streak"
)

// 通知(entity.Notification)のカテゴリ。webappのNotificationCategoryと一致させる。
const (
	NotificationCategoryBadge  = "badge"
	NotificationCategoryStreak = "streak"
)

// notificationLinkUrlForBadge はバッジ獲得通知のリンク先(バッジ一覧があるプロフィールページ)。
const notificationLinkUrlForBadge = "/users"

// streakFreezeMaxGapWeeks は、記録が途切れてもフリーズ枠で連続扱いを維持できる
// 最大の空白週数(2週間分)。旅行・繁忙期等でのストリーク断絶による離脱を防ぐ
// (BADGE_STREAK_PLAN.md 2-4)。
const streakFreezeMaxGapWeeks = 2

// StreakMaxFreezeCount は同時に保持できるフリーズ枠の上限回数。
// ストリークがリセットされると再び上限までフリーズを使えるようになる。また、フリーズを
// 使わずに streakFreezeRegenWeeks 週連続で記録するごとに使用済み枠が1つ回復する
// (下記 streakFreezeRegenWeeks 参照)。
const StreakMaxFreezeCount = 2

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
	// 更新し、オンボーディング系バッジを判定する。マイルストーン系・週次ストリーク系バッジは
	// シーズンごとに再獲得可能なため、書き込み時ではなく一覧取得時(usecase/badge.go)に
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
}

type BadgeEvaluation struct {
	badgeDefinitionRepo    repository.BadgeDefinitionInterface
	userBadgeRepo          repository.UserBadgeInterface
	userStreakRepo         repository.UserStreakInterface
	badgeStatsRepo         repository.BadgeStatsInterface
	notificationRepo       repository.NotificationInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
}

func NewBadgeEvaluation(
	badgeDefinitionRepo repository.BadgeDefinitionInterface,
	userBadgeRepo repository.UserBadgeInterface,
	userStreakRepo repository.UserStreakInterface,
	badgeStatsRepo repository.BadgeStatsInterface,
	notificationRepo repository.NotificationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
) BadgeEvaluationInterface {
	return &BadgeEvaluation{
		badgeDefinitionRepo:    badgeDefinitionRepo,
		userBadgeRepo:          userBadgeRepo,
		userStreakRepo:         userStreakRepo,
		badgeStatsRepo:         badgeStatsRepo,
		notificationRepo:       notificationRepo,
		championshipSeriesRepo: championshipSeriesRepo,
	}
}

// mondayOf は t が属する週(月曜始まり)の月曜日 00:00 を返す。
func mondayOf(t time.Time) time.Time {
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
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

// updateStreak はイベント発生週を基準にストリーク状態を更新する。
func (u *BadgeEvaluation) updateStreak(
	ctx context.Context,
	userId string,
	eventDate time.Time,
	createdAt time.Time,
) (*entity.UserStreak, error) {
	week := mondayOf(RecordBasisTime(eventDate, createdAt))

	current, err := u.userStreakRepo.FindByUserId(ctx, userId)
	if err != nil {
		if !errors.Is(err, apperror.ErrRecordNotFound) {
			return nil, err
		}
		current = nil
	}

	if current == nil {
		streak := entity.NewUserStreak(userId, 1, 1, 0, 0, week, time.Now().Local())
		if err := u.userStreakRepo.Save(ctx, streak); err != nil {
			return nil, err
		}
		return streak, nil
	}

	diffDays := int(week.Sub(current.LastRecordedWeek).Hours() / 24)
	diffWeeks := diffDays / 7

	currentWeeks := current.CurrentWeeks
	freezeUsedCount := current.FreezeUsedCount
	freezeRegenProgress := current.FreezeRegenProgress
	lastRecordedWeek := current.LastRecordedWeek

	switch {
	case diffWeeks == 0:
		// 同じ週内の記録は連続数に影響しない
		return current, nil
	case diffDays < 0:
		// 過去日付をまとめて後入力した場合も連続数に影響させない
		return current, nil
	case diffWeeks == 1:
		// フリーズを使わず途切れずに継続した「クリーンな週」。回復進捗を1つ進め、
		// 回復間隔に達したら使用済みフリーズ枠を1つ戻して進捗をリセットする。
		currentWeeks++
		freezeUsedCount, freezeRegenProgress = advanceFreezeRegen(freezeUsedCount, freezeRegenProgress)
		lastRecordedWeek = week
	case diffWeeks <= streakFreezeMaxGapWeeks && freezeUsedCount < StreakMaxFreezeCount:
		// 2週間分の空白まではフリーズ枠を消費して連続扱いにする。フリーズ消費で回復進捗は0に戻る。
		currentWeeks++
		freezeUsedCount++
		freezeRegenProgress = 0
		lastRecordedWeek = week
	default:
		// 猶予を超えて途切れた場合はストリークをリセットする
		currentWeeks = 1
		freezeUsedCount = 0
		freezeRegenProgress = 0
		lastRecordedWeek = week
	}

	longestWeeks := current.LongestWeeks
	if currentWeeks > longestWeeks {
		longestWeeks = currentWeeks
	}

	streak := entity.NewUserStreak(userId, currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek, time.Now().Local())
	if err := u.userStreakRepo.Save(ctx, streak); err != nil {
		return nil, err
	}

	return streak, nil
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
		diffWeeks := int(weeks[i].Sub(weeks[i-1]).Hours()/24) / 7

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

// StreakWeeksAchievedAt は dates(記録の基準日時、順不同・重複可)を走査し、週次ストリークの
// 連続週数ごとに「その週数へ初めて到達した実際の記録日」を 連続週数 -> achievedAt のマップで
// 返す。ComputeStreakStateと同じ連続判定ロジック(フリーズ枠含む)を週の推移ごとに適用する。
// backfill-notifications が、シーズンごとにライブ集計する週次ストリーク系バッジの
// 「実際の達成日」を過去の記録から遡って求めるために使う(通常のリアルタイム評価では
// notifySeasonalStreakMilestonesの判定で十分なため使わない)。
func StreakWeeksAchievedAt(dates []time.Time) map[int]time.Time {
	if len(dates) == 0 {
		return nil
	}

	firstDateOfWeek := make(map[time.Time]time.Time)
	for _, d := range dates {
		mon := mondayOf(d)
		if existing, ok := firstDateOfWeek[mon]; !ok || d.Before(existing) {
			firstDateOfWeek[mon] = d
		}
	}

	mondays := make([]time.Time, 0, len(firstDateOfWeek))
	for mon := range firstDateOfWeek {
		mondays = append(mondays, mon)
	}
	sort.Slice(mondays, func(i, j int) bool { return mondays[i].Before(mondays[j]) })

	currentWeeks := 1
	freezeUsedCount := 0
	freezeRegenProgress := 0
	achievedAt := map[int]time.Time{currentWeeks: firstDateOfWeek[mondays[0]]}

	for i := 1; i < len(mondays); i++ {
		diffWeeks := int(mondays[i].Sub(mondays[i-1]).Hours()/24) / 7

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

		if _, exists := achievedAt[currentWeeks]; !exists {
			achievedAt[currentWeeks] = firstDateOfWeek[mondays[i]]
		}
	}

	return achievedAt
}

// ComputeStreakMilestoneDates は記録日の集合(重複・順不同可)から、連続週数が
// 1,2,3...と各段階に初めて到達した実際の日付(その週内で最も早い記録日)を求める。
// ComputeStreakState と異なりストリークの最終状態ではなく到達履歴を追うため、
// 一度ストリークが途切れて同じ週数に再到達しても、シーズン内で最初に到達した
// 日付を保持する(週次ストリーク系バッジの「初回到達日」表示用)。
func ComputeStreakMilestoneDates(dates []time.Time) map[int]time.Time {
	if len(dates) == 0 {
		return nil
	}

	earliestDateInWeek := make(map[time.Time]time.Time, len(dates))
	for _, d := range dates {
		week := mondayOf(d)
		if existing, ok := earliestDateInWeek[week]; !ok || d.Before(existing) {
			earliestDateInWeek[week] = d
		}
	}

	weeks := make([]time.Time, 0, len(earliestDateInWeek))
	for w := range earliestDateInWeek {
		weeks = append(weeks, w)
	}
	sort.Slice(weeks, func(i, j int) bool { return weeks[i].Before(weeks[j]) })

	milestoneDates := make(map[int]time.Time)
	recordIfNew := func(n int, date time.Time) {
		if _, ok := milestoneDates[n]; !ok {
			milestoneDates[n] = date
		}
	}

	currentWeeks := 1
	freezeUsedCount := 0
	freezeRegenProgress := 0
	recordIfNew(currentWeeks, earliestDateInWeek[weeks[0]])

	for i := 1; i < len(weeks); i++ {
		diffWeeks := int(weeks[i].Sub(weeks[i-1]).Hours()/24) / 7

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

		recordIfNew(currentWeeks, earliestDateInWeek[weeks[i]])
	}

	return milestoneDates
}

func (u *BadgeEvaluation) achievedBadgeDefinitionIds(
	ctx context.Context,
	userId string,
) (map[string]bool, error) {
	userBadges, err := u.userBadgeRepo.FindByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	achieved := make(map[string]bool, len(userBadges))
	for _, ub := range userBadges {
		achieved[ub.BadgeDefinitionId] = true
	}

	return achieved, nil
}

// onboardingDefinitions は定義一覧からオンボーディング系(category="onboarding")のみを返す。
// マイルストーン系・週次ストリーク系は書き込み時に評価・永続化しないため、書き込み時の
// award() には常にこの絞り込み済みの一覧を渡す。
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

// streakDefinitions は定義一覧から週次ストリーク系(category="streak")のみを返す。
func streakDefinitions(definitions []*entity.BadgeDefinition) []*entity.BadgeDefinition {
	filtered := make([]*entity.BadgeDefinition, 0, len(definitions))
	for _, def := range definitions {
		if def.Category == BadgeCategoryStreak {
			filtered = append(filtered, def)
		}
	}

	return filtered
}

// notifyBadgeAchieved はバッジ獲得を知らせる通知を1件作成する。オンボーディング系
// (user_badgesに永続化される)・マイルストーン系/週次ストリーク系(都度ライブ判定、
// user_badgesには永続化されない)のどちらからも呼ばれる共通の通知生成ロジック。
// seasonLabel はシーズンごとにライブ集計する実績(マイルストーン系・週次ストリーク系)の
// 場合のみ空文字以外を渡し、本文にどのシーズンの実績かを明記する。オンボーディング系は
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
		return err
	}

	category := NotificationCategoryBadge
	title := "バッジを獲得しました"
	if def.Category == BadgeCategoryStreak {
		category = NotificationCategoryStreak
		title = "ストリークを継続中です"
	}

	body := fmt.Sprintf("「%s」バッジを獲得しました！", def.Name)
	if seasonLabel != "" {
		body = fmt.Sprintf("%sシーズンで「%s」バッジを獲得しました！", seasonLabel, def.Name)
	}

	notification := entity.NewNotification(
		id,
		achievedAt,
		userId,
		category,
		title,
		body,
		notificationLinkUrlForBadge,
	)

	return u.notificationRepo.Save(ctx, notification)
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
			return err
		}
	}

	return nil
}

// notifySeasonalStreakMilestones はシーズンスコープの記録日付一覧から現在の連続週数を求め、
// 「今回の記録がその週で最初の記録か」(=同じ週により早い日付の記録が既に無いか)を見て、
// 週次ストリーク系定義のcriteria_valueと一致すれば通知する。同じ週の2件目以降の記録では
// currentWeeksが変化しないため、この判定により重複通知を避ける。
func (u *BadgeEvaluation) notifySeasonalStreakMilestones(
	ctx context.Context,
	userId string,
	definitions []*entity.BadgeDefinition,
	seasonRecordDates []time.Time,
	thisRecordBasis time.Time,
	createdAt time.Time,
	seasonLabel string,
) error {
	currentWeeks, _, _, _, _ := ComputeStreakState(seasonRecordDates)

	thisWeek := mondayOf(thisRecordBasis)
	for _, d := range seasonRecordDates {
		if d.Equal(thisRecordBasis) {
			continue
		}
		if mondayOf(d).Equal(thisWeek) && d.Before(thisRecordBasis) {
			// 同じ週により早い記録が既にある = 今回の記録が連続週数を進めたわけではない
			return nil
		}
	}

	for _, def := range definitions {
		if def.CriteriaType != BadgeCriteriaTypeStreakWeeks {
			continue
		}
		if def.CriteriaValue != currentWeeks {
			continue
		}

		// 週の判定自体はthisRecordBasis(対戦日基準)で行うが、通知のcreated_atは
		// 実際の処理時刻(createdAt)を使う。対戦日を使うと他の通知とのcreated_at
		// 基準がずれて並び順が崩れるため。
		if err := u.notifyBadgeAchieved(ctx, userId, def, seasonLabel, createdAt); err != nil {
			return err
		}
	}

	return nil
}

// notifySeasonalMilestonesOnRecordCreated は記録作成時、シーズンスコープのマイルストーン系
// (record_count)・週次ストリーク系(streak_weeks)バッジについて新規達成があれば通知する。
// championship_seriesが見つからない等でシーズン範囲が定まらない場合は、記録作成自体を
// 失敗させないため何もせず処理を終える(通知は付随的な機能であり、本体の書き込みを
// 阻害してはならない)。
// thisRecordBasis は週次ストリーク等の判定基準(対戦日優先)、createdAt は通知の
// created_atに使う実際の処理時刻。両者を分けているのは、対戦日を通知のcreated_atに
// 使うと他の通知(バッジ獲得等)とのcreated_at基準がずれ、通知一覧の並び順が崩れるため。
func (u *BadgeEvaluation) notifySeasonalMilestonesOnRecordCreated(
	ctx context.Context,
	userId string,
	definitions []*entity.BadgeDefinition,
	thisRecordBasis time.Time,
	createdAt time.Time,
) {
	now := time.Now().Local()

	fromDate, toDate, err := seasonRange(ctx, u.championshipSeriesRepo, "", now)
	if err != nil {
		return
	}

	// 通知本文にどのシーズンの実績かを明記するためのラベル。取得できなくても
	// (通常発生しない)通知自体は空文字のまま作成し、判定を止めない。
	seasonLabel, err := CurrentSeasonLabel(ctx, u.championshipSeriesRepo, now)
	if err != nil {
		seasonLabel = ""
	}

	if seasonRecordCount, err := u.badgeStatsRepo.CountRecordsByUserId(ctx, userId, fromDate, toDate); err == nil {
		_ = u.notifySeasonalCountMilestones(ctx, userId, milestoneDefinitions(definitions), BadgeCriteriaTypeRecordCount, seasonRecordCount, seasonLabel, createdAt)
	}

	if seasonRecordDates, err := u.badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, fromDate, toDate); err == nil {
		_ = u.notifySeasonalStreakMilestones(ctx, userId, streakDefinitions(definitions), seasonRecordDates, thisRecordBasis, createdAt, seasonLabel)
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
		return
	}

	seasonLabel, err := CurrentSeasonLabel(ctx, u.championshipSeriesRepo, now)
	if err != nil {
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
			return nil, err
		}

		userBadge := entity.NewUserBadge(id, time.Now().Local(), userId, def.ID, recordId, achievedAt)

		if err := u.userBadgeRepo.Save(ctx, userBadge); err != nil {
			return nil, err
		}

		// 通知のcreated_atにもachievedAtを使う(time.Now()を使わない)。他の通知
		// (マイルストーン系・環境バッジ・称号/ランクアップ)と同じ基準の時刻に揃えることで、
		// created_at同値時のid DESCタイブレークが機能し、通知一覧の並び順を呼び出し順で
		// 制御できるようにするため。
		if err := u.notifyBadgeAchieved(ctx, userId, def, "", achievedAt); err != nil {
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
	// user_streaks(StreakPanel用の全期間ストリーク)は引き続きここで更新する。
	// バッジとしての週次ストリーク判定はシーズンごとにライブ集計するため、ここでは行わない。
	if _, err := u.updateStreak(ctx, userId, record.EventDate, record.CreatedAt); err != nil {
		return nil, err
	}

	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		return nil, err
	}

	recordCount, err := u.badgeStatsRepo.CountRecordsByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
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
		return nil, err
	}

	// シーズン系マイルストーン・週次ストリークの達成判定は実際に対戦した日(event_date)
	// 基準のまま。ただし通知のcreated_atはrecord.CreatedAt(実際の処理時刻)を使う。
	u.notifySeasonalMilestonesOnRecordCreated(ctx, userId, definitions, RecordBasisTime(record.EventDate, record.CreatedAt), record.CreatedAt)

	return awarded, nil
}

func (u *BadgeEvaluation) EvaluateOnRecordDeleted(
	ctx context.Context,
	userId string,
) error {
	dates, err := u.badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return err
	}

	currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek := ComputeStreakState(dates)

	streak := entity.NewUserStreak(userId, currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek, time.Now().Local())
	return u.userStreakRepo.Save(ctx, streak)
}

func (u *BadgeEvaluation) EvaluateOnMatchCreated(
	ctx context.Context,
	userId string,
	match *entity.Match,
) ([]*entity.UserBadge, error) {
	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		return nil, err
	}

	matchCount, err := u.badgeStatsRepo.CountMatchesByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	// onboarding系(初対戦)の通知を最も古く(=通知一覧の一番下に)するため、onboarding系を
	// 先に評価する(record作成時のEvaluateOnRecordCreatedと同じ理由)。
	awarded, err := u.award(ctx, userId, match.RecordId, onboardingDefinitions(definitions), BadgeCriteriaTypeMatchCount, matchCount, achieved, match.CreatedAt)
	if err != nil {
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
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		return nil, err
	}

	deckCount, err := u.badgeStatsRepo.CountDecksByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	// onboarding系(初デッキ)の通知を最も古く(=通知一覧の一番下に)するため、onboarding系を
	// 先に評価する(record作成時のEvaluateOnRecordCreatedと同じ理由)。
	// デッキ起点のバッジ獲得のため、紐づく record は存在しない
	awarded, err := u.award(ctx, userId, "", onboardingDefinitions(definitions), BadgeCriteriaTypeDeckCount, deckCount, achieved, deck.CreatedAt)
	if err != nil {
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
		return nil, err
	}

	achieved, err := u.achievedBadgeDefinitionIds(ctx, userId)
	if err != nil {
		return nil, err
	}

	// ユーザー登録自体が条件のため、集計クエリを挟まずその場で「1」を満たしたものとして評価する
	return u.award(ctx, userId, "", onboardingDefinitions(definitions), BadgeCriteriaTypeSignup, 1, achieved, createdAt)
}
