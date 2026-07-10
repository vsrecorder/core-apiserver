package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UserStatRecent struct {
	db *gorm.DB
}

func NewUserStatRecent(
	db *gorm.DB,
) repository.UserStatRecentInterface {
	return &UserStatRecent{db}
}

type recentMatchResult struct {
	MatchId           string
	EventDate         time.Time
	DeckId            string
	OpponentsDeckInfo string
	VictoryFlg        bool
}

func (i *UserStatRecent) FindRecentMatches(
	ctx context.Context,
	userId string,
	count int,
	deckId string,
) ([]*entity.RecentMatch, error) {
	var results []recentMatchResult

	query := i.db.Table("matches").
		Select(
			"matches.id AS match_id, "+
				"records.event_date AS event_date, "+
				"matches.deck_id AS deck_id, "+
				"matches.opponents_deck_info AS opponents_deck_info, "+
				"matches.victory_flg AS victory_flg",
		).
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false").
		Where("matches.user_id = ? AND matches.deleted_at IS NULL", userId)

	if deckId != "" {
		query = query.Where("matches.deck_id = ?", deckId)
	}

	query = query.
		Order("records.event_date DESC, matches.created_at DESC").
		Limit(count)

	if tx := query.Scan(&results); tx.Error != nil {
		return nil, tx.Error
	}

	if len(results) == 0 {
		return []*entity.RecentMatch{}, nil
	}

	matchIds := make([]string, 0, len(results))
	for _, r := range results {
		matchIds = append(matchIds, r.MatchId)
	}

	var spriteModels []*model.MatchPokemonSprite
	if tx := i.db.Where("match_id IN ?", matchIds).Order("position ASC").Find(&spriteModels); tx.Error != nil {
		return nil, tx.Error
	}

	spritesByMatch := make(map[string][]*entity.PokemonSprite, len(results))
	for _, s := range spriteModels {
		spritesByMatch[s.MatchId] = append(spritesByMatch[s.MatchId], entity.NewPokemonSprite(s.PokemonSpriteId))
	}

	// 対戦日の古い順に並び替える（取得はDESC・LIMITのため逆順になっている）
	matches := make([]*entity.RecentMatch, 0, len(results))
	for idx := len(results) - 1; idx >= 0; idx-- {
		r := results[idx]
		matches = append(matches, entity.NewRecentMatch(
			0,
			r.EventDate,
			r.DeckId,
			r.OpponentsDeckInfo,
			r.VictoryFlg,
			0,
			"",
			"",
			spritesByMatch[r.MatchId],
		))
	}

	return matches, nil
}
