package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
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
	Name   string
	Count  int
	Wins   int
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
	query := i.db.Table("matches").
		Select("records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(*) AS count, SUM(CASE WHEN matches.victory_flg THEN 1 ELSE 0 END) AS wins").
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("LEFT JOIN decks ON records.deck_id = decks.id").
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

	decks := []*entity.DeckUsage{}
	for _, r := range results {
		// デッキに設定されたポケモンスプライトを取得する
		var deckPokemonSpriteModels []*model.DeckPokemonSprite
		if tx := i.db.Where("deck_id = ?", r.DeckId).Order("position ASC").Find(&deckPokemonSpriteModels); tx.Error != nil {
			return nil, tx.Error
		}

		var pokemonSprites []*entity.PokemonSprite
		for _, m := range deckPokemonSpriteModels {
			pokemonSprites = append(pokemonSprites, entity.NewPokemonSprite(m.PokemonSpriteId))
		}

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

		decks = append(decks, entity.NewDeckUsage(r.DeckId, name, r.Count, usageRate, r.Wins, losses, winRate, pokemonSprites))
	}

	return entity.NewDeckUsageStat(userId, totalMatches, decks), nil
}
