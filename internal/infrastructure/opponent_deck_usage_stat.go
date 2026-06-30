package infrastructure

import (
	"context"
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

type opponentDeckUsageResult struct {
	DeckInfo string
	Count    int
}

type latestMatchIdResult struct {
	MatchId string
}

func (i *OpponentDeckUsageStat) FindOpponentDeckUsageStat(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (*entity.OpponentDeckUsageStat, error) {
	var results []opponentDeckUsageResult

	// matches と records を結合し、対戦相手デッキ名ごとに集計する。
	// opponents_deck_info が空の対戦（未記録）は除外する。
	query := i.db.Table("matches").
		Select("matches.opponents_deck_info AS deck_info, COUNT(*) AS count").
		Joins("JOIN records ON matches.record_id = records.id").
		Where(
			"records.user_id = ? AND records.deleted_at IS NULL AND matches.deleted_at IS NULL AND matches.opponents_deck_info != ''",
			userId,
		)

	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	query = query.Group("matches.opponents_deck_info").Order("count DESC")

	if tx := query.Scan(&results); tx.Error != nil {
		return nil, tx.Error
	}

	totalMatches := 0
	for _, r := range results {
		totalMatches += r.Count
	}

	decks := []*entity.OpponentDeckUsage{}
	for _, r := range results {
		// 同じ opponents_deck_info を持つ最新マッチの ID を取得してスプライトを取得する
		var latestMatch latestMatchIdResult
		latestQuery := i.db.Table("matches").
			Select("matches.id AS match_id").
			Joins("JOIN records ON matches.record_id = records.id").
			Where(
				"records.user_id = ? AND records.deleted_at IS NULL AND matches.deleted_at IS NULL AND matches.opponents_deck_info = ?",
				userId, r.DeckInfo,
			)

		if !fromDate.IsZero() {
			latestQuery = latestQuery.Where("records.event_date >= ?", fromDate)
		}
		if !toDate.IsZero() {
			latestQuery = latestQuery.Where("records.event_date < ?", toDate)
		}

		latestQuery = latestQuery.Order("records.event_date DESC").Limit(1)

		if tx := latestQuery.Scan(&latestMatch); tx.Error != nil {
			return nil, tx.Error
		}

		var pokemonSprites []*entity.PokemonSprite
		if latestMatch.MatchId != "" {
			var matchPokemonSpriteModels []*model.MatchPokemonSprite
			if tx := i.db.Where("match_id = ?", latestMatch.MatchId).Order("position ASC").Find(&matchPokemonSpriteModels); tx.Error != nil {
				return nil, tx.Error
			}
			for _, m := range matchPokemonSpriteModels {
				pokemonSprites = append(pokemonSprites, entity.NewPokemonSprite(m.PokemonSpriteId))
			}
		}

		var usageRate float64
		if totalMatches > 0 {
			usageRate = float64(r.Count) / float64(totalMatches)
		}

		decks = append(decks, entity.NewOpponentDeckUsage(r.DeckInfo, r.Count, usageRate, pokemonSprites))
	}

	return entity.NewOpponentDeckUsageStat(userId, totalMatches, decks), nil
}
