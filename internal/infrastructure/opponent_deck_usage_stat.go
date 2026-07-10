package infrastructure

import (
	"context"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type OpponentDeckUsageStat struct {
	db *gorm.DB
}

func NewOpponentDeckUsageStat(
	db *gorm.DB,
) repository.OpponentDeckUsageStatInterface {
	return &OpponentDeckUsageStat{db}
}

type opponentMatchResult struct {
	MatchId    string
	DeckInfo   string
	VictoryFlg bool
}

type opponentDeckGroup struct {
	deckInfo  string
	spriteIds []string
	count     int
	wins      int
}

func (i *OpponentDeckUsageStat) FindOpponentDeckUsageStat(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
	deckId string,
) (*entity.OpponentDeckUsageStat, error) {
	var rows []opponentMatchResult

	// matches と records を結合し、対戦相手デッキ名が記録されている対戦を全件取得する。
	// opponents_deck_info が空の対戦（未記録）は除外する。
	query := i.db.Table("matches").
		Select("matches.id AS match_id, matches.opponents_deck_info AS deck_info, matches.victory_flg AS victory_flg").
		Joins("JOIN records ON matches.record_id = records.id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND matches.opponents_deck_info != ''",
			userId,
		)

	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}
	if deckId != "" {
		// 「自分のデッキ」セレクタは records.deck_id を基準に選択肢を作っている（deck_usage_stat.go参照）。
		// matches.deck_id はマッチ作成時点の値がコピーされたまま更新されないため、
		// 記録後にデッキを変更した場合にズレて対戦相手デッキが見つからなくなる。
		// そのため records.deck_id で絞り込み、選択肢と実データの基準を一致させる。
		query = query.Where("records.deck_id = ?", deckId)
	}

	query = query.Order("records.event_date ASC")

	if tx := query.Scan(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	if len(rows) == 0 {
		return entity.NewOpponentDeckUsageStat(userId, 0, []*entity.OpponentDeckUsage{}), nil
	}

	matchIds := make([]string, 0, len(rows))
	for _, r := range rows {
		matchIds = append(matchIds, r.MatchId)
	}

	var spriteModels []*model.MatchPokemonSprite
	if tx := i.db.Where("match_id IN ?", matchIds).Order("position ASC").Find(&spriteModels); tx.Error != nil {
		return nil, tx.Error
	}

	spritesByMatch := make(map[string][]string, len(rows))
	for _, s := range spriteModels {
		spritesByMatch[s.MatchId] = append(spritesByMatch[s.MatchId], s.PokemonSpriteId)
	}

	// デッキ名が同じでも、対戦相手デッキのスプライト構成が異なれば別デッキとして扱う。
	// そのためグループキーはデッキ名とスプライト構成の組み合わせにする。
	groups := make(map[string]*opponentDeckGroup, len(rows))
	order := make([]string, 0, len(rows))
	totalMatches := 0

	for _, r := range rows {
		spriteIds := spritesByMatch[r.MatchId]
		key := r.DeckInfo + "|" + strings.Join(spriteIds, ",")

		g, ok := groups[key]
		if !ok {
			g = &opponentDeckGroup{deckInfo: r.DeckInfo, spriteIds: spriteIds}
			groups[key] = g
			order = append(order, key)
		}

		g.count++
		if r.VictoryFlg {
			g.wins++
		}
		totalMatches++
	}

	// 出現件数の降順（同数の場合は初出順=対戦日の古い順を維持）にソートする
	sort.SliceStable(order, func(a, b int) bool {
		return groups[order[a]].count > groups[order[b]].count
	})

	decks := make([]*entity.OpponentDeckUsage, 0, len(order))
	for _, key := range order {
		g := groups[key]

		var usageRate float64
		if totalMatches > 0 {
			usageRate = float64(g.count) / float64(totalMatches)
		}

		losses := g.count - g.wins
		var winRate float64
		if g.count > 0 {
			winRate = float64(g.wins) / float64(g.count)
		}

		pokemonSprites := make([]*entity.PokemonSprite, 0, len(g.spriteIds))
		for _, spriteId := range g.spriteIds {
			pokemonSprites = append(pokemonSprites, entity.NewPokemonSprite(spriteId))
		}

		decks = append(decks, entity.NewOpponentDeckUsage(g.deckInfo, g.count, usageRate, g.wins, losses, winRate, pokemonSprites))
	}

	return entity.NewOpponentDeckUsageStat(userId, totalMatches, decks), nil
}
