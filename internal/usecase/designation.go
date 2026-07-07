package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// DesignationCriteriaTypeRecord は、公式イベント・Tonamelイベント・記入形式のいずれで
	// あるかを問わない、記録全体の件数を条件とするティア(駆け出し・見習い)に使う。
	DesignationCriteriaTypeRecord                   = "record"
	DesignationCriteriaTypeOfficialLeagueRecord     = "official_league_record"
	DesignationCriteriaTypeOfficialCityLeagueRecord = "official_city_league_record"

	// DesignationCriteriaTypeOfficialCityLeaguePlacement は、プレイヤーズクラブ連携済みの
	// プレイヤーIDで、公式サイトの結果(cityleague_results)にそのプレイヤーIDのレコードが
	// 選択中のシーズン内に1件以上あることを条件とするティア(ベテラン)に使う。
	// records テーブル(バトレコ上でユーザー自身が作成した記録)を集計する他の criteria_type
	// と異なり、公式サイトからスクレイピングした cityleague_results を直接参照する。
	// 加えて、その cityleague_results と同じ official_event_id を持つ records が本人に
	// 存在することも内部的な条件とする(公式サイト側の結果だけでバトレコ側に記録が無い
	// 状態での到達を防ぐためのもので、ユーザーへ提示する説明文には含めない)。
	DesignationCriteriaTypeOfficialCityLeaguePlacement = "official_city_league_placement"

	// DesignationCriteriaTypeOfficialCityLeagueFinalTournament は、プレイヤーズクラブ連携済みの
	// プレイヤーIDで、公式サイトの結果(cityleague_results)にそのプレイヤーIDかつ rank が5以下の
	// レコードが選択中のシーズン内に1件以上あることを条件とするティア(熟練)に使う。
	DesignationCriteriaTypeOfficialCityLeagueFinalTournament = "official_city_league_playoff"

	// DesignationCityLeagueFinalTournamentMaxRank は熟練(criteria_type=
	// official_city_league_playoff)の判定に使う、決勝トーナメント進出とみなす
	// cityleague_results.rank の上限値。
	DesignationCityLeagueFinalTournamentMaxRank = 5

	// DesignationCityLeagueStandaloneThreshold はレギュラー(criteria_type=
	// official_city_league_record)の「前シーズンに引き続き」という継続条件を
	// 満たさなくても、今シーズン単独でこの件数以上シティリーグ記録があれば
	// 達成とみなす閾値。criteria_value(継続条件側の閾値)とは独立した固定値で、
	// presenter層がAPIレスポンス(standalone_threshold)経由でフロントエンドへ渡す。
	DesignationCityLeagueStandaloneThreshold = 2
)

// DesignationLadderItem は称号のロードマップ表示用に、称号定義へ
// 対象ユーザーの到達状況(achieved)・進捗値(currentValue)を重ねたもの。
type DesignationLadderItem struct {
	Designation  *entity.Designation
	Achieved     bool
	CurrentValue int
	// PreviousValue は常連(criteria_type=official_city_league_record)の「前シーズンに
	// 引き続き」という継続条件を表示するための、前シーズンの集計値。
	// それ以外の criteria_type では常に0(継続条件が無いため意味を持たない)。
	PreviousValue int
	// MissingOfficialEventRecord は、ベテラン(official_city_league_placement)・
	// 熟練(official_city_league_playoff)が未達成の場合に限り、その原因が
	// 「公式サイトの結果(cityleague_results)は連携済みプレイヤーIDで存在するが、
	// 対応する official_event_id の記録(records)をユーザー自身がまだ作成していないこと」
	// であるかを表す。称号詳細モーダルで「対象の大会の記録を作成してください」という
	// 案内を出し分けるためのヒント用途であり、それ以外の criteria_type では常にfalse。
	MissingOfficialEventRecord bool
	// CityLeagueRecordWithoutPlayerLink は、ベテラン(official_city_league_placement)・
	// 熟練(official_city_league_playoff)についてのみ、プレイヤーズクラブ未連携で
	// あるにもかかわらず、対象シーズン内にシティリーグの記録(records)を既に
	// 作成済みであるかを表す。称号詳細モーダルで「連携すれば達成できる可能性がある」
	// という、より具体的な案内を出し分けるためのヒント用途であり、それ以外の
	// criteria_type や、プレイヤーズクラブ連携済みの場合は常にfalse。
	CityLeagueRecordWithoutPlayerLink bool
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
	designationRepo        repository.DesignationInterface
	designationStatsRepo   repository.DesignationStatsInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
	userPlayerRepo         repository.UserPlayerInterface
}

func NewDesignation(
	designationRepo repository.DesignationInterface,
	designationStatsRepo repository.DesignationStatsInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	userPlayerRepo repository.UserPlayerInterface,
) DesignationInterface {
	return &Designation{designationRepo, designationStatsRepo, championshipSeriesRepo, userPlayerRepo}
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

	currentValues, hints, err := u.seasonValuesByCriteriaType(ctx, userId, season)
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
		previousValue := 0
		if def.CriteriaType == DesignationCriteriaTypeOfficialCityLeagueRecord {
			previousValue = previousCityLeagueCount
		}

		cityLeagueRecordWithoutPlayerLink := false
		if def.CriteriaType == DesignationCriteriaTypeOfficialCityLeaguePlacement ||
			def.CriteriaType == DesignationCriteriaTypeOfficialCityLeagueFinalTournament {
			cityLeagueRecordWithoutPlayerLink = hints.CityLeagueRecordWithoutPlayerLink
		}

		ladder = append(ladder, &DesignationLadderItem{
			Designation: def,
			Achieved:    def.Tier <= currentTier,
			// currentValues に該当 criteria_type が無い(=unimplemented)場合はゼロ値のまま
			CurrentValue:                      currentValues[def.CriteriaType],
			PreviousValue:                     previousValue,
			MissingOfficialEventRecord:        hints.MissingOfficialEventRecord[def.CriteriaType],
			CityLeagueRecordWithoutPlayerLink: cityLeagueRecordWithoutPlayerLink,
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
// (例: 見習いは記録3件、一人前は記録3件+リーグ記録)。
// そのため tier 昇順(definitions の並び順)に評価し、最初に条件を満たさなかった時点で
// 打ち切ることで、途中のティアを飛び越えて到達することを防ぐ。
//
// レギュラー(criteria_type=official_city_league_record)のみ、次のいずれかを満たす
// 特殊なティアなので、previousCityLeagueCount を使って別途判定する。
//   - 今シーズン・前シーズンともにcriteria_value以上のシティリーグ記録がある
//     (=「前シーズンに引き続き」の継続条件)
//   - 前シーズンの実績を問わず、今シーズン単独でDesignationCityLeagueStandaloneThreshold
//     件以上のシティリーグ記録がある
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

		if def.CriteriaType == DesignationCriteriaTypeOfficialCityLeagueRecord {
			continuedFromPreviousSeason := value >= def.CriteriaValue && previousCityLeagueCount >= def.CriteriaValue
			achievedAloneThisSeason := value >= DesignationCityLeagueStandaloneThreshold
			if !continuedFromPreviousSeason && !achievedAloneThisSeason {
				break
			}
			current = def
			continue
		}

		if value < def.CriteriaValue {
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

	fromDate, toDate, err := seasonRange(ctx, u.championshipSeriesRepo, season, time.Now().Local())
	if err != nil {
		return nil, err
	}

	recordCounts, err := u.designationStatsRepo.CountRecordsGroupByUserId(ctx, fromDate, toDate)
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

	previousFromDate, previousToDate, err := previousSeasonRange(ctx, u.championshipSeriesRepo, season, time.Now().Local())
	if err != nil {
		return nil, err
	}

	previousCityLeagueCounts, err := u.designationStatsRepo.CountCityLeagueRecordsGroupByUserId(ctx, previousFromDate, previousToDate)
	if err != nil {
		return nil, err
	}

	cityLeaguePlacements, err := u.designationStatsRepo.ExistsCityLeagueResultGroupByUserId(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	cityLeagueFinalTournaments, err := u.designationStatsRepo.ExistsCityLeagueFinalTournamentResultGroupByUserId(ctx, DesignationCityLeagueFinalTournamentMaxRank, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	// いずれかの記録を持つユーザーのみが称号判定の対象になりうる(記録が全く無ければ
	// 必ず tier=0 のため、集計に含める意味が無い)。
	userIds := make(map[string]struct{})
	for userId := range recordCounts {
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
			DesignationCriteriaTypeRecord:                            recordCounts[userId],
			DesignationCriteriaTypeOfficialLeagueRecord:              leagueCounts[userId],
			DesignationCriteriaTypeOfficialCityLeagueRecord:          cityLeagueCounts[userId],
			DesignationCriteriaTypeOfficialCityLeaguePlacement:       cityLeaguePlacements[userId],
			DesignationCriteriaTypeOfficialCityLeagueFinalTournament: cityLeagueFinalTournaments[userId],
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

// designationSeasonHints は seasonValuesByCriteriaType が集計値とあわせて返す、称号詳細
// モーダルの案内メッセージの出し分けにのみ使う補助情報(達成条件の判定そのものには使わない)。
type designationSeasonHints struct {
	// MissingOfficialEventRecord は DesignationLadderItem.MissingOfficialEventRecord と同じ
	// 意味を持つ値を criteria_type(ベテラン・熟練)をキーに保持する。
	MissingOfficialEventRecord map[string]bool
	// CityLeagueRecordWithoutPlayerLink は DesignationLadderItem.CityLeagueRecordWithoutPlayerLink
	// と同じ意味を持つ値。プレイヤーズクラブの連携有無は criteria_type によらずユーザー単位で
	// 決まるため、ベテラン・熟練の両方で共通の値をそのまま使う。
	CityLeagueRecordWithoutPlayerLink bool
}

// seasonValuesByCriteriaType は判定ロジックが実装済みの criteria_type についてのみ、
// 指定シーズン(9月始まり。season空文字なら現在のシーズン)の集計値を返す。
// ここに無い criteria_type(例: "unimplemented")は「準備中」として常に未達成のまま扱われる。
// あわせて designationSeasonHints(ベテラン・熟練の案内メッセージ出し分け用の補助情報)も返す。
func (u *Designation) seasonValuesByCriteriaType(
	ctx context.Context,
	userId string,
	season string,
) (map[string]int, *designationSeasonHints, error) {
	fromDate, toDate, err := seasonRange(ctx, u.championshipSeriesRepo, season, time.Now().Local())
	if err != nil {
		return nil, nil, err
	}

	recordCount, err := u.designationStatsRepo.CountRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, nil, err
	}

	leagueCount, err := u.designationStatsRepo.CountLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, nil, err
	}

	cityLeagueCount, err := u.designationStatsRepo.CountCityLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, nil, err
	}

	cityLeaguePlacement := 0
	cityLeagueFinalTournament := 0
	hints := &designationSeasonHints{
		MissingOfficialEventRecord: make(map[string]bool, 2),
	}
	userPlayer, err := u.userPlayerRepo.FindByUserId(ctx, userId)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		return nil, nil, err
	}

	// プレイヤーズクラブ未連携でも、既にシティリーグの記録(records)があるなら
	// 「連携すれば達成できる可能性がある」という案内を出すためのヒント。
	hints.CityLeagueRecordWithoutPlayerLink = userPlayer == nil && cityLeagueCount > 0

	if userPlayer != nil {
		exists, err := u.designationStatsRepo.ExistsCityLeagueResultByPlayerId(ctx, userId, userPlayer.PlayerId, fromDate, toDate)
		if err != nil {
			return nil, nil, err
		}
		if exists {
			cityLeaguePlacement = 1
		} else {
			missingRecord, err := u.designationStatsRepo.ExistsCityLeagueResultWithoutMatchingRecordByPlayerId(ctx, userId, userPlayer.PlayerId, fromDate, toDate)
			if err != nil {
				return nil, nil, err
			}
			hints.MissingOfficialEventRecord[DesignationCriteriaTypeOfficialCityLeaguePlacement] = missingRecord
		}

		existsFinalTournament, err := u.designationStatsRepo.ExistsCityLeagueFinalTournamentResultByPlayerId(ctx, userId, userPlayer.PlayerId, DesignationCityLeagueFinalTournamentMaxRank, fromDate, toDate)
		if err != nil {
			return nil, nil, err
		}
		if existsFinalTournament {
			cityLeagueFinalTournament = 1
		} else {
			missingRecord, err := u.designationStatsRepo.ExistsCityLeagueFinalTournamentResultWithoutMatchingRecordByPlayerId(ctx, userId, userPlayer.PlayerId, DesignationCityLeagueFinalTournamentMaxRank, fromDate, toDate)
			if err != nil {
				return nil, nil, err
			}
			hints.MissingOfficialEventRecord[DesignationCriteriaTypeOfficialCityLeagueFinalTournament] = missingRecord
		}
	}

	values := map[string]int{
		DesignationCriteriaTypeRecord:                            recordCount,
		DesignationCriteriaTypeOfficialLeagueRecord:              leagueCount,
		DesignationCriteriaTypeOfficialCityLeagueRecord:          cityLeagueCount,
		DesignationCriteriaTypeOfficialCityLeaguePlacement:       cityLeaguePlacement,
		DesignationCriteriaTypeOfficialCityLeagueFinalTournament: cityLeagueFinalTournament,
	}

	return values, hints, nil
}

// previousSeasonCityLeagueCount は常連(criteria_type=official_city_league_record)の
// 「前シーズンに引き続き」という継続条件を判定するための、対象シーズンのひとつ前の
// シーズンにおけるシティリーグ記録件数を返す。
func (u *Designation) previousSeasonCityLeagueCount(
	ctx context.Context,
	userId string,
	season string,
) (int, error) {
	fromDate, toDate, err := previousSeasonRange(ctx, u.championshipSeriesRepo, season, time.Now().Local())
	if err != nil {
		return 0, err
	}

	return u.designationStatsRepo.CountCityLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
}
