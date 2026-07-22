package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type Kizuna struct {
	db *gorm.DB
}

func NewKizuna(
	db *gorm.DB,
) repository.KizunaInterface {
	return &Kizuna{db}
}

/*
 * きずなLv.の素材をデッキごとに集計する。
 *
 * デッキ数ぶんクエリを撃つ（N+1）とデッキを多く持つ人ほど遅くなるため、
 * すべて「ユーザー単位で1回引いて deck_id で GROUP BY」する方針で統一する。
 * デッキが100個あってもクエリは5本で固定になる。
 *
 * 点数付け（舞台の格の重み・正規化・重み配分）は SQL に一切書かない。
 * ここでやるのは事実の集計だけで、方針は entity.CalculateKizuna が持つ。
 * SQL に点数を書くと、算出方法を調整するたびにクエリを触ることになる。
 */

/*
 * きずなLv.の集計で共通して除外する記録の条件。
 *
 * 除くのは論理削除済みの記録だけで、集計対象外(ignore_stats_flg)の記録も数える。
 * deck_usage_stat など勝率まわりの集計とは、ここだけ方針が違う。
 *
 * 集計対象外は「この記録で勝率を歪めたくない」という意思表示であって、
 * そのデッキを連れて行った事実まで取り消すものではない。きずなは勝率を一切見ず
 * 「どう歩んできたか」だけを見る指標なので、除外する理由がない
 * （むしろ、握った回数が多いデッキほど集計対象外の記録も溜まりやすく、
 * 除外すると一緒に過ごした分だけ数値が下がるという逆転が起きる）。
 */
const kizunaRecordCondition = "records.user_id = ? AND records.deleted_at IS NULL AND records.deck_id IS NOT NULL AND records.deck_id != ''"

/*
 * 記録を「どの格の舞台か」に分類する式。
 *
 * 数値は entity.KizunaStageKind と対応させること（entity 側の定数と一致することは
 * TestKizunaStageKindSQLLiterals が検証している）。
 *   1,2,3,4,6,7 … official_events.type_id をそのまま使う
 *   90 … official_event_id はあるが type_id を引けなかった公式イベント
 *   91 … Tonamel の大会
 *   92 … 自主大会
 *   0  … イベントに紐づかない対戦
 */
const kizunaStageKindExpr = `CASE
	WHEN records.official_event_id IS NOT NULL AND records.official_event_id > 0 THEN
		CASE WHEN official_events.type_id IN (1,2,3,4,6,7) THEN official_events.type_id ELSE 90 END
	WHEN records.tonamel_event_id IS NOT NULL AND records.tonamel_event_id != '' THEN 91
	WHEN records.unofficial_event_id IS NOT NULL AND records.unofficial_event_id != '' THEN 92
	ELSE 0
END`

/*
 * 大会に向けた調整とみなす時間帯：開催日の3日前から、当日の昼まで。
 *
 * event_date は日付（その日の0時）なので、「前夜だけ」を見ると当日朝の調整を
 * 取り逃がす。幅を持たせて当日の午前も含める。
 */
const (
	kizunaEveWindow      = "72 hours"
	kizunaEventDayWindow = "12 hours"
)

type kizunaRecordResult struct {
	DeckId          string
	EventDayCount   int
	RecordCount     int
	MemoCount       int
	MemoTotalLength int
}

type kizunaStageResult struct {
	DeckId    string
	StageKind int
	Count     int
}

type kizunaDeckCodeResult struct {
	DeckId    string
	CodeCount int
}

type kizunaEveResult struct {
	DeckId   string
	EveCount int
}

type kizunaMatchResult struct {
	DeckId     string
	MatchCount int
	Wins       int
}

func (i *Kizuna) FindKizunaDeckAggregates(
	ctx context.Context,
	userId string,
) ([]*entity.KizunaDeckAggregate, error) {
	// 記録が1件も無いデッキも「出会ったばかり」として返したいので、
	// 集計結果ではなくデッキ一覧を軸にする。
	var deckIds []string
	if tx := i.db.WithContext(ctx).
		Table("decks").
		Where("user_id = ? AND deleted_at IS NULL", userId).
		Order("created_at ASC").
		Pluck("id", &deckIds); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	if len(deckIds) == 0 {
		return []*entity.KizunaDeckAggregate{}, nil
	}

	aggregates := make(map[string]*entity.KizunaDeckAggregate, len(deckIds))
	for _, deckId := range deckIds {
		aggregates[deckId] = &entity.KizunaDeckAggregate{
			DeckId:      deckId,
			StageCounts: map[entity.KizunaStageKind]int{},
		}
	}

	// ── 同行日数・語り度：記録そのものの集計
	// メモは空文字と NULL の両方がありうるため、TRIM して長さで判定する。
	var recordResults []kizunaRecordResult
	if tx := i.db.WithContext(ctx).
		Table("records").
		Select("records.deck_id AS deck_id, "+
			"COUNT(DISTINCT records.event_date) AS event_day_count, "+
			"COUNT(*) AS record_count, "+
			"COUNT(CASE WHEN COALESCE(TRIM(records.memo), '') != '' THEN 1 END) AS memo_count, "+
			"COALESCE(SUM(CHAR_LENGTH(COALESCE(TRIM(records.memo), ''))), 0) AS memo_total_length").
		Where(kizunaRecordCondition, userId).
		Group("records.deck_id").
		Scan(&recordResults); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	for _, r := range recordResults {
		if a, ok := aggregates[r.DeckId]; ok {
			a.EventDayCount = r.EventDayCount
			a.RecordCount = r.RecordCount
			a.MemoCount = r.MemoCount
			a.MemoTotalLength = r.MemoTotalLength
		}
	}

	// ── 託し度：どの格の舞台に何回連れて行ったか
	var stageResults []kizunaStageResult
	if tx := i.db.WithContext(ctx).
		Table("records").
		Select("records.deck_id AS deck_id, "+kizunaStageKindExpr+" AS stage_kind, COUNT(*) AS count").
		Joins("LEFT JOIN official_events ON official_events.id = records.official_event_id").
		Where(kizunaRecordCondition, userId).
		Group("records.deck_id, stage_kind").
		Scan(&stageResults); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	for _, r := range stageResults {
		if a, ok := aggregates[r.DeckId]; ok {
			a.StageCounts[entity.KizunaStageKind(r.StageKind)] += r.Count
		}
	}

	// ── 手入れ度その1：デッキコードの総数（登録時の1件を含む）
	var codeResults []kizunaDeckCodeResult
	if tx := i.db.WithContext(ctx).
		Table("deck_codes").
		Select("deck_codes.deck_id AS deck_id, COUNT(*) AS code_count").
		Where("deck_codes.user_id = ? AND deck_codes.deleted_at IS NULL", userId).
		Group("deck_codes.deck_id").
		Scan(&codeResults); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	for _, r := range codeResults {
		if a, ok := aggregates[r.DeckId]; ok {
			a.DeckCodeCount = r.CodeCount
		}
	}

	// ── 手入れ度その2：大会に向けた調整とみなせるデッキコードの数。
	// 同じコードが複数の大会の窓に入りうるため DISTINCT で数える。
	//
	// JOIN 側にも records.user_id を必ず入れること。deck_codes を本人に絞れば
	// deck_id 経由で自然に絞られるように見えるが、プランナはそれを導けず
	// records を全件走査してから結合する。実測で 305ms → 23ms（記録10万件時）。
	// この差は「サービス全体の記録数」に比例して開くため、放置してはならない。
	var eveResults []kizunaEveResult
	if tx := i.db.WithContext(ctx).
		Table("deck_codes").
		Select("deck_codes.deck_id AS deck_id, COUNT(DISTINCT deck_codes.id) AS eve_count").
		Joins("JOIN records ON records.deck_id = deck_codes.deck_id "+
			"AND records.user_id = ? "+
			"AND records.deleted_at IS NULL "+
			"AND records.event_date IS NOT NULL "+
			"AND deck_codes.created_at >= records.event_date - INTERVAL '"+kizunaEveWindow+"' "+
			"AND deck_codes.created_at <= records.event_date + INTERVAL '"+kizunaEventDayWindow+"'",
			userId).
		Where("deck_codes.user_id = ? AND deck_codes.deleted_at IS NULL", userId).
		Group("deck_codes.deck_id").
		Scan(&eveResults); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	for _, r := range eveResults {
		if a, ok := aggregates[r.DeckId]; ok {
			a.EveCodeCount = r.EveCount
		}
	}

	// ── 逆境ロイヤルティ・一途度：対戦数と勝数
	//
	// デッキへの紐付けは records.deck_id を正とする（matches.deck_id は記録後の
	// デッキ変更に追随しないため使わない。deck_usage_stat と同じ方針）。
	//
	// DISTINCT は付けない。games を結合していないので、records.id が主キーである以上
	// 1つの matches 行は高々1つの records 行としか結合せず、行は増えないため
	// （deck_usage_stat が DISTINCT を要するのは games を LEFT JOIN しているから）。
	//
	// matches.user_id で先に絞る。records 側だけを絞っても、プランナは matches を
	// 全件走査してから結合するため、サービス全体の対戦数に比例して遅くなる。
	// matches.user_id は記録の所有者と同じ値が入る（usecase.Match が認証ユーザーから設定する）。
	var matchResults []kizunaMatchResult
	if tx := i.db.WithContext(ctx).
		Table("matches").
		Select("records.deck_id AS deck_id, "+
			"COUNT(matches.id) AS match_count, "+
			"COUNT(CASE WHEN matches.victory_flg THEN 1 END) AS wins").
		Joins("JOIN records ON matches.record_id = records.id").
		Where(kizunaRecordCondition+" AND matches.user_id = ? AND matches.deleted_at IS NULL",
			userId, userId).
		Group("records.deck_id").
		Scan(&matchResults); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	for _, r := range matchResults {
		if a, ok := aggregates[r.DeckId]; ok {
			a.MatchCount = r.MatchCount
			a.Wins = r.Wins
		}
	}

	// デッキの登録順で返す（呼び出し側の並び替えに依存させない）
	ret := make([]*entity.KizunaDeckAggregate, 0, len(deckIds))
	for _, deckId := range deckIds {
		ret = append(ret, aggregates[deckId])
	}

	return ret, nil
}
