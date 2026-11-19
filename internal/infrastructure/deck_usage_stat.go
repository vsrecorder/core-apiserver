package infrastructure

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type DeckUsageStat struct {
	db *gorm.DB
}

func NewDeckUsageStat(
	db *gorm.DB,
) repository.DeckUsageStatInterface {
	return &DeckUsageStat{db}
}

type deckUsageResult struct {
	DeckId string
	// DeckCodeId は記録が使ったバージョン。スプライトの組み合わせごとに行を分ける集計
	// (splitBySprites)のときだけ取得する。バージョンが紐付いていない記録は空。
	DeckCodeId   string
	Name         string
	Count        int
	Wins         int
	Draws        int
	GameCount    int
	GoFirstCount int
	GoFirstWins  int
	GoSecondWins int
}

// deckIgnoredResult は集計対象外(ignore_stats_flg=true)の記録件数をデッキごとに表す。
type deckIgnoredResult struct {
	DeckId       string
	Name         string
	IgnoredCount int
}

// deckDefaultMatchResult は不戦勝/不戦敗として集計から外した対戦数をデッキごとに表す。
type deckDefaultMatchResult struct {
	DeckId            string
	Name              string
	DefaultMatchCount int
}

func (i *DeckUsageStat) FindDeckUsageStat(
	ctx context.Context,
	userId string,
	period repository.StatPeriod,
	regulationId uint,
	excludeDefaultMatches bool,
) (*entity.DeckUsageStat, error) {
	return i.findDeckUsageStat(ctx, userId, period, regulationId, excludeDefaultMatches, false)
}

func (i *DeckUsageStat) FindDeckUsageStatBySprites(
	ctx context.Context,
	userId string,
	period repository.StatPeriod,
	regulationId uint,
	excludeDefaultMatches bool,
) (*entity.DeckUsageStat, error) {
	return i.findDeckUsageStat(ctx, userId, period, regulationId, excludeDefaultMatches, true)
}

// findDeckUsageStat はデッキごとの使用率を集計する。
//
// splitBySprites が false なら1デッキ1行(行のアイコンは期間中にいちばん多く使ったスプライト)。
// true なら、同じデッキでも対戦で使ったスプライトの組み合わせごとに行を分ける
// (mergeDeckUsageBySprites)。同じデッキ名のまま相棒を入れ替えた前後の成績を分けて見るため。
func (i *DeckUsageStat) findDeckUsageStat(
	ctx context.Context,
	userId string,
	period repository.StatPeriod,
	regulationId uint,
	excludeDefaultMatches bool,
	splitBySprites bool,
) (*entity.DeckUsageStat, error) {
	var results []deckUsageResult

	// 対戦(matches)をデッキごとに集計する。records.deck_id を正としてデッキに紐付ける。
	// matches.deck_id は記録後にデッキを変更しても更新されないため使用しない
	// （opponent_deck_usage_stat.go と同様の方針）。
	// デッキが削除済みでも名称を表示できるよう decks.deleted_at は条件に含めない。
	// games は1対戦(match)につき複数行になりうる（BO3）ため、count/winsは
	// matches.id のDISTINCTで数え、先攻/後攻はgames行をそのままカウントする。
	// スプライトの組み合わせごとに分けるときは、いったんバージョン単位で集計し、あとで
	// 同じ組み合わせのバージョンを足し合わせる(1つの対戦は1つの記録＝1つのバージョンにしか
	// 属さないので、どの列もバージョンをまたいで単純に足せる)。
	versionSelect, versionGroup := "", ""
	if splitBySprites {
		versionSelect = "COALESCE(records.deck_code_id, '') AS deck_code_id, "
		versionGroup = ", COALESCE(records.deck_code_id, '')"
	}

	query := i.db.Table("matches").
		Select("records.deck_id AS deck_id, "+versionSelect+"COALESCE(decks.name, '') AS name, COUNT(DISTINCT matches.id) AS count, COUNT(DISTINCT CASE WHEN matches.victory_flg THEN matches.id END) AS wins, COUNT(DISTINCT CASE WHEN matches.draw_flg THEN matches.id END) AS draws, COUNT(games.id) AS game_count, SUM(CASE WHEN games.go_first THEN 1 ELSE 0 END) AS go_first_count, SUM(CASE WHEN games.go_first AND games.winning_flg THEN 1 ELSE 0 END) AS go_first_wins, SUM(CASE WHEN games.go_first = false AND games.winning_flg THEN 1 ELSE 0 END) AS go_second_wins").
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("LEFT JOIN decks ON records.deck_id = decks.id").
		Joins("LEFT JOIN games ON games.match_id = matches.id AND games.deleted_at IS NULL").
		Where("records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND records.deck_id != ''", userId)

	// レギュレーション(スタンダード/エクストラ/殿堂)での絞り込み。0 は絞り込みなし。
	if regulationId != 0 {
		query = query.Where("records.regulation_id = ?", regulationId)
	}

	// 不戦勝/不戦敗の除外は対戦(matches)の集計にだけ効かせる。下の集計対象外
	// (ignore_stats_flg=true)の件数は記録単位の数え上げで、対戦の有無とは無関係なため触らない。
	// 先攻/後攻(games)は不戦には行が存在しない(entity.Match の検証で禁じている)ので、
	// ここで matches を絞れば games 側も自動的に外れる。
	query = applyExcludeDefaultMatches(query, excludeDefaultMatches)

	query = applyStatPeriod(query, period)

	query = query.Group("records.deck_id, decks.name" + versionGroup).Order("count DESC")

	if tx := query.Scan(&results); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	totalMatches := 0
	for _, r := range results {
		totalMatches += r.Count
	}

	// 集計対象外(ignore_stats_flg=true)の記録件数をデッキごとに集計する。
	// 上の集計と同じ期間条件・ユーザー条件で、除外された記録の件数のみを数える。
	ignoredQuery := i.db.Table("records").
		Select("records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT records.id) AS ignored_count").
		Joins("LEFT JOIN decks ON records.deck_id = decks.id").
		Where("records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = true AND records.deck_id != ''", userId)

	if regulationId != 0 {
		ignoredQuery = ignoredQuery.Where("records.regulation_id = ?", regulationId)
	}

	ignoredQuery = applyStatPeriod(ignoredQuery, period)

	ignoredQuery = ignoredQuery.Group("records.deck_id, decks.name").Order("ignored_count DESC")

	var ignoredResults []deckIgnoredResult
	if tx := ignoredQuery.Scan(&ignoredResults); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ignoredMap := make(map[string]int, len(ignoredResults))
	for _, r := range ignoredResults {
		ignoredMap[r.DeckId] = r.IgnoredCount
	}

	// 除外したぶんの不戦勝/不戦敗をデッキごとに数える。上の集計と同じ条件で、
	// 不戦だけを数え直す(除外していないときは Count に含まれているので数えない)。
	// これが無いと、不戦しか記録が無いデッキが「対戦記録が無いデッキ」として
	// 一覧から消え、画面に「まだ対戦記録がありません」と出てしまう。
	var defaultResults []deckDefaultMatchResult
	if excludeDefaultMatches {
		defaultQuery := i.db.Table("matches").
			Select("records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT matches.id) AS default_match_count").
			Joins("JOIN records ON matches.record_id = records.id").
			Joins("LEFT JOIN decks ON records.deck_id = decks.id").
			Where(
				"records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND records.deck_id != ''"+
					" AND (matches.default_victory_flg = true OR matches.default_defeat_flg = true)",
				userId,
			)

		if regulationId != 0 {
			defaultQuery = defaultQuery.Where("records.regulation_id = ?", regulationId)
		}

		defaultQuery = applyStatPeriod(defaultQuery, period)

		defaultQuery = defaultQuery.Group("records.deck_id, decks.name").Order("default_match_count DESC")

		if tx := defaultQuery.Scan(&defaultResults); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	}

	defaultMap := make(map[string]int, len(defaultResults))
	for _, r := range defaultResults {
		defaultMap[r.DeckId] = r.DefaultMatchCount
	}

	// 集計対象の対戦を持たないデッキ(集計対象外のみ・不戦のみ)を一覧に加えるのは
	// 全期間集計のときだけ。期間を指定する画面では使用率0のデッキが並んで邪魔になる。
	isAllTime := period.From.IsZero() && period.To.IsZero()

	// 集計対象の記録があるデッキの deck_id 集合。集計対象外のみのデッキを
	// 後段で追加する際に、重複を避けるために使う。
	seen := make(map[string]bool, len(results))

	// スプライトはデッキごとに引くとデッキ数に比例してクエリが増える(N+1)ため、
	// 後段のループで使う分をここで1クエリにまとめて取得しておく。
	// 集計対象外のみ・不戦のみのデッキは全期間集計のときだけ一覧に加わるので、その場合だけ含める。
	// 同じデッキが複数の結果に現れるため、IDは重複を除いて渡す。
	spriteDeckIds := make([]string, 0, len(results)+len(ignoredResults)+len(defaultResults))
	spriteDeckIdSeen := make(map[string]struct{}, len(results)+len(ignoredResults)+len(defaultResults))

	appendSpriteDeckId := func(deckId string) {
		if _, ok := spriteDeckIdSeen[deckId]; ok {
			return
		}
		spriteDeckIdSeen[deckId] = struct{}{}
		spriteDeckIds = append(spriteDeckIds, deckId)
	}

	for _, r := range results {
		appendSpriteDeckId(r.DeckId)
	}
	if isAllTime {
		for _, r := range ignoredResults {
			appendSpriteDeckId(r.DeckId)
		}
		for _, r := range defaultResults {
			appendSpriteDeckId(r.DeckId)
		}
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, spriteDeckIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// 各行に出すスプライト(results と同じ並び)。
	var rowSprites [][]*entity.PokemonSprite
	if splitBySprites {
		// バージョン単位の行を、スプライトの組み合わせごとの行へまとめ直す。
		deckCodeIds := make([]string, 0, len(results))
		for _, r := range results {
			if r.DeckCodeId != "" {
				deckCodeIds = append(deckCodeIds, r.DeckCodeId)
			}
		}

		spritesByDeckCodeId, err := findDeckCodePokemonSpritesByDeckCodeIds(ctx, i.db, deckCodeIds)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}

		results, rowSprites = mergeDeckUsageBySprites(results, spritesByDeckId, spritesByDeckCodeId)
	} else {
		// 行のアイコンは、その期間に実際に使ったバージョンのスプライトから選ぶ。
		representativeSprites, err := i.findRepresentativeSprites(
			ctx, userId, period, regulationId, excludeDefaultMatches, results, spritesByDeckId,
		)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}

		rowSprites = make([][]*entity.PokemonSprite, 0, len(results))
		for _, r := range results {
			rowSprites = append(rowSprites, representativeSprites[r.DeckId])
		}
	}

	decks := []*entity.DeckUsage{}
	for index, r := range results {
		// 集計対象外・不戦の件数はデッキ単位の数なので、行を分けたときは、そのデッキの
		// 先頭の行(対戦数がいちばん多い組み合わせ)にだけ付ける(複数の行に付けると二重に数えられる)。
		firstRowOfDeck := !seen[r.DeckId]
		seen[r.DeckId] = true
		pokemonSprites := rowSprites[index]

		name := r.Name

		var usageRate float64
		if totalMatches > 0 {
			usageRate = float64(r.Count) / float64(totalMatches)
		}

		// 引き分けは負けに数えない。勝率も分母から除外する(勝ち/(勝ち+負け))。
		losses := r.Count - r.Wins - r.Draws
		var winRate float64
		if decided := r.Wins + losses; decided > 0 {
			winRate = float64(r.Wins) / float64(decided)
		}

		goSecondCount := r.GameCount - r.GoFirstCount
		var goFirstRate float64
		if r.GameCount > 0 {
			goFirstRate = float64(r.GoFirstCount) / float64(r.GameCount)
		}

		var goFirstWinRate float64
		if r.GoFirstCount > 0 {
			goFirstWinRate = float64(r.GoFirstWins) / float64(r.GoFirstCount)
		}

		var goSecondWinRate float64
		if goSecondCount > 0 {
			goSecondWinRate = float64(r.GoSecondWins) / float64(goSecondCount)
		}

		deckUsage := entity.NewDeckUsage(
			r.DeckId, name, r.Count, usageRate, r.Wins, losses, winRate,
			r.GameCount, r.GoFirstCount, goSecondCount, goFirstRate,
			r.GoFirstWins, goFirstWinRate, r.GoSecondWins, goSecondWinRate,
			pokemonSprites,
		)
		deckUsage.DeckPokemonSprites = spritesByDeckId[r.DeckId]
		if firstRowOfDeck {
			deckUsage.IgnoredCount = ignoredMap[r.DeckId]
			deckUsage.DefaultMatchCount = defaultMap[r.DeckId]
		}
		decks = append(decks, deckUsage)
	}

	// 全期間集計(all_time)の場合のみ、集計対象の対戦を持たないデッキ
	// (集計対象外の記録しか無い・不戦しか無い)も一覧に含める。デッキ一覧カードで
	// 「集計対象外の記録がN件ある」「不戦勝・不戦敗がN件ある」と示すためで、
	// count=0 のため使用率ランキング等では他の画面（期間指定）には現れない。
	if isAllTime {
		for _, r := range ignoredResults {
			if seen[r.DeckId] {
				continue
			}
			seen[r.DeckId] = true

			deckUsage := entity.NewDeckUsage(
				r.DeckId, r.Name, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, spritesByDeckId[r.DeckId],
			)
			deckUsage.DeckPokemonSprites = spritesByDeckId[r.DeckId]
			deckUsage.IgnoredCount = r.IgnoredCount
			deckUsage.DefaultMatchCount = defaultMap[r.DeckId]
			decks = append(decks, deckUsage)
		}

		for _, r := range defaultResults {
			if seen[r.DeckId] {
				continue
			}
			seen[r.DeckId] = true

			deckUsage := entity.NewDeckUsage(
				r.DeckId, r.Name, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, spritesByDeckId[r.DeckId],
			)
			deckUsage.DeckPokemonSprites = spritesByDeckId[r.DeckId]
			deckUsage.IgnoredCount = ignoredMap[r.DeckId]
			deckUsage.DefaultMatchCount = r.DefaultMatchCount
			decks = append(decks, deckUsage)
		}
	}

	return entity.NewDeckUsageStat(userId, totalMatches, decks), nil
}

// mergeDeckUsageBySprites は、バージョン単位で集計した行を「デッキ × スプライトの組み合わせ」の
// 行へまとめ直す。戻り値は、まとめた行と、それぞれの行に出すスプライト(同じ並び)。
//
//   - 行のスプライトは、記録が使ったバージョンのもの。バージョンが紐付いていない・未設定の
//     行は、デッキのスプライトを使ったものとして扱う(画面の表示と同じ規則)
//   - 同じデッキで同じ組み合わせ(枠まで同じ)の行は足し合わせる。メモだけ違うバージョンを
//     作り直しても行は増えない
//   - 並びは対戦数の多い順。同数は元の並び(SQL の対戦数順)を保つ
func mergeDeckUsageBySprites(
	results []deckUsageResult,
	spritesByDeckId map[string][]*entity.PokemonSprite,
	spritesByDeckCodeId map[string][]*entity.PokemonSprite,
) ([]deckUsageResult, [][]*entity.PokemonSprite) {
	type mergedRow struct {
		result  deckUsageResult
		sprites []*entity.PokemonSprite
	}

	rows := make([]*mergedRow, 0, len(results))
	byKey := make(map[string]*mergedRow, len(results))

	for _, r := range results {
		sprites := spritesByDeckCodeId[r.DeckCodeId]
		if len(sprites) == 0 {
			sprites = spritesByDeckId[r.DeckId]
		}

		key := r.DeckId + "|" + spriteLayoutKey(sprites)
		row, ok := byKey[key]
		if !ok {
			merged := r
			merged.DeckCodeId = ""
			row = &mergedRow{result: merged, sprites: sprites}
			byKey[key] = row
			rows = append(rows, row)
			continue
		}

		row.result.Count += r.Count
		row.result.Wins += r.Wins
		row.result.Draws += r.Draws
		row.result.GameCount += r.GameCount
		row.result.GoFirstCount += r.GoFirstCount
		row.result.GoFirstWins += r.GoFirstWins
		row.result.GoSecondWins += r.GoSecondWins
	}

	sort.SliceStable(rows, func(a, b int) bool {
		return rows[a].result.Count > rows[b].result.Count
	})

	merged := make([]deckUsageResult, 0, len(rows))
	sprites := make([][]*entity.PokemonSprite, 0, len(rows))
	for _, row := range rows {
		merged = append(merged, row.result)
		sprites = append(sprites, row.sprites)
	}

	return merged, sprites
}

// deckVersionCountResult はデッキ×バージョン(デッキコード)ごとの対戦数。
type deckVersionCountResult struct {
	DeckId     string
	DeckCodeId string
	Count      int
}

// spriteLayoutKey はスプライトの並び(枠とID)を比べるためのキーを返す。
func spriteLayoutKey(sprites []*entity.PokemonSprite) string {
	parts := make([]string, 0, len(sprites))
	for index, sprite := range sprites {
		position := sprite.Position
		if position == 0 {
			position = uint(index + 1)
		}
		parts = append(parts, fmt.Sprintf("%d:%s", position, sprite.ID))
	}
	sort.Strings(parts)

	return strings.Join(parts, ",")
}

// findRepresentativeSprites は、使用率の各行(デッキ)に出すスプライトをデッキごとに返す。
//
// 行はデッキ単位だが、スプライトはバージョンごとに持てる(同じデッキ名のまま相棒を入れ替える)。
// デッキのスプライトは最新バージョンに合わせて変わっていく(usecase/deck_sprite_sync.go)ので、
// デッキのものを出すと、過去の月の集計にも「いまの相棒」が出てしまう。そこで、その期間の
// 対戦でいちばん多く使ったスプライトの組み合わせを行のアイコンにする。
//
//   - 対戦ごとに、記録が使ったバージョンのスプライトを見る。バージョンが紐付いていない・
//     未設定の対戦は、デッキのスプライトを使ったものとして数える(画面の表示と同じ規則)
//   - 同じ組み合わせのバージョンは合算する(メモだけ違うバージョンを作り直した場合など)
//   - 同数なら、いまのデッキのスプライト(＝最新バージョン)を優先する。それも当たらなければ
//     並びのキーの昇順で決める(呼び出しのたびに結果が変わらないように)
//
// results は集計済みの行(デッキごとの対戦数)。条件はその集計と同じものを渡すこと。
// 集計対象の対戦が無いデッキ(results に無いもの)は返さない(呼び出し側がデッキのものを使う)。
func (i *DeckUsageStat) findRepresentativeSprites(
	ctx context.Context,
	userId string,
	period repository.StatPeriod,
	regulationId uint,
	excludeDefaultMatches bool,
	results []deckUsageResult,
	spritesByDeckId map[string][]*entity.PokemonSprite,
) (map[string][]*entity.PokemonSprite, error) {
	representative := make(map[string][]*entity.PokemonSprite, len(results))
	if len(results) == 0 {
		return representative, nil
	}

	// バージョンが紐付いた対戦を、デッキ×バージョンで数える(条件は行の集計と同じ)。
	query := i.db.Table("matches").
		Select("records.deck_id AS deck_id, records.deck_code_id AS deck_code_id, COUNT(DISTINCT matches.id) AS count").
		Joins("JOIN records ON matches.record_id = records.id").
		Where("records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND records.deck_id != '' AND records.deck_code_id <> ''", userId)

	if regulationId != 0 {
		query = query.Where("records.regulation_id = ?", regulationId)
	}

	query = applyExcludeDefaultMatches(query, excludeDefaultMatches)
	query = applyStatPeriod(query, period)
	query = query.Group("records.deck_id, records.deck_code_id")

	var versionCounts []deckVersionCountResult
	if tx := query.Scan(&versionCounts); tx.Error != nil {
		return nil, tx.Error
	}

	deckCodeIds := make([]string, 0, len(versionCounts))
	for _, v := range versionCounts {
		deckCodeIds = append(deckCodeIds, v.DeckCodeId)
	}

	spritesByDeckCodeId, err := findDeckCodePokemonSpritesByDeckCodeIds(ctx, i.db, deckCodeIds)
	if err != nil {
		return nil, err
	}

	type layoutTally struct {
		sprites []*entity.PokemonSprite
		count   int
	}

	// デッキごとに、スプライトの並び → 対戦数。
	tallies := make(map[string]map[string]*layoutTally, len(results))
	// バージョンのスプライトで数えた対戦数(残りはデッキのスプライトを使った対戦)。
	versionedCounts := make(map[string]int, len(results))

	add := func(deckId string, sprites []*entity.PokemonSprite, count int) {
		if len(sprites) == 0 || count <= 0 {
			return
		}

		byLayout, ok := tallies[deckId]
		if !ok {
			byLayout = make(map[string]*layoutTally)
			tallies[deckId] = byLayout
		}

		key := spriteLayoutKey(sprites)
		tally, ok := byLayout[key]
		if !ok {
			tally = &layoutTally{sprites: sprites}
			byLayout[key] = tally
		}
		tally.count += count
	}

	for _, v := range versionCounts {
		sprites := spritesByDeckCodeId[v.DeckCodeId]
		if len(sprites) == 0 {
			continue
		}
		add(v.DeckId, sprites, v.Count)
		versionedCounts[v.DeckId] += v.Count
	}

	for _, r := range results {
		deckSprites := spritesByDeckId[r.DeckId]
		add(r.DeckId, deckSprites, r.Count-versionedCounts[r.DeckId])

		// 対戦数の多い順。同数はいまのデッキのスプライトを優先し、それ以外どうしはキーの昇順。
		deckKey := spriteLayoutKey(deckSprites)
		var bestTally *layoutTally
		bestKey := ""
		for key, tally := range tallies[r.DeckId] {
			if bestTally == nil || tally.count > bestTally.count ||
				(tally.count == bestTally.count && bestKey != deckKey && (key == deckKey || key < bestKey)) {
				bestTally, bestKey = tally, key
			}
		}

		// どの対戦にもスプライトが無ければ、デッキのもの(空)のまま返す。
		best := deckSprites
		if bestTally != nil {
			best = bestTally.sprites
		}

		representative[r.DeckId] = best
	}

	return representative, nil
}

// deckCodeUsageResult はバージョン(デッキコード)ごとの対戦成績の集計行。
type deckCodeUsageResult struct {
	DeckCodeId string
	Count      int
	Wins       int
	Draws      int
}

func (i *DeckUsageStat) FindDeckCodeUsageStat(
	ctx context.Context,
	userId string,
	deckId string,
	excludeDefaultMatches bool,
) (*entity.DeckCodeUsageStat, error) {
	var results []deckCodeUsageResult

	// FindDeckUsageStat と同じく records 側の deck_id / deck_code_id を正とする
	// (matches 側は記録後にデッキを変更しても更新されないため)。
	// user_id を必ず条件に含めるのは、デッキIDだけで絞ると他人の非公開記録まで数えてしまうため。
	// 勝敗は対戦(matches)単位で数えればよく、先攻/後攻は使わないので games は結合しない。
	query := i.db.Table("matches").
		Select("COALESCE(records.deck_code_id, '') AS deck_code_id, COUNT(matches.id) AS count, SUM(CASE WHEN matches.victory_flg THEN 1 ELSE 0 END) AS wins, SUM(CASE WHEN matches.draw_flg THEN 1 ELSE 0 END) AS draws").
		Joins("JOIN records ON matches.record_id = records.id").
		Where("records.user_id = ? AND records.deck_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL", userId, deckId)

	query = applyExcludeDefaultMatches(query, excludeDefaultMatches)

	query = query.Group("COALESCE(records.deck_code_id, '')")

	if tx := query.Scan(&results); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	stat := &entity.DeckCodeUsageStat{
		UserId:    userId,
		DeckId:    deckId,
		DeckCodes: []*entity.DeckCodeUsage{},
	}

	for _, r := range results {
		// バージョン未指定の記録はどのバージョンにも振り分けられないので件数だけ返す
		if r.DeckCodeId == "" {
			stat.UnassignedCount += r.Count
			continue
		}

		stat.DeckCodes = append(stat.DeckCodes, entity.NewDeckCodeUsage(r.DeckCodeId, r.Count, r.Wins, r.Draws))
	}

	return stat, nil
}
