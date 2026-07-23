package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// 通知(entity.Notification)のカテゴリ。webappのNotificationCategoryと一致させる。
const (
	NotificationCategoryDesignation = "designation"
	NotificationCategoryRank        = "rank"
)

// notificationLinkUrlForDesignation は称号/ランク通知のリンク先(称号ロードマップがある
// プロフィールページ)。バッジ通知と同じ遷移先を使う。
const notificationLinkUrlForDesignation = "/users"

// rankRange はwebapp/src/app/utils/designationRank.tsのRANKSと同じ、称号tierを
// まとめた「ランク」のグルーピング。ランク自体は独自の永続データを持たず現在のtierから
// 都度導出する設計のため、バックエンド側にも同じテーブルを複製する。
type rankRange struct {
	minTier int
	maxTier int
	name    string
}

var rankRanges = []rankRange{
	{1, 2, "モンスターボール級"},
	{3, 4, "スーパーボール級"},
	{5, 6, "ハイパーボール級"},
	{7, 8, "マスターボール級"},
	{9, 10, "ウルトラボール級"},
}

// RankNameForTier はtierが属するランク名を返す(該当なし=称号未達成なら空文字)。
func RankNameForTier(tier int) string {
	for _, r := range rankRanges {
		if tier >= r.minTier && tier <= r.maxTier {
			return r.name
		}
	}

	return ""
}

// MinTierForRank はrankNameに属するランクの最小tierを返す(該当なしなら0)。
// backfill-notifications がランク到達日を探索する際、tierの離散サンプリングで
// tier幅の狭いランク帯(例: ハイパーボール級はtier5のみ)を飛び越えてしまい、
// rankNameの完全一致では到達日が見つからなくなるケースがあるため、
// 「このランクの最小tier以上に達した最初の日」を探す用途で使う。
func MinTierForRank(rankName string) int {
	for _, r := range rankRanges {
		if r.name == rankName {
			return r.minTier
		}
	}

	return 0
}

// DesignationEvaluationInterface は記録作成時に称号(designation)のtierが上がったか
// 判定し、上がっていれば称号獲得・ランクアップの通知を作成する。
//
// 称号のtierは6種類のcriteria_type(record/official_league_record/
// official_city_league_record/official_city_league_placement/official_city_league_playoff/
// official_city_league_champion)の組み合わせで判定されるが、CurrentTier/NotifyIfTierChanged/
// NotifyIfTierLost(record作成・削除のイベント駆動で呼ばれるもの)が対象とするのはrecords起因の
// 最初の3つのみ(usecase/designation.goのDesignationCriteriaType*定数を参照)。残り3つ
// (ベテラン・熟練・達人)は連携済みプレイヤーIDでの公式サイト結果(cityleague_results)の有無で
// 判定され、これは import-cityleague-result-job(別リポジトリ、日次バッチ)がデータを取り込んだ
// 瞬間に変化しうるものであり、core-apiserverの書き込みイベントと無関係なため対象外とする。
// currentDesignation()はvaluesマップに無いcriteria_typeに到達すると判定を打ち切る
// ため、この3つだけを渡しても後続tier(5以降)を誤って達成扱いにすることはない。
// ただし TierAsOf のみ、この3つも含めて判定する(TierAsOfのコメント参照)。
type DesignationEvaluationInterface interface {
	// CurrentTier は現在のシーズンにおける現在のtier(称号未達成なら0)を返す。
	// record作成の前後で比較するため、呼び出し側は保存前に一度呼んでおく。
	CurrentTier(
		ctx context.Context,
		userId string,
	) (int, error)

	// TierAsOf は現在のシーズン内で、asOf 時点までの実績のみで判定した場合のtierを返す
	// (称号未達成なら0)。backfill-notifications が「実際に達成した日」を過去の
	// 記録から遡って特定するために使う特殊な用途で、通常のリアルタイム評価では使わない。
	TierAsOf(
		ctx context.Context,
		userId string,
		asOf time.Time,
	) (int, error)

	// NotifyIfTierChanged はbeforeTier(record保存前のtier)と現在のtierを比較し、
	// 上がっていれば称号獲得の通知を、ランクの区分もまたいでいればランクアップの
	// 通知も作成する。record作成自体を失敗させたくないため、内部のエラーは
	// 握りつぶす(戻り値なし)。achievedAt は通知のcreated_atに使う実際の処理時刻
	// (record/matchのCreatedAt)。event_dateのような入力値を渡すと、他の通知との
	// created_at基準がずれて並び順が崩れるため使わないこと。
	NotifyIfTierChanged(
		ctx context.Context,
		userId string,
		beforeTier int,
		achievedAt time.Time,
	)

	// NotifyIfTierLost はbeforeTier(record削除前のtier)と現在のtierを比較し、
	// 下がっていれば称号を失った通知を、ランクの区分もまたいでいればランクダウンの
	// 通知も作成する。record削除自体を失敗させたくないため、内部のエラーは
	// 握りつぶす(戻り値なし)。
	NotifyIfTierLost(
		ctx context.Context,
		userId string,
		beforeTier int,
	)
}

type DesignationEvaluation struct {
	designationRepo        repository.DesignationInterface
	designationStatsRepo   repository.DesignationStatsInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
	notificationRepo       repository.NotificationInterface
	userPlayerRepo         repository.UserPlayerInterface
}

func NewDesignationEvaluation(
	designationRepo repository.DesignationInterface,
	designationStatsRepo repository.DesignationStatsInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	notificationRepo repository.NotificationInterface,
	userPlayerRepo repository.UserPlayerInterface,
) DesignationEvaluationInterface {
	return &DesignationEvaluation{
		designationRepo:        designationRepo,
		designationStatsRepo:   designationStatsRepo,
		championshipSeriesRepo: championshipSeriesRepo,
		notificationRepo:       notificationRepo,
		userPlayerRepo:         userPlayerRepo,
	}
}

func (u *DesignationEvaluation) CurrentTier(
	ctx context.Context,
	userId string,
) (int, error) {
	def, _, _, err := u.currentDesignationForRecordCriteria(ctx, userId)
	if err != nil {
		return 0, err
	}
	if def == nil {
		return 0, nil
	}

	return def.Tier, nil
}

func (u *DesignationEvaluation) TierAsOf(
	ctx context.Context,
	userId string,
	asOf time.Time,
) (int, error) {
	def, _, _, err := u.currentDesignationForRecordCriteriaAsOf(ctx, userId, asOf, true)
	if err != nil {
		return 0, err
	}
	if def == nil {
		return 0, nil
	}

	return def.Tier, nil
}

func (u *DesignationEvaluation) NotifyIfTierChanged(
	ctx context.Context,
	userId string,
	beforeTier int,
	achievedAt time.Time,
) {
	def, definitions, seasonLabel, err := u.currentDesignationForRecordCriteria(ctx, userId)
	if err != nil || def == nil || def.Tier <= beforeTier {
		return
	}

	// 1回の評価でtierが複数段上がることがある(称号機能の導入時点で既に多数の記録が
	// あった場合等)。通過した各tierをすべて通知しないと、間のtierの通知が永久に
	// 欠落するため、beforeTierの次から現在のtierまで1段ずつ通知する。
	beforeRank := RankNameForTier(beforeTier)
	for tier := beforeTier + 1; tier <= def.Tier; tier++ {
		tierDef := designationForTier(definitions, tier)
		if tierDef == nil {
			continue
		}

		if err := u.notifyDesignationAchieved(ctx, userId, tierDef, seasonLabel, achievedAt); err != nil {
			continue
		}

		afterRank := RankNameForTier(tier)
		if afterRank != "" && afterRank != beforeRank {
			_ = u.notifyRankUp(ctx, userId, afterRank, seasonLabel, achievedAt)
			beforeRank = afterRank
		}
	}
}

func (u *DesignationEvaluation) NotifyIfTierLost(
	ctx context.Context,
	userId string,
	beforeTier int,
) {
	if beforeTier <= 0 {
		return
	}

	def, definitions, seasonLabel, err := u.currentDesignationForRecordCriteria(ctx, userId)
	if err != nil {
		return
	}

	afterTier := 0
	if def != nil {
		afterTier = def.Tier
	}
	if afterTier >= beforeTier {
		return
	}

	lostDef := designationForTier(definitions, beforeTier)
	if lostDef == nil {
		return
	}

	if err := u.notifyDesignationLost(ctx, userId, lostDef, seasonLabel); err != nil {
		return
	}

	beforeRank := RankNameForTier(beforeTier)
	afterRank := RankNameForTier(afterTier)
	if beforeRank != "" && beforeRank != afterRank {
		_ = u.notifyRankDown(ctx, userId, beforeRank, seasonLabel)
	}
}

// designationForTier はdefinitionsからtierが一致する称号定義を返す(無ければnil)。
func designationForTier(definitions []*entity.Designation, tier int) *entity.Designation {
	for _, def := range definitions {
		if def.Tier == tier {
			return def
		}
	}

	return nil
}

// currentDesignationForRecordCriteria はrecords起因の3criteria_typeのみを使い、
// usecase/designation.goのcurrentDesignation()で現在の到達tierを判定する。
// championship_seriesが見つからない等でシーズン範囲が定まらない場合はerrを返し、
// 呼び出し側(CurrentTier/NotifyIfTierChanged/NotifyIfTierLost)で評価自体をスキップする。
// 併せて称号定義一覧(NotifyIfTierLostが「失った称号」の名前を引くために使う)と、
// 通知本文にどのシーズンの実績かを明記するためのシーズンラベルも返す。
func (u *DesignationEvaluation) currentDesignationForRecordCriteria(
	ctx context.Context,
	userId string,
) (*entity.Designation, []*entity.Designation, string, error) {
	return u.currentDesignationForRecordCriteriaAsOf(ctx, userId, time.Time{}, true)
}

// currentDesignationForRecordCriteriaAsOf は currentDesignationForRecordCriteria と同様だが、
// asOf が非ゼロの場合はシーズン終了日ではなく asOf を集計期間の上限として扱う(TierAsOf専用)。
// asOf がゼロ値の場合はシーズン終了日まで(=currentDesignationForRecordCriteriaと同じ)集計する。
//
// includeCityLeagueResultCriteria が true の場合のみ、ベテラン(official_city_league_placement)・
// 熟練(official_city_league_playoff)・達人(official_city_league_champion)の3criteria_typeも
// valuesに含める。
//
// currentDesignation() は values に無い criteria_type に当たると判定を打ち切るため、この3つを
// 除外すると tier5 以降には決して到達できず、返る tier は最大4(レギュラー)で頭打ちになる。
// かつて CurrentTier/NotifyIfTierChanged/NotifyIfTierLost は false で呼んでいたが、その結果
// ベテラン・熟練のユーザーで beforeTier が 4 と誤計算され、称号の獲得・剥奪・ランクアップ/
// ランクダウン通知がいずれも飛ばなかった。表示側(usecase/designation.go)はこれらを含めて
// 判定するため、表示と通知で到達tierが食い違っていたのが原因。現在はリアルタイム評価・
// TierAsOf(backfill-notifications)ともに true で呼び、表示側と判定を揃えている。
func (u *DesignationEvaluation) currentDesignationForRecordCriteriaAsOf(
	ctx context.Context,
	userId string,
	asOf time.Time,
	includeCityLeagueResultCriteria bool,
) (*entity.Designation, []*entity.Designation, string, error) {
	definitions, err := u.designationRepo.FindAll(ctx)
	if err != nil {
		return nil, nil, "", err
	}

	now := time.Now().Local()

	fromDate, toDate, err := seasonRange(ctx, u.championshipSeriesRepo, "", now)
	if err != nil {
		return nil, nil, "", err
	}
	if !asOf.IsZero() && asOf.Before(toDate) {
		toDate = asOf
	}

	seasonLabel, err := CurrentSeasonLabel(ctx, u.championshipSeriesRepo, now)
	if err != nil {
		seasonLabel = ""
	}

	// TierAsOf(asOf非ゼロ)からの呼び出しのみ CountRecordsAsOfByUserId を使う。デッキ
	// 未登録のまま作成した記録に後からデッキを登録(使用したデッキとして編集)した
	// ケースで、通常のCountRecordsByUserIdだと「現在デッキが登録されているか」しか
	// 見ないため、実際にデッキが登録されるより前のasOfでも達成済みと誤判定してしまう。
	// CountRecordsAsOfByUserIdはrecords.updated_atも見て、まだ登録されていなかった
	// 記録を正しく除外する(詳細はDesignationStatsInterface.CountRecordsAsOfByUserId参照)。
	var recordCount int
	if !asOf.IsZero() {
		recordCount, err = u.designationStatsRepo.CountRecordsAsOfByUserId(ctx, userId, fromDate, toDate)
	} else {
		recordCount, err = u.designationStatsRepo.CountRecordsByUserId(ctx, userId, fromDate, toDate)
	}
	if err != nil {
		return nil, nil, "", err
	}

	leagueCount, err := u.designationStatsRepo.CountLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, nil, "", err
	}

	cityLeagueCount, err := u.designationStatsRepo.CountCityLeagueRecordsByUserId(ctx, userId, fromDate, toDate)
	if err != nil {
		return nil, nil, "", err
	}

	previousFromDate, previousToDate, err := previousSeasonRange(ctx, u.championshipSeriesRepo, "", now)
	if err != nil {
		return nil, nil, "", err
	}

	previousCityLeagueCount, err := u.designationStatsRepo.CountCityLeagueRecordsByUserId(ctx, userId, previousFromDate, previousToDate)
	if err != nil {
		return nil, nil, "", err
	}

	values := map[string]int{
		DesignationCriteriaTypeRecord:                   recordCount,
		DesignationCriteriaTypeOfficialLeagueRecord:     leagueCount,
		DesignationCriteriaTypeOfficialCityLeagueRecord: cityLeagueCount,
	}

	if includeCityLeagueResultCriteria {
		cityLeaguePlacement := 0
		cityLeagueFinalTournament := 0
		cityLeagueChampion := 0

		userPlayer, err := u.userPlayerRepo.FindByUserId(ctx, userId)
		if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
			return nil, nil, "", err
		}

		if userPlayer != nil {
			// asOf(TierAsOf経由)ではAsOf版を使う。「現在の状態」だけでは、cityleague_resultsの
			// 方が先に存在し、対応するuserId自身のrecordを後から作成したケースを、過去時点の
			// 判定で正しく除外できないため(ExistsCityLeagueResultAsOfByPlayerIdのコメント参照)。
			// 一方リアルタイム評価(asOfゼロ)では、表示側(usecase/designation.goの
			// seasonValuesByCriteriaType)と同じ非AsOf版を使う。表示と通知で別のクエリを使うと
			// 到達tierの判定が食い違い、表示上は称号が変わったのに通知だけ飛ばない(またはその逆)
			// という不整合が起きるため。
			var exists bool
			var existsFinalTournament bool

			if !asOf.IsZero() {
				exists, err = u.designationStatsRepo.ExistsCityLeagueResultAsOfByPlayerId(ctx, userId, userPlayer.PlayerId, fromDate, toDate)
			} else {
				exists, err = u.designationStatsRepo.ExistsCityLeagueResultByPlayerId(ctx, userId, userPlayer.PlayerId, fromDate, toDate)
			}
			if err != nil {
				return nil, nil, "", err
			}
			if exists {
				cityLeaguePlacement = 1
			}

			if !asOf.IsZero() {
				existsFinalTournament, err = u.designationStatsRepo.ExistsCityLeagueFinalTournamentResultAsOfByPlayerId(ctx, userId, userPlayer.PlayerId, DesignationCityLeagueFinalTournamentMaxRank, fromDate, toDate)
			} else {
				existsFinalTournament, err = u.designationStatsRepo.ExistsCityLeagueFinalTournamentResultByPlayerId(ctx, userId, userPlayer.PlayerId, DesignationCityLeagueFinalTournamentMaxRank, fromDate, toDate)
			}
			if err != nil {
				return nil, nil, "", err
			}
			if existsFinalTournament {
				cityLeagueFinalTournament = 1
			}

			// 達人(優勝=rank1)。熟練(決勝トーナメント進出=rank5以下)と同じメソッドを
			// しきい値 DesignationCityLeagueChampionMaxRank(=1)にして流用する。
			// asOf/非asOfの使い分けの理由は上の熟練・ベテランと同じ。
			var existsChampion bool
			if !asOf.IsZero() {
				existsChampion, err = u.designationStatsRepo.ExistsCityLeagueFinalTournamentResultAsOfByPlayerId(ctx, userId, userPlayer.PlayerId, DesignationCityLeagueChampionMaxRank, fromDate, toDate)
			} else {
				existsChampion, err = u.designationStatsRepo.ExistsCityLeagueFinalTournamentResultByPlayerId(ctx, userId, userPlayer.PlayerId, DesignationCityLeagueChampionMaxRank, fromDate, toDate)
			}
			if err != nil {
				return nil, nil, "", err
			}
			if existsChampion {
				cityLeagueChampion = 1
			}
		}

		values[DesignationCriteriaTypeOfficialCityLeaguePlacement] = cityLeaguePlacement
		values[DesignationCriteriaTypeOfficialCityLeagueFinalTournament] = cityLeagueFinalTournament
		values[DesignationCriteriaTypeOfficialCityLeagueChampion] = cityLeagueChampion
	}

	return currentDesignation(definitions, values, previousCityLeagueCount), definitions, seasonLabel, nil
}

func (u *DesignationEvaluation) notifyDesignationAchieved(
	ctx context.Context,
	userId string,
	def *entity.Designation,
	seasonLabel string,
	achievedAt time.Time,
) error {
	id, err := generateId()
	if err != nil {
		return err
	}

	body := fmt.Sprintf("称号「%s %s」を獲得しました！", def.Emoji, def.Name)
	if seasonLabel != "" {
		body = fmt.Sprintf("%sシーズンで称号「%s %s」を獲得しました！", seasonLabel, def.Emoji, def.Name)
	}

	notification := entity.NewNotification(
		id,
		achievedAt,
		userId,
		NotificationCategoryDesignation,
		"称号を獲得しました",
		body,
		notificationLinkUrlForDesignation,
	)

	return u.notificationRepo.Save(ctx, notification)
}

func (u *DesignationEvaluation) notifyRankUp(
	ctx context.Context,
	userId string,
	rankName string,
	seasonLabel string,
	achievedAt time.Time,
) error {
	id, err := generateId()
	if err != nil {
		return err
	}

	body := fmt.Sprintf("ランクが「%s」に上がりました！", rankName)
	if seasonLabel != "" {
		body = fmt.Sprintf("%sシーズンでランクが「%s」に上がりました！", seasonLabel, rankName)
	}

	notification := entity.NewNotification(
		id,
		achievedAt,
		userId,
		NotificationCategoryRank,
		"ランクが上がりました",
		body,
		notificationLinkUrlForDesignation,
	)

	return u.notificationRepo.Save(ctx, notification)
}

func (u *DesignationEvaluation) notifyDesignationLost(
	ctx context.Context,
	userId string,
	def *entity.Designation,
	seasonLabel string,
) error {
	id, err := generateId()
	if err != nil {
		return err
	}

	body := fmt.Sprintf("称号「%s %s」の条件を満たさなくなりました", def.Emoji, def.Name)
	if seasonLabel != "" {
		body = fmt.Sprintf("%sシーズンで称号「%s %s」の条件を満たさなくなりました", seasonLabel, def.Emoji, def.Name)
	}

	notification := entity.NewNotification(
		id,
		time.Now().Local(),
		userId,
		NotificationCategoryDesignation,
		"称号を失いました",
		body,
		notificationLinkUrlForDesignation,
	)

	return u.notificationRepo.Save(ctx, notification)
}

func (u *DesignationEvaluation) notifyRankDown(
	ctx context.Context,
	userId string,
	rankName string,
	seasonLabel string,
) error {
	id, err := generateId()
	if err != nil {
		return err
	}

	body := fmt.Sprintf("ランクが「%s」から下がりました", rankName)
	if seasonLabel != "" {
		body = fmt.Sprintf("%sシーズンでランクが「%s」から下がりました", seasonLabel, rankName)
	}

	notification := entity.NewNotification(
		id,
		time.Now().Local(),
		userId,
		NotificationCategoryRank,
		"ランクが下がりました",
		body,
		notificationLinkUrlForDesignation,
	)

	return u.notificationRepo.Save(ctx, notification)
}
