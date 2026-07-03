package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	DesignationCriteriaTypeOfficialGymBattleRecord  = "official_gym_battle_record"
	DesignationCriteriaTypeOfficialLeagueRecord     = "official_league_record"
	DesignationCriteriaTypeOfficialCityLeagueRecord = "official_city_league_record"
)

// DesignationLadderItem は称号のロードマップ表示用に、称号定義へ
// 対象ユーザーの到達状況(achieved)・進捗値(currentValue)を重ねたもの。
type DesignationLadderItem struct {
	Designation  *entity.Designation
	Achieved     bool
	CurrentValue int
}

// UserDesignationView はユーザーの現在の称号と、称号ロードマップ全体を表す。
type UserDesignationView struct {
	Current *entity.Designation
	Ladder  []*DesignationLadderItem
}

// DesignationTierStat は指定シーズンにおける称号ティア1つあたりの到達ユーザー数。
type DesignationTierStat struct {
	Tier      int
	UserCount int
}

// DesignationRankStatsView は指定シーズンにおける称号ティア別のユーザー数分布を表す。
// TotalUsers は tier=0(称号未達成)のユーザーを含まない、いずれかのティアに到達した
// ユーザーの合計数(=称号ランク一覧モーダルでの「モンスターボール級以上」の分母)。
type DesignationRankStatsView struct {
	TotalUsers int
	Tiers      []*DesignationTierStat
}

type DesignationInterface interface {
	GetAllDefinitions(
		ctx context.Context,
	) ([]*entity.Designation, error)

	// GetByUserId は指定シーズンの称号とロードマップを返す。season は "YYYY"
	// (シーズン識別子=終了年、例:"2026")形式。空文字なら現在のシーズン。
	// 称号は「指定シーズンの集計値」のみで都度ライブ判定する(過去シーズンの実績を
	// 永続的に保持することはせず、シーズンを切り替えるとその期間の状態がそのまま表示される)。
	GetByUserId(
		ctx context.Context,
		userId string,
		season string,
	) (*UserDesignationView, error)

	// GetRankStats は指定シーズンにおける称号ティア別のユーザー数分布を返す。
	// season の意味は GetByUserId と同じ。
	GetRankStats(
		ctx context.Context,
		season string,
	) (*DesignationRankStatsView, error)
}

type Designation struct {
	designationRepo      repository.DesignationInterface
	designationStatsRepo repository.DesignationStatsInterface
}

func NewDesignation(
	designationRepo repository.DesignationInterface,
	designationStatsRepo repository.DesignationStatsInterface,
) DesignationInterface {
	return &Designation{designationRepo, designationStatsRepo}
}

func (u *Designation) GetAllDefinitions(
	ctx context.Context,
) ([]*entity.Designation, error) {
	return u.designationRepo.FindAll(ctx)
}

func (u *Designation) GetByUserId(
	ctx context.Context,
	userId string,
	season string,
) (*UserDesignationView, error) {
	definitions, err := u.designationRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	currentValues, err := u.seasonValuesByCriteriaType(ctx, userId, season)
	if err != nil {
		return nil, err
	}

	previousCityLeagueCount, err := u.previousSeasonCityLeagueCount(ctx, userId, season)
	if err != nil {
		return nil, err
	}

	current := currentDesignation(definitions, currentValues, previousCityLeagueCount)

	currentTier := 0
	if current != nil {
		currentTier = current.Tier
	}

	ladder := make([]*DesignationLadderItem, 0, len(definitions))
	for _, def := range definitions {
		ladder = append(ladder, &DesignationLadderItem{
			Designation: def,
			Achieved:    def.Tier <= currentTier,
			// currentValues に該当 criteria_type が無い(=unimplemented)場合はゼロ値のまま
			CurrentValue: currentValues[def.CriteriaType],
		})
	}

	return &UserDesignationView{
		Current: current,
		Ladder:  ladder,
	}, nil
}

// currentDesignation は集計値(criteria_type別)から、到達している最高ティアの称号を返す。
// 称号は一本道のランクであり、各ティアの説明文が示す通り「ひとつ前のティアの条件を
// 満たした上で、さらに固有の条件を満たす」という累積構造になっている
// (例: 見習いはジムバトル5件、一人前はジムバトル5件+リーグ記録)。
// そのため tier 昇順(definitions の並び順)に評価し、最初に条件を満たさなかった時点で
// 打ち切ることで、途中のティアを飛び越えて到達することを防ぐ。
//
// 常連(criteria_type=official_city_league_record)のみ、今シーズンの件数がcriteria_valueを
// 満たすだけでなく、前シーズンにも同じ件数以上のシティリーグ記録があること(=「前シーズンに
// 引き続き」の継続条件)を求める特殊なティアなので、previousCityLeagueCount で別途判定する。
func currentDesignation(
	definitions []*entity.Designation,
	values map[string]int,
	previousCityLeagueCount int,
) *entity.Designation {
	var current *entity.Designation
	for _, def := range definitions {
		value, ok := values[def.CriteriaType]
		if !ok {
			// 判定ロジックが未実装(=「準備中」)のティアに到達したら打ち切る
			break
		}
		if value < def.CriteriaValue {
			break
		}
		if def.CriteriaType == DesignationCriteriaTypeOfficialCityLeagueRecord &&
			previousCityLeagueCount < def.CriteriaValue {
			break
		}
		current = def
	}

	return current
}

func (u *Designation) GetRankStats(
	ctx context.Context,
	season string,
) (*DesignationRankStatsView, error) {
	definitions, err := u.designationRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	fromDate, toDate, err := seasonRange(season, time.Now().Local())
	if err != nil {
		return nil, err
	}

	gymBattleCounts, err := u.designationStatsRepo.CountGymBattleRecordsGroupByUserId(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	leagueCounts, err := u.designationStatsRepo.CountLeagueRecordsGroupByUserId(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	cityLeagueCounts, err := u.designationStatsRepo.CountCityLeagueRecordsGroupByUserId(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	previousFromDate, previousToDate, err := previousSeasonRange(season, time.Now().Local())
	if err != nil {
		return nil, err
	}

	previousCityLeagueCounts, err := u.designationStatsRepo.CountCityLeagueRecordsGroupByUserId(ctx, previousFromDate, previousToDate)
	if err != nil {
		return nil, err
	}

	// いずれかの記録を持つユーザーのみが称号判定の対象になりうる(記録が全く無ければ
	// 必ず tier=0 のため、集計に含める意味が無い)。
	userIds := make(map[string]struct{})
	for userId := range gymBattleCounts {
		userIds[userId] = struct{}{}
	}
	for userId := range leagueCounts {
		userIds[userId] = struct{}{}
	}
	for userId := range cityLeagueCounts {
		userIds[userId] = struct{}{}
	}

	tierCounts := make(map[int]int)
	totalUsers := 0
	for userId := range userIds {
		values := map[string]int{
			DesignationCriteriaTypeOfficialGymBattleRecord:  gymBattleCounts[userId],
			DesignationCriteriaTypeOfficialLeagueRecord:     leagueCounts[userId],
			DesignationCriteriaTypeOfficialCityLeagueRecord: cityLeagueCounts[userId],
		}

		current := currentDesignation(definitions, values, previousCityLeagueCounts[userId])
		if current == nil {
			continue
		}

		tierCounts[current.Tier]++
		totalUsers++
	}

	tiers := make([]*DesignationTierStat, 0, len(definitions))
	for _, def := range definitions {
		tiers = append(tiers, &DesignationTierStat{
			Tier:      def.Tier,
			UserCount: tierCounts[def.Tier],
		})
	}

	return &DesignationRankStatsView{
		TotalUsers: totalUsers,
		Tiers:      tiers,
	}, nil
}

// seasonValuesByCriteriaType は判定ロジックが実装済みの criteria_type についてのみ、
// 指定シーズン(9月始まり。season空文字なら現在のシーズン)の集計値を返す。
// ここに無い criteria_type(例: "unimplemented")は「準備中」として常に未達成のまま扱われる。
func (u *Designation) seasonValuesByCriteriaType(
	ctx context.Context,
	userId string,
	season string,
) (map[string]int, error) {
	fromDate, toDate, err := seasonRange(season, time.Now().Local())
	if err != nil {
		return nil, err
	}

	gymBattleCount, err := u.designationStatsRepo.CountGymBattleRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	leagueCount, err := u.designationStatsRepo.CountLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	cityLeagueCount, err := u.designationStatsRepo.CountCityLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	return map[string]int{
		DesignationCriteriaTypeOfficialGymBattleRecord:  gymBattleCount,
		DesignationCriteriaTypeOfficialLeagueRecord:     leagueCount,
		DesignationCriteriaTypeOfficialCityLeagueRecord: cityLeagueCount,
	}, nil
}

// previousSeasonCityLeagueCount は常連(criteria_type=official_city_league_record)の
// 「前シーズンに引き続き」という継続条件を判定するための、対象シーズンのひとつ前の
// シーズンにおけるシティリーグ記録件数を返す。
func (u *Designation) previousSeasonCityLeagueCount(
	ctx context.Context,
	userId string,
	season string,
) (int, error) {
	fromDate, toDate, err := previousSeasonRange(season, time.Now().Local())
	if err != nil {
		return 0, err
	}

	return u.designationStatsRepo.CountCityLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
}
