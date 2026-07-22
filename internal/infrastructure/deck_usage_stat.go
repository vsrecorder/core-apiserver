package infrastructure

import (
	"context"
	"time"

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
	DeckId       string
	Name         string
	Count        int
	Wins         int
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

func (i *DeckUsageStat) FindDeckUsageStat(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (*entity.DeckUsageStat, error) {
	var results []deckUsageResult

	// 対戦(matches)をデッキごとに集計する。records.deck_id を正としてデッキに紐付ける。
	// matches.deck_id は記録後にデッキを変更しても更新されないため使用しない
	// （opponent_deck_usage_stat.go と同様の方針）。
	// デッキが削除済みでも名称を表示できるよう decks.deleted_at は条件に含めない。
	// games は1対戦(match)につき複数行になりうる（BO3）ため、count/winsは
	// matches.id のDISTINCTで数え、先攻/後攻はgames行をそのままカウントする。
	query := i.db.Table("matches").
		Select("records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(DISTINCT matches.id) AS count, COUNT(DISTINCT CASE WHEN matches.victory_flg THEN matches.id END) AS wins, COUNT(games.id) AS game_count, SUM(CASE WHEN games.go_first THEN 1 ELSE 0 END) AS go_first_count, SUM(CASE WHEN games.go_first AND games.winning_flg THEN 1 ELSE 0 END) AS go_first_wins, SUM(CASE WHEN games.go_first = false AND games.winning_flg THEN 1 ELSE 0 END) AS go_second_wins").
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("LEFT JOIN decks ON records.deck_id = decks.id").
		Joins("LEFT JOIN games ON games.match_id = matches.id AND games.deleted_at IS NULL").
		Where("records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND records.deck_id != ''", userId)

	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	query = query.Group("records.deck_id, decks.name").Order("count DESC")

	if tx := query.Scan(&results); tx.Error != nil {
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

	if !fromDate.IsZero() {
		ignoredQuery = ignoredQuery.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		ignoredQuery = ignoredQuery.Where("records.event_date < ?", toDate)
	}

	ignoredQuery = ignoredQuery.Group("records.deck_id, decks.name").Order("ignored_count DESC")

	var ignoredResults []deckIgnoredResult
	if tx := ignoredQuery.Scan(&ignoredResults); tx.Error != nil {
		return nil, tx.Error
	}

	ignoredMap := make(map[string]int, len(ignoredResults))
	for _, r := range ignoredResults {
		ignoredMap[r.DeckId] = r.IgnoredCount
	}

	// 集計対象の記録があるデッキの deck_id 集合。集計対象外のみのデッキを
	// 後段で追加する際に、重複を避けるために使う。
	seen := make(map[string]bool, len(results))

	// スプライトはデッキごとに引くとデッキ数に比例してクエリが増える(N+1)ため、
	// 後段のループで使う分をここで1クエリにまとめて取得しておく。
	// 集計対象外のみのデッキは全期間集計のときだけ一覧に加わるので、その場合だけ含める。
	// 同じデッキが集計対象・集計対象外の両方に現れるため、IDは重複を除いて渡す。
	spriteDeckIds := make([]string, 0, len(results)+len(ignoredResults))
	spriteDeckIdSeen := make(map[string]struct{}, len(results)+len(ignoredResults))

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
	if fromDate.IsZero() && toDate.IsZero() {
		for _, r := range ignoredResults {
			appendSpriteDeckId(r.DeckId)
		}
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, spriteDeckIds)
	if err != nil {
		return nil, err
	}

	decks := []*entity.DeckUsage{}
	for _, r := range results {
		seen[r.DeckId] = true
		pokemonSprites := spritesByDeckId[r.DeckId]

		name := r.Name

		var usageRate float64
		if totalMatches > 0 {
			usageRate = float64(r.Count) / float64(totalMatches)
		}

		losses := r.Count - r.Wins
		var winRate float64
		if r.Count > 0 {
			winRate = float64(r.Wins) / float64(r.Count)
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
		deckUsage.IgnoredCount = ignoredMap[r.DeckId]
		decks = append(decks, deckUsage)
	}

	// 全期間集計(all_time)の場合のみ、集計対象外の記録しか持たないデッキも
	// 一覧に含める。デッキ一覧カードで「集計対象外の記録がN件ある」と示すためで、
	// count=0 のため使用率ランキング等では他の画面（期間指定）には現れない。
	if fromDate.IsZero() && toDate.IsZero() {
		for _, r := range ignoredResults {
			if seen[r.DeckId] {
				continue
			}

			deckUsage := entity.NewDeckUsage(
				r.DeckId, r.Name, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, spritesByDeckId[r.DeckId],
			)
			deckUsage.IgnoredCount = r.IgnoredCount
			decks = append(decks, deckUsage)
		}
	}

	return entity.NewDeckUsageStat(userId, totalMatches, decks), nil
}
