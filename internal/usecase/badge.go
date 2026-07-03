package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// UserBadgeView はバッジ定義に、対象ユーザーの獲得状況・進捗値を重ねた読み取り専用モデル。
type UserBadgeView struct {
	Definition   *entity.BadgeDefinition
	Achieved     bool
	AchievedAt   time.Time
	CurrentValue int
}

type BadgeInterface interface {
	// GetAllDefinitions はバッジ定義マスタの一覧を返す(ユーザー非依存)。
	GetAllDefinitions(
		ctx context.Context,
	) ([]*entity.BadgeDefinition, error)

	// GetByUserId は全バッジ定義に、指定ユーザーの獲得状況・進捗値を重ねて返す。
	// season は "YYYY"(シーズン識別子=終了年、例:"2026")形式。空文字なら現在のシーズン。
	//
	// オンボーディング系(category="onboarding")は一度達成したら永久に保持される実績のため、
	// season の指定に関わらず user_badges に永続化された獲得記録をそのまま参照する。
	// マイルストーン系・週次ストリーク系はシーズンごとに再獲得可能な仕様のため、
	// 永続化された記録は参照せず、指定されたシーズン(9月始まり)の集計値から都度ライブ判定する。
	GetByUserId(
		ctx context.Context,
		userId string,
		season string,
	) ([]*UserBadgeView, error)
}

type Badge struct {
	badgeDefinitionRepo repository.BadgeDefinitionInterface
	userBadgeRepo       repository.UserBadgeInterface
	badgeStatsRepo      repository.BadgeStatsInterface
}

func NewBadge(
	badgeDefinitionRepo repository.BadgeDefinitionInterface,
	userBadgeRepo repository.UserBadgeInterface,
	badgeStatsRepo repository.BadgeStatsInterface,
) BadgeInterface {
	return &Badge{
		badgeDefinitionRepo: badgeDefinitionRepo,
		userBadgeRepo:       userBadgeRepo,
		badgeStatsRepo:      badgeStatsRepo,
	}
}

func (u *Badge) GetAllDefinitions(
	ctx context.Context,
) ([]*entity.BadgeDefinition, error) {
	return u.badgeDefinitionRepo.FindAll(ctx)
}

func (u *Badge) GetByUserId(
	ctx context.Context,
	userId string,
	season string,
) ([]*UserBadgeView, error) {
	definitions, err := u.badgeDefinitionRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	userBadges, err := u.userBadgeRepo.FindByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	achievedMap := make(map[string]*entity.UserBadge, len(userBadges))
	for _, ub := range userBadges {
		achievedMap[ub.BadgeDefinitionId] = ub
	}

	allTimeValues, err := u.allTimeValuesByCriteriaType(ctx, userId)
	if err != nil {
		return nil, err
	}

	seasonValues, err := u.seasonValuesByCriteriaType(ctx, userId, season)
	if err != nil {
		return nil, err
	}

	views := make([]*UserBadgeView, 0, len(definitions))
	for _, def := range definitions {
		if def.Category == BadgeCategoryOnboarding {
			view := &UserBadgeView{
				Definition:   def,
				CurrentValue: allTimeValues[def.CriteriaType],
			}
			if ub, ok := achievedMap[def.ID]; ok {
				view.Achieved = true
				view.AchievedAt = ub.AchievedAt
			}
			views = append(views, view)
			continue
		}

		// マイルストーン系・週次ストリーク系: 現在のシーズンの集計値だけで判定する
		// (永続化された過去の獲得記録は参照しない=シーズンが変わればライブに未達成へ戻る)。
		currentValue := seasonValues[def.CriteriaType]
		views = append(views, &UserBadgeView{
			Definition:   def,
			Achieved:     currentValue >= def.CriteriaValue,
			CurrentValue: currentValue,
		})
	}

	return views, nil
}

// allTimeValuesByCriteriaType はオンボーディング系バッジの判定に使う、全期間の集計値。
func (u *Badge) allTimeValuesByCriteriaType(
	ctx context.Context,
	userId string,
) (map[string]int, error) {
	recordCount, err := u.badgeStatsRepo.CountRecordsByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	matchCount, err := u.badgeStatsRepo.CountMatchesByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	deckCount, err := u.badgeStatsRepo.CountDecksByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	return map[string]int{
		BadgeCriteriaTypeRecordCount: recordCount,
		BadgeCriteriaTypeMatchCount:  matchCount,
		BadgeCriteriaTypeDeckCount:   deckCount,
	}, nil
}

// seasonValuesByCriteriaType はマイルストーン系・週次ストリーク系バッジの判定に使う、
// 指定シーズン(9月始まり。season空文字なら現在のシーズン)内の集計値。
func (u *Badge) seasonValuesByCriteriaType(
	ctx context.Context,
	userId string,
	season string,
) (map[string]int, error) {
	fromDate, toDate, err := seasonRange(season, time.Now().Local())
	if err != nil {
		return nil, err
	}

	recordCount, err := u.badgeStatsRepo.CountRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	matchCount, err := u.badgeStatsRepo.CountMatchesByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	deckCount, err := u.badgeStatsRepo.CountDecksByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	dates, err := u.badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	return map[string]int{
		BadgeCriteriaTypeRecordCount: recordCount,
		BadgeCriteriaTypeMatchCount:  matchCount,
		BadgeCriteriaTypeDeckCount:   deckCount,
		BadgeCriteriaTypeStreakWeeks: seasonStreakWeeks(dates),
	}, nil
}

// seasonStreakWeeks は日付の集合(重複・順不同可)から、シーズン内で直近まで継続している
// 週次記録の連続週数を求める。computeStreakState(badge_evaluation.go)と同じロジックを
// 共有し、シーズン内の期間に絞った dates を渡して連続週数だけを取り出す。
func seasonStreakWeeks(dates []time.Time) int {
	currentWeeks, _, _, _ := computeStreakState(dates)
	return currentWeeks
}
