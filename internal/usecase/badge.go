package usecase

import (
	"context"
	"sort"
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

	fromDate, toDate, err := seasonRange(season, time.Now().Local())
	if err != nil {
		return nil, err
	}

	seasonAggregate, err := u.seasonAggregateByCriteriaType(ctx, userId, fromDate, toDate)
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
		currentValue := seasonAggregate.values[def.CriteriaType]
		view := &UserBadgeView{
			Definition:   def,
			Achieved:     currentValue >= def.CriteriaValue,
			CurrentValue: currentValue,
		}
		if at, ok := seasonAggregate.achievedAt(def.CriteriaType, def.CriteriaValue); ok {
			view.AchievedAt = at
		}
		views = append(views, view)
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

// seasonAggregate はマイルストーン系・週次ストリーク系バッジの判定に使う、指定シーズン
// 内の集計値と、criteria_value 番目の条件を満たした実際の日時(初回到達日)を求めるための
// 生データを保持する。
type seasonAggregate struct {
	values      map[string]int
	recordDates []time.Time
	deckDates   []time.Time
	matchDates  []time.Time
	streakDates map[int]time.Time
}

// achievedAt は criteriaType の criteriaValue 番目の条件を、シーズン内で最初に満たした
// 実際の日時を返す。まだ criteriaValue に届いていない(=シーズン内で未達成)場合は
// ok=false を返す。
func (a *seasonAggregate) achievedAt(criteriaType string, criteriaValue int) (time.Time, bool) {
	nthDate := func(dates []time.Time, n int) (time.Time, bool) {
		if n <= 0 || n > len(dates) {
			return time.Time{}, false
		}
		return dates[n-1], true
	}

	switch criteriaType {
	case BadgeCriteriaTypeRecordCount:
		return nthDate(a.recordDates, criteriaValue)
	case BadgeCriteriaTypeDeckCount:
		return nthDate(a.deckDates, criteriaValue)
	case BadgeCriteriaTypeMatchCount:
		return nthDate(a.matchDates, criteriaValue)
	case BadgeCriteriaTypeStreakWeeks:
		at, ok := a.streakDates[criteriaValue]
		return at, ok
	default:
		return time.Time{}, false
	}
}

// seasonAggregateByCriteriaType は指定シーズン(9月始まり。season空文字なら現在のシーズン)
// 内の集計値、および各criteria_typeの「シーズン内で何番目の条件達成が閾値に到達したか」を
// 求めるための日付一覧を取得する。
func (u *Badge) seasonAggregateByCriteriaType(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (*seasonAggregate, error) {
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

	recordDates, err := u.badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	sort.Slice(recordDates, func(i, j int) bool { return recordDates[i].Before(recordDates[j]) })

	deckDates, err := u.badgeStatsRepo.FindDeckDatesByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	sort.Slice(deckDates, func(i, j int) bool { return deckDates[i].Before(deckDates[j]) })

	matchDates, err := u.badgeStatsRepo.FindMatchDatesByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	sort.Slice(matchDates, func(i, j int) bool { return matchDates[i].Before(matchDates[j]) })

	return &seasonAggregate{
		values: map[string]int{
			BadgeCriteriaTypeRecordCount: recordCount,
			BadgeCriteriaTypeMatchCount:  matchCount,
			BadgeCriteriaTypeDeckCount:   deckCount,
			BadgeCriteriaTypeStreakWeeks: seasonStreakWeeks(recordDates),
		},
		recordDates: recordDates,
		deckDates:   deckDates,
		matchDates:  matchDates,
		streakDates: ComputeStreakMilestoneDates(recordDates),
	}, nil
}

// seasonStreakWeeks は日付の集合(重複・順不同可)から、シーズン内で直近まで継続している
// 週次記録の連続週数を求める。ComputeStreakState(badge_evaluation.go)と同じロジックを
// 共有し、シーズン内の期間に絞った dates を渡して連続週数だけを取り出す。
func seasonStreakWeeks(dates []time.Time) int {
	currentWeeks, _, _, _ := ComputeStreakState(dates)
	return currentWeeks
}
