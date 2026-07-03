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

	// 称号は一本道のランクであり、各ティアの説明文が示す通り「ひとつ前のティアの条件を
	// 満たした上で、さらに固有の条件を満たす」という累積構造になっている
	// (例: 見習いはジムバトル5件、一人前はジムバトル5件+リーグ記録)。
	// そのため tier 昇順(designationRepo.FindAll の並び順)に評価し、最初に条件を
	// 満たさなかった時点で打ち切ることで、途中のティアを飛び越えて到達することを防ぐ。
	var current *entity.Designation
	for _, def := range definitions {
		value, ok := currentValues[def.CriteriaType]
		if !ok {
			// 判定ロジックが未実装(=「準備中」)のティアに到達したら打ち切る
			break
		}
		if value < def.CriteriaValue {
			break
		}
		current = def
	}

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
