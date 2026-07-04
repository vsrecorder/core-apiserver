package usecase

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	BadgeCriteriaTypeSignup      = "signup"
	BadgeCriteriaTypeRecordCount = "record_count"
	BadgeCriteriaTypeMatchCount  = "match_count"
	BadgeCriteriaTypeDeckCount   = "deck_count"
	BadgeCriteriaTypeStreakWeeks = "streak_weeks"
)

const (
	BadgeCategoryOnboarding = "onboarding"
	BadgeCategoryMilestone  = "milestone"
	BadgeCategoryStreak     = "streak"
)

// streakFreezeMaxGapWeeks は、記録が途切れてもフリーズ枠で連続扱いを維持できる
// 最大の空白週数(2週間分)。旅行・繁忙期等でのストリーク断絶による離脱を防ぐ
// (BADGE_STREAK_PLAN.md 2-4)。
const streakFreezeMaxGapWeeks = 2

// StreakMaxFreezeCount は1回の連続記録(ストリーク)につき使えるフリーズの上限回数。
// ストリークがリセットされると再び上限までフリーズを使えるようになる。
const StreakMaxFreezeCount = 1

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
	badgeDefinitionRepo repository.BadgeDefinitionInterface
	userBadgeRepo       repository.UserBadgeInterface
	userStreakRepo      repository.UserStreakInterface
	badgeStatsRepo      repository.BadgeStatsInterface
}

func NewBadgeEvaluation(
	badgeDefinitionRepo repository.BadgeDefinitionInterface,
	userBadgeRepo repository.UserBadgeInterface,
	userStreakRepo repository.UserStreakInterface,
	badgeStatsRepo repository.BadgeStatsInterface,
) BadgeEvaluationInterface {
	return &BadgeEvaluation{
		badgeDefinitionRepo: badgeDefinitionRepo,
		userBadgeRepo:       userBadgeRepo,
		userStreakRepo:      userStreakRepo,
		badgeStatsRepo:      badgeStatsRepo,
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

// recordBasisTime は record の日時判定の基準となる時刻を返す。
// event_date が未入力の場合は記録作成日時を代わりに使う。
func recordBasisTime(eventDate time.Time, createdAt time.Time) time.Time {
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
	week := mondayOf(recordBasisTime(eventDate, createdAt))

	current, err := u.userStreakRepo.FindByUserId(ctx, userId)
	if err != nil {
		if !errors.Is(err, apperror.ErrRecordNotFound) {
			return nil, err
		}
		current = nil
	}

	if current == nil {
		streak := entity.NewUserStreak(userId, 1, 1, 0, week, time.Now().Local())
		if err := u.userStreakRepo.Save(ctx, streak); err != nil {
			return nil, err
		}
		return streak, nil
	}

	diffDays := int(week.Sub(current.LastRecordedWeek).Hours() / 24)
	diffWeeks := diffDays / 7

	var currentWeeks, freezeUsedCount int
	lastRecordedWeek := current.LastRecordedWeek

	switch {
	case diffWeeks == 0:
		// 同じ週内の記録は連続数に影響しない
		return current, nil
	case diffDays < 0:
		// 過去日付をまとめて後入力した場合も連続数に影響させない
		return current, nil
	case diffWeeks == 1:
		currentWeeks = current.CurrentWeeks + 1
		freezeUsedCount = current.FreezeUsedCount
		lastRecordedWeek = week
	case diffWeeks <= streakFreezeMaxGapWeeks && current.FreezeUsedCount < StreakMaxFreezeCount:
		// 2週間分の空白まではフリーズ枠を消費して連続扱いにする
		currentWeeks = current.CurrentWeeks + 1
		freezeUsedCount = current.FreezeUsedCount + 1
		lastRecordedWeek = week
	default:
		// 猶予を超えて途切れた場合はストリークをリセットする
		currentWeeks = 1
		freezeUsedCount = 0
		lastRecordedWeek = week
	}

	longestWeeks := current.LongestWeeks
	if currentWeeks > longestWeeks {
		longestWeeks = currentWeeks
	}

	streak := entity.NewUserStreak(userId, currentWeeks, longestWeeks, freezeUsedCount, lastRecordedWeek, time.Now().Local())
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
func ComputeStreakState(dates []time.Time) (currentWeeks int, longestWeeks int, freezeUsedCount int, lastRecordedWeek time.Time) {
	if len(dates) == 0 {
		return 0, 0, 0, time.Time{}
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
		case diffWeeks <= streakFreezeMaxGapWeeks && freezeUsedCount < StreakMaxFreezeCount:
			currentWeeks++
			freezeUsedCount++
		default:
			currentWeeks = 1
			freezeUsedCount = 0
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

	achievedAt := recordBasisTime(record.EventDate, record.CreatedAt)
	return u.award(ctx, userId, record.ID, onboardingDefinitions(definitions), BadgeCriteriaTypeRecordCount, recordCount, achieved, achievedAt)
}

func (u *BadgeEvaluation) EvaluateOnRecordDeleted(
	ctx context.Context,
	userId string,
) error {
	dates, err := u.badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return err
	}

	currentWeeks, longestWeeks, freezeUsedCount, lastRecordedWeek := ComputeStreakState(dates)

	streak := entity.NewUserStreak(userId, currentWeeks, longestWeeks, freezeUsedCount, lastRecordedWeek, time.Now().Local())
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

	return u.award(ctx, userId, match.RecordId, onboardingDefinitions(definitions), BadgeCriteriaTypeMatchCount, matchCount, achieved, match.CreatedAt)
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

	// デッキ起点のバッジ獲得のため、紐づく record は存在しない
	return u.award(ctx, userId, "", onboardingDefinitions(definitions), BadgeCriteriaTypeDeckCount, deckCount, achieved, deck.CreatedAt)
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
